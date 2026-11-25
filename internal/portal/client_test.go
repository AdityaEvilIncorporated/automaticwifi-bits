package portal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"autowifi/internal/config"
	"autowifi/internal/creds"
	"autowifi/internal/mockportal"
)

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func testClient(s *mockportal.Server) *Client {
	return &Client{
		Cfg:      s.Config(),
		Probe:    &http.Client{Timeout: 5 * time.Second, CheckRedirect: noRedirect},
		Portal:   &http.Client{Timeout: 5 * time.Second, CheckRedirect: noRedirect, Transport: &http.Transport{TLSClientConfig: s.TLSClientConfig()}},
		Resolver: net.DefaultResolver,
		Roots:    s.CertPool(),
	}
}

func goodCreds() (creds.Credentials, error) {
	return creds.Credentials{Username: mockportal.DefaultUsername, Password: mockportal.DefaultPassword}, nil
}

type failResolver struct{ calls int }

func (f *failResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	f.calls++
	return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
}

type okResolver struct{}

func (okResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return []string{"10.1.0.1"}, nil
}

func TestScenariosSharedCases(t *testing.T) {
	var f struct {
		Cases []struct {
			Name string `json:"name"`
			Mock struct {
				Style         string            `json:"style"`
				Authenticated bool              `json:"authenticated"`
				Fail          int               `json:"fail"`
				Fields        mockportal.Fields `json:"fields"`
			} `json:"mock"`
			Credentials *creds.Credentials `json:"credentials"`
			Expect      struct {
				Outcome  Outcome        `json:"outcome"`
				Counters map[string]int `json:"counters"`
			} `json:"expect"`
		} `json:"cases"`
	}
	loadCases(t, "scenarios.json", &f)
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			s.Reset()
			if c.Mock.Style != "" {
				s.SetStyle(c.Mock.Style)
			}
			s.SetAuthenticated(c.Mock.Authenticated)
			s.FailNext(c.Mock.Fail)
			s.SetFields(c.Mock.Fields)
			get := func() (creds.Credentials, error) {
				if c.Credentials == nil {
					return creds.Credentials{}, creds.ErrNotFound
				}
				return *c.Credentials, nil
			}
			r := testClient(s).CheckOnce(context.Background(), get)
			if r.Outcome != c.Expect.Outcome {
				t.Fatalf("outcome = %s (err %v), want %s", r.Outcome, r.Err, c.Expect.Outcome)
			}
			snap := s.Snapshot()
			b, _ := json.Marshal(snap)
			var counters map[string]any
			json.Unmarshal(b, &counters)
			for k, want := range c.Expect.Counters {
				if got := int(counters[k].(float64)); got != want {
					t.Errorf("counter %s = %d, want %d", k, got, want)
				}
			}
			if r.Outcome == LoggedIn {
				base := s.PortalBase()
				if r.KeepaliveURL != base+"/keepalive?"+s.Token() || r.LogoutURL != base+"/logout?"+s.Token() {
					t.Fatalf("session URLs %q %q, token %q", r.KeepaliveURL, r.LogoutURL, s.Token())
				}
				if r.Magic == "" {
					t.Fatal("Magic empty on LoggedIn")
				}
			} else if r.KeepaliveURL != "" || r.LogoutURL != "" {
				t.Fatalf("session URLs set on %s", r.Outcome)
			}
		})
	}
}

func TestCheckOnceOffCampus(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	c.Cfg.PortalHost = "fw.bits-pilani.invalid"
	res := &failResolver{}
	c.Resolver = res
	called := false
	r := c.CheckOnce(context.Background(), func() (creds.Credentials, error) { called = true; return goodCreds() })
	if r.Outcome != OffCampus || res.calls != 1 {
		t.Fatalf("outcome %s, resolver calls %d", r.Outcome, res.calls)
	}
	if called || s.Snapshot().ProbeHits != 0 {
		t.Fatal("off campus must not probe or read credentials")
	}
}

