// Package mockportal is a local stand-in for the BITS FortiGate captive portal
// (protocol/PROTOCOL.md section 11). It is test infrastructure: Go tests start it in-process,
// the Kotlin tests run it as the cmd/mockportal binary.
//
// Two servers:
//   - Probe (plain HTTP): "/" is the neverssl stand-in, "/generate_204" the Android probe.
//     Unauthenticated, both send the client to the portal in the configured style.
//     "/__mock/..." is the control API.
//   - Portal (HTTPS, own test CA, certificate valid for 127.0.0.1): "/fgtauth?<magic>",
//     "POST /", "/keepalive?<token>", "/logout?<token>".
//
// The mock models a single client: one "authenticated" flag, like a FortiGate keyed on the
// caller's IP.
package mockportal

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"

	"autowifi/internal/config"
)

// Default test credentials.
const (
	DefaultUsername = "f20250001"
	DefaultPassword = "correct-horse"
)

// Probe styles.
const (
	StyleRedirect = "redirect" // 303 Location: https://host:port/fgtauth?magic
	StylePage     = "page"     // 200 page with window.location="https://host:port/fgtauth?magic"
	StyleJS       = "js"       // 200 page with the \/-escaped URL inside a JS string
)

// Options configure Start. Zero values mean the defaults above. ProbeAddr/PortalAddr ("" =
// random 127.0.0.1 port) are used by cmd/mockportal.
type Options struct {
	Username   string
	Password   string
	ProbeAddr  string
	PortalAddr string
}

// Fields are the input names the login form uses (and the POST handler reads).
type Fields struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Magic    string `json:"magic"`
	Redirect string `json:"redirect"`
}

// State is the counters exposed by Snapshot and GET /__mock/state.
type State struct {
	Authenticated bool   `json:"authenticated"`
	LoginsOK      int    `json:"logins_ok"`
	LoginsFailed  int    `json:"logins_failed"`
	Keepalives    int    `json:"keepalives"`
	Logouts       int    `json:"logouts"`
	ProbeHits     int    `json:"probe_hits"`
	MagicsIssued  int    `json:"magics_issued"`
	Style         string `json:"style"`
	FailNext      int    `json:"fail_next"`
	Fields        Fields `json:"fields"`
}

// Server is a running mock portal.
type Server struct {
	Probe  *httptest.Server
	Portal *httptest.Server

	user, pass string

	mu     sync.Mutex
	st     State
	magics map[string]bool // issued and not yet used
	token  string          // current session token, "" when not authenticated
}

// Start starts both servers. Call Close when done.
func Start(opts Options) *Server {
	s := &Server{user: opts.Username, pass: opts.Password}
	if s.user == "" {
		s.user = DefaultUsername
	}
	if s.pass == "" {
		s.pass = DefaultPassword
	}
	s.reset()

	s.Portal = httptest.NewUnstartedServer(http.HandlerFunc(s.portalHandler))
	if opts.PortalAddr != "" {
		s.Portal.Listener = mustListen(opts.PortalAddr)
	}
	s.Portal.StartTLS()

	s.Probe = httptest.NewUnstartedServer(http.HandlerFunc(s.probeHandler))
	if opts.ProbeAddr != "" {
		s.Probe.Listener = mustListen(opts.ProbeAddr)
	}
	s.Probe.Start()
	return s
}

func mustListen(addr string) net.Listener {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("mockportal: listen %s: %v", addr, err))
	}
	return l
}

// Close stops both servers.
func (s *Server) Close() {
	s.Probe.Close()
	s.Portal.Close()
}

// PortalHostPort returns the portal host ("127.0.0.1") and port.
func (s *Server) PortalHostPort() (string, int) {
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(s.Portal.URL, "https://"))
	p, _ := strconv.Atoi(port)
	return host, p
}

// PortalBase returns "https://127.0.0.1:<port>".
func (s *Server) PortalBase() string {
	h, p := s.PortalHostPort()
	return fmt.Sprintf("https://%s:%d", h, p)
}