func TestOnCampus(t *testing.T) {
	cfg := config.Default()
	res := &failResolver{}
	c := &Client{Cfg: cfg, Resolver: res}
	if c.OnCampus(context.Background()) {
		t.Fatal("DNS failure must be off campus")
	}
	c.Resolver = okResolver{}
	if !c.OnCampus(context.Background()) {
		t.Fatal("resolved host must be on campus")
	}
	cfg.PortalHost = "127.0.0.1"
	res2 := &failResolver{}
	c = &Client{Cfg: cfg, Resolver: res2}
	if !c.OnCampus(context.Background()) || res2.calls != 0 {
		t.Fatal("IP literal counts as resolved without DNS")
	}
}

func TestCredentialsReadOnlyWhenCaptive(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	s.SetAuthenticated(true)
	called := false
	r := testClient(s).CheckOnce(context.Background(), func() (creds.Credentials, error) { called = true; return goodCreds() })
	if r.Outcome != Online || called {
		t.Fatalf("outcome %s, creds read %v", r.Outcome, called)
	}
}

func TestCredsGetOtherError(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	r := testClient(s).CheckOnce(context.Background(), func() (creds.Credentials, error) {
		return creds.Credentials{}, errors.New("keychain locked")
	})
	if r.Outcome != PortalError || r.Err == nil {
		t.Fatalf("outcome %s err %v", r.Outcome, r.Err)
	}
}

func TestProbeFallback(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	c.Cfg.CheckURL = deadURL + "/"
	p, err := c.ProbeOnce(context.Background())
	if err != nil || p.Kind != ProbeCaptive {
		t.Fatalf("fallback probe = %+v, %v", p, err)
	}
	c.Cfg.FallbackCheckURL = deadURL + "/generate_204"
	if _, err := c.ProbeOnce(context.Background()); err == nil {
		t.Fatal("both probes dead: want error")
	}
	if r := c.CheckOnce(context.Background(), goodCreds); r.Outcome != PortalError {
		t.Fatalf("both probes dead: outcome %s", r.Outcome)
	}
}

func TestProbeHTTPStatusIsNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	c := &Client{Cfg: config.Default(), Probe: &http.Client{CheckRedirect: noRedirect}}
	c.Cfg.CheckURL = srv.URL
	p, err := c.ProbeOnce(context.Background())
	if err != nil || p.Kind != ProbeForeign {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestProbeReadsAtMostMaxBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("a", MaxBody)))
		w.Write([]byte(`window.location="https://fw.bits-pilani.ac.in:8090/fgtauth?abc"`))
	}))
	defer srv.Close()
	c := &Client{Cfg: config.Default(), Probe: &http.Client{CheckRedirect: noRedirect}}
	c.Cfg.CheckURL = srv.URL
	p, err := c.ProbeOnce(context.Background())
	if err != nil || p.Kind != ProbeOnline {
		t.Fatalf("text beyond MaxBody must be ignored: %+v %v", p, err)
	}
}

func TestLoginRefusesForeignMagicURL(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	for _, u := range []string{
		"https://evil.example:" + strconv.Itoa(c.Cfg.PortalPort) + "/fgtauth?abc",
		"http://127.0.0.1:" + strconv.Itoa(c.Cfg.PortalPort) + "/fgtauth?abc",
		"https://127.0.0.1:1/fgtauth?abc",
	} {
		r := c.Login(context.Background(), ProbeResult{Kind: ProbeCaptive, Magic: "abc", MagicURL: u}, creds.Credentials{Username: "u", Password: "p"})
		if r.Outcome != Foreign {
			t.Errorf("%s: outcome %s, want foreign", u, r.Outcome)
		}
	}
	if st := s.Snapshot(); st.LoginsOK+st.LoginsFailed != 0 {
		t.Fatal("credentials were sent")
	}
}

// fakePortal serves a login page and records the POST.
func fakePortal(t *testing.T, post func(w http.ResponseWriter, body string)) (*httptest.Server, *Client) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/fgtauth":
			io.WriteString(w, `<form action="/" method="post"><input type=hidden name=4Tredir value="http://neverssl.com/?a=1&b=2"><input type=hidden name=magic value="`+r.URL.RawQuery+`"><input name=username><input type=password name=password></form>`)
		case r.Method == "POST" && r.URL.Path == "/":
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("content type %q", ct)
			}
			b, _ := io.ReadAll(r.Body)
			post(w, string(b))
		default:
			http.NotFound(w, r)
		}
	}))
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	cfg := config.Default()
	cfg.PortalHost = host
	cfg.PortalPort, _ = strconv.Atoi(port)
	c := &Client{Cfg: cfg, Portal: srv.Client(), Resolver: okResolver{}}
	c.Portal.CheckRedirect = noRedirect
	return srv, c
}

func TestLoginPostBodyOrderAndEncoding(t *testing.T) {
	var got string
	srv, c := fakePortal(t, func(w http.ResponseWriter, body string) {
		got = body
		io.WriteString(w, "ok")
	})
	defer srv.Close()
	magicURL := c.Cfg.PortalBase() + "/fgtauth?m123"
	c.Login(context.Background(), ProbeResult{Kind: ProbeCaptive, Magic: "m123", MagicURL: magicURL}, creds.Credentials{Username: "f2025 01", Password: "p&ss=wörd"})
	want := "username=f2025+01&password=p%26ss%3Dw%C3%B6rd&magic=m123&4Tredir=http%3A%2F%2Fneverssl.com%2F%3Fa%3D1%26b%3D2"
	if got != want {
		t.Fatalf("POST body\n got %s\nwant %s", got, want)
	}
}

func TestLoginErrorNeverContainsPassword(t *testing.T) {
	srv, c := fakePortal(t, func(w http.ResponseWriter, body string) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	})
	defer srv.Close()
	pw := "S3cret-Pass-123"
	r := c.Login(context.Background(), ProbeResult{Kind: ProbeCaptive, Magic: "m1", MagicURL: c.Cfg.PortalBase() + "/fgtauth?m1"}, creds.Credentials{Username: "f20250001", Password: pw})
	if r.Outcome != PortalError || r.Err == nil {
		t.Fatalf("outcome %s err %v", r.Outcome, r.Err)
	}
	if strings.Contains(r.Err.Error(), pw) || strings.Contains(r.Err.Error(), "f20250001") {
		t.Fatalf("error leaks credentials: %v", r.Err)
	}
}

func TestLoginPageNon2xx(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	p, err := c.ProbeOnce(context.Background())
	if err != nil || p.Kind != ProbeCaptive {
		t.Fatal(p, err)
	}
	s.FailNext(1)
	if r := c.Login(context.Background(), p, creds.Credentials{Username: "a", Password: "b"}); r.Outcome != PortalError {
		t.Fatalf("outcome %s", r.Outcome)
	}
}

func TestKeepaliveAndLogout(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	ctx := context.Background()
	r := c.CheckOnce(ctx, goodCreds)
	if r.Outcome != LoggedIn {
		t.Fatalf("login: %s %v", r.Outcome, r.Err)
	}
	if !c.Keepalive(ctx, r.KeepaliveURL) || s.Snapshot().Keepalives != 1 {
		t.Fatal("keepalive should be alive")
	}
	if c.Keepalive(ctx, "https://evil.example:1/keepalive?"+s.Token()) {
		t.Fatal("foreign keepalive URL must be false")
	}
	if s.Snapshot().Keepalives != 1 {
		t.Fatal("foreign keepalive URL was fetched")
	}
	if err := c.Logout(ctx, r.LogoutURL); err != nil {
		t.Fatal(err)
	}
	if st := s.Snapshot(); st.Authenticated || st.Logouts != 1 {
		t.Fatalf("after logout %+v", st)
	}
	if c.Keepalive(ctx, r.KeepaliveURL) {
		t.Fatal("keepalive after logout must be false")
	}
	if r := c.CheckOnce(ctx, goodCreds); r.Outcome != LoggedIn {
		t.Fatalf("relogin: %s", r.Outcome)
	}
	s.Expire()
	if c.Keepalive(ctx, r.KeepaliveURL) {
		t.Fatal("keepalive after expire must be false")
	}
}

func TestMagicIsSingleUse(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	ctx := context.Background()
	p, _ := c.ProbeOnce(ctx)
	if r := c.Login(ctx, p, creds.Credentials{Username: "x", Password: "wrong"}); r.Outcome != Rejected {
		t.Fatalf("first: %s", r.Outcome)
	}
	// a fresh check gets a fresh magic and succeeds
	if r := c.CheckOnce(ctx, goodCreds); r.Outcome != LoggedIn {
		t.Fatalf("second: %s %v", r.Outcome, r.Err)
	}
}