// Config returns a PortalConfig pointing at this mock with the default field names.
func (s *Server) Config() config.PortalConfig {
	h, p := s.PortalHostPort()
	return config.PortalConfig{
		CheckURL:         s.Probe.URL + "/",
		FallbackCheckURL: s.Probe.URL + "/generate_204",
		PortalHost:       h,
		PortalPort:       p,
		LoginPath:        "/",
		UsernameField:    "username",
		PasswordField:    "password",
		MagicField:       "magic",
		RedirectField:    "4Tredir",
	}
}

// CertPool returns a pool holding the portal's certificate, for the portal client's RootCAs.
func (s *Server) CertPool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(s.Portal.Certificate())
	return p
}

// CertPEM returns the portal certificate in PEM form (cmd/mockportal writes it for Kotlin).
func (s *Server) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.Portal.Certificate().Raw})
}

// TLSClientConfig returns a client TLS config trusting only the mock CA.
func (s *Server) TLSClientConfig() *tls.Config {
	return &tls.Config{RootCAs: s.CertPool(), MinVersion: tls.VersionTLS12}
}

// --- controls (also reachable over HTTP under /__mock/) ---

// Reset restores defaults: not authenticated, counters zero, style redirect, default fields.
func (s *Server) Reset() { s.mu.Lock(); defer s.mu.Unlock(); s.reset() }

func (s *Server) reset() {
	s.st = State{Style: StyleRedirect, Fields: Fields{"username", "password", "magic", "4Tredir"}}
	s.magics = map[string]bool{}
	s.token = ""
}

// Expire ends the session (as the firewall's idle timeout would).
func (s *Server) Expire() { s.mu.Lock(); defer s.mu.Unlock(); s.st.Authenticated = false; s.token = "" }

// SetAuthenticated marks the client logged in (or out) without a login.
func (s *Server) SetAuthenticated(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.st.Authenticated = v
	if v {
		s.token = randHex(8)
	} else {
		s.token = ""
	}
}

// FailNext makes the next n portal requests answer 500.
func (s *Server) FailNext(n int) { s.mu.Lock(); defer s.mu.Unlock(); s.st.FailNext = n }

// SetStyle sets the probe style (StyleRedirect, StylePage, StyleJS).
func (s *Server) SetStyle(style string) { s.mu.Lock(); defer s.mu.Unlock(); s.st.Style = style }

// SetFields renames form inputs; empty values keep the current name.
func (s *Server) SetFields(f Fields) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Username != "" {
		s.st.Fields.Username = f.Username
	}
	if f.Password != "" {
		s.st.Fields.Password = f.Password
	}
	if f.Magic != "" {
		s.st.Fields.Magic = f.Magic
	}
	if f.Redirect != "" {
		s.st.Fields.Redirect = f.Redirect
	}
}

// Snapshot returns the current counters.
func (s *Server) Snapshot() State { s.mu.Lock(); defer s.mu.Unlock(); return s.st }

// Token returns the current session token ("" when logged out).
func (s *Server) Token() string { s.mu.Lock(); defer s.mu.Unlock(); return s.token }

// --- handlers ---
//
// The "nosemgrep" markers below silence the Fprintf-to-ResponseWriter XSS rule: this is test
// infrastructure, and every interpolated value is a server-generated hex token, the mock's own
// base URL, or html.EscapeString output.

func (s *Server) probeHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/__mock/") {
		s.control(w, r)
		return
	}
	s.mu.Lock()
	s.st.ProbeHits++
	authed, style := s.st.Authenticated, s.st.Style
	var magic string
	if !authed {
		magic = s.issueMagic()
	}
	s.mu.Unlock()

	if authed {
		if r.URL.Path == "/generate_204" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><head><title>NeverSSL - Connecting ...</title></head><body>NeverSSL</body></html>") // nosemgrep
		return
	}
	target := s.PortalBase() + "/fgtauth?" + magic
	switch style {
	case StylePage:
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><script language=\"JavaScript\">window.location=\"%s\";</script></body></html>", target) // nosemgrep
	case StyleJS:
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<script>var u = \"%s\"; window.location = u;</script>", strings.ReplaceAll(target, "/", `\/`)) // nosemgrep
	default:
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}

func (s *Server) control(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch strings.TrimPrefix(r.URL.Path, "/__mock/") {
	case "reset":
		s.Reset()
	case "expire":
		s.Expire()
	case "auth":
		s.SetAuthenticated(q.Get("value") == "1" || q.Get("value") == "true")
	case "fail":
		n, err := strconv.Atoi(q.Get("n"))
		if err != nil || n < 0 {
			http.Error(w, "n must be a non-negative integer", http.StatusBadRequest)
			return
		}
		s.FailNext(n)
	case "style":
		st := q.Get("probe")
		if st != StyleRedirect && st != StylePage && st != StyleJS {
			http.Error(w, "probe must be redirect, page or js", http.StatusBadRequest)
			return
		}
		s.SetStyle(st)
	case "fields":
		s.SetFields(Fields{q.Get("username"), q.Get("password"), q.Get("magic"), q.Get("redirect")})
	case "state":
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Snapshot())
}

func (s *Server) portalHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.st.FailNext > 0 {
		s.st.FailNext--
		s.mu.Unlock()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	s.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/fgtauth":
		s.loginPage(w, r.URL.RawQuery, "")
	case r.Method == http.MethodPost && r.URL.Path == "/":
		s.login(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/keepalive":
		s.keepalive(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/logout":
		s.logout(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) loginPage(w http.ResponseWriter, magic, banner string) {
	s.mu.Lock()
	f := s.st.Fields
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html")
	// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
	fmt.Fprintf(w, `<html><head><title>Firewall Authentication</title></head><body>%s
<form action="/" method="post">
<input type="hidden" name="%s" value="http://neverssl.com/">
<input type="hidden" name="%s" value="%s">
<h2>Authentication Required</h2>
<input type="text" name="%s" id="ft_un">
<input type="password" name="%s" id="ft_pd">
<input type="submit" value="Continue">
</form></body></html>`,
		banner, html.EscapeString(f.Redirect), html.EscapeString(f.Magic), html.EscapeString(magic),
		html.EscapeString(f.Username), html.EscapeString(f.Password))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	f := s.st.Fields
	magic := r.PostForm.Get(f.Magic)
	if !s.magics[magic] {
		s.mu.Unlock()
		http.Error(w, "invalid or reused magic", http.StatusBadRequest)
		return
	}
	delete(s.magics, magic) // one POST per magic
	ok := r.PostForm.Get(f.Username) == s.user && r.PostForm.Get(f.Password) == s.pass
	var token, next string
	if ok {
		s.st.LoginsOK++
		s.st.Authenticated = true
		s.token = randHex(8)
		token = s.token
	} else {
		s.st.LoginsFailed++
		next = s.issueMagic()
	}
	s.mu.Unlock()

	if !ok {
		s.loginPage(w, next, "<h2>Firewall Authentication Failed</h2><p>Please try again.</p>")
		return
	}
	base := s.PortalBase()
	w.Header().Set("Content-Type", "text/html")
	// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
	fmt.Fprintf(w, `<html><body><script language="JavaScript">window.location="%s/keepalive?%s";</script>
<a href="%s/logout?%s">Logout</a></body></html>`, base, token, base, token)
}

func (s *Server) keepalive(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	alive := s.st.Authenticated && s.token != "" && r.URL.RawQuery == s.token
	var magic string
	if alive {
		s.st.Keepalives++
	} else {
		magic = s.issueMagic()
	}
	s.mu.Unlock()
	if !alive {
		http.Redirect(w, r, s.PortalBase()+"/fgtauth?"+magic, http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<html><h2>Authentication Keepalive</h2><p>This browser window is used to keep your authentication session active.</p></html>") // nosemgrep
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.token != "" && r.URL.RawQuery == s.token {
		s.st.Authenticated = false
		s.token = ""
		s.st.Logouts++
	}
	s.mu.Unlock()
	fmt.Fprint(w, "<html><h2>You have been logged out.</h2></html>") // nosemgrep
}

// issueMagic must be called with s.mu held.
func (s *Server) issueMagic() string {
	m := randHex(8)
	s.magics[m] = true
	s.st.MagicsIssued++
	return m
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
