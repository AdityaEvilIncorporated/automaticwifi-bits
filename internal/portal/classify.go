// Package portal implements the FortiGate protocol in protocol/PROTOCOL.md: pure classifiers
// (this file), the network client (client.go) and the capture diagnostic (capture.go).
package portal

import (
	"net/url"
	"strconv"
	"strings"

	"autowifi/internal/config"
)

// Outcome is the result of one check (PROTOCOL.md section 2). String values are the
// protocol names and are used in logs and in the shared JSON cases.
type Outcome string

const (
	OffCampus     Outcome = "off_campus"
	Online        Outcome = "online"
	Foreign       Outcome = "foreign"
	LoggedIn      Outcome = "logged_in"
	Rejected      Outcome = "rejected"
	FormChanged   Outcome = "form_changed"
	PortalError   Outcome = "portal_error"
	NoCredentials Outcome = "no_credentials"
)

// ProbeKind classifies a probe reply (PROTOCOL.md section 3).
type ProbeKind string

const (
	ProbeOnline    ProbeKind = "online"
	ProbeForeign   ProbeKind = "foreign"
	ProbeCaptive   ProbeKind = "captive"
	ProbeMalformed ProbeKind = "malformed"
)

// ProbeResult is the classification of a probe reply. Magic and MagicURL are set only for
// ProbeCaptive; MagicURL is always rebuilt as cfg.PortalBase()+"/fgtauth?"+Magic.
type ProbeResult struct {
	Kind     ProbeKind
	Magic    string
	MagicURL string
}

// ClassifyProbe implements PROTOCOL.md section 3 exactly. It must pass
// protocol/testcases/probe.json.
func ClassifyProbe(cfg config.PortalConfig, status int, location string, body []byte) ProbeResult {
	if status >= 300 && status <= 399 && location != "" {
		if OnPortal(cfg, location) {
			if u, err := url.Parse(location); err == nil && u.Path == "/fgtauth" {
				q := u.RawQuery
				if q != "" && isAllMagic(q) {
					return ProbeResult{
						Kind:     ProbeCaptive,
						Magic:    q,
						MagicURL: cfg.PortalBase() + "/fgtauth?" + q,
					}
				}
				return ProbeResult{Kind: ProbeMalformed}
			}
		}
		return ProbeResult{Kind: ProbeForeign}
	}

	bodyStr := strings.ToValidUTF8(string(body), "\uFFFD")
	if magic, found := findFgtauthMagic(cfg, bodyStr); found {
		if magic == "" {
			return ProbeResult{Kind: ProbeMalformed}
		}
		return ProbeResult{
			Kind:     ProbeCaptive,
			Magic:    magic,
			MagicURL: cfg.PortalBase() + "/fgtauth?" + magic,
		}
	}

	if status == 200 || status == 204 {
		return ProbeResult{Kind: ProbeOnline}
	}
	return ProbeResult{Kind: ProbeForeign}
}

// LoginForm is what the login POST needs from the login page.
type LoginForm struct {
	Magic    string // from the form's magic input if non-empty, else the probe magic
	Redirect string // value of redirect_field, "" when absent
	Action   string // the form's action attribute, for capture only (never used for the POST)
}

// ErrFormChanged is returned when the page no longer has the expected form.
var ErrFormChanged = errFormChanged{}

type errFormChanged struct{}

func (errFormChanged) Error() string { return "portal form changed" }

// ParseLoginPage implements PROTOCOL.md section 4 using ParseForms. It returns ErrFormChanged
// (the exact value, compare with errors.Is) when no form has both the username and password
// fields, or that form lacks the magic field. Must pass protocol/testcases/login_page.json.
func ParseLoginPage(cfg config.PortalConfig, body []byte, probeMagic string) (LoginForm, error) {
	for _, f := range ParseForms(body) {
		if !f.Has(cfg.UsernameField) || !f.Has(cfg.PasswordField) {
			continue
		}
		magic, ok := f.Value(cfg.MagicField)
		if !ok {
			return LoginForm{}, ErrFormChanged
		}
		if magic == "" {
			magic = probeMagic
		}
		redirect, _ := f.Value(cfg.RedirectField)
		return LoginForm{
			Magic:    magic,
			Redirect: redirect,
			Action:   f.Action,
		}, nil
	}
	return LoginForm{}, ErrFormChanged
}

// LoginReply is the classification of the login POST reply (PROTOCOL.md section 5).
// KeepaliveURL and LogoutURL are set only when Outcome == LoggedIn.
type LoginReply struct {
	Outcome      Outcome
	KeepaliveURL string
	LogoutURL    string
}

// ClassifyLoginReply implements PROTOCOL.md section 5, including the host rule (keepalive or
// logout URLs not on cfg.PortalBase() are ignored). Outcome is one of LoggedIn, Rejected,
// PortalError, FormChanged. Must pass protocol/testcases/login_reply.json.
func ClassifyLoginReply(cfg config.PortalConfig, status int, location string, body []byte) LoginReply {
	bodyStr := strings.ToValidUTF8(string(body), "\uFFFD")

	keepaliveToken, hasKeepalive := findPortalToken(cfg, location, "keepalive")
	if !hasKeepalive {
		keepaliveToken, hasKeepalive = findPortalToken(cfg, bodyStr, "keepalive")
	}
	if hasKeepalive {
		logoutToken, hasLogout := findPortalToken(cfg, location, "logout")
		if !hasLogout {
			logoutToken, hasLogout = findPortalToken(cfg, bodyStr, "logout")
		}
		if !hasLogout {
			logoutToken = keepaliveToken
		}
		base := cfg.PortalBase()
		return LoginReply{
			Outcome:      LoggedIn,
			KeepaliveURL: base + "/keepalive?" + keepaliveToken,
			LogoutURL:    base + "/logout?" + logoutToken,
		}
	}

	if strings.Contains(asciiLower(bodyStr), "authentication failed") || hasPasswordField(body, cfg.PasswordField) {
		return LoginReply{Outcome: Rejected}
	}
	if status < 200 || status >= 400 {
		return LoginReply{Outcome: PortalError}
	}
	return LoginReply{Outcome: FormChanged}
}

// KeepaliveAlive implements PROTOCOL.md section 7 "Alive". Must pass
// protocol/testcases/keepalive.json.
func KeepaliveAlive(cfg config.PortalConfig, status int, body []byte) bool {
	if status < 200 || status >= 300 {
		return false
	}
	if strings.Contains(asciiLower(string(body)), "fgtauth") {
		return false
	}
	return !hasPasswordField(body, cfg.PasswordField)
}

// OnPortal reports whether rawURL is https, on cfg.PortalHost (case-insensitive) and on
// cfg.PortalPort (explicit port required, except that port 443 may be implicit). It is the
// host rule from PROTOCOL.md section 2 and guards every portal request.
func OnPortal(cfg config.PortalConfig, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return false
	}
	if !strings.EqualFold(u.Hostname(), cfg.PortalHost) {
		return false
	}
	portStr := u.Port()
	if portStr == "" {
		return cfg.PortalPort == 443
	}
	port, err := strconv.Atoi(portStr)
	return err == nil && port == cfg.PortalPort
}

func isMagicByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isAllMagic(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isMagicByte(s[i]) {
			return false
		}
	}
	return true
}

func magicPrefix(s string) string {
	i := 0
	for i < len(s) && isMagicByte(s[i]) {
		i++
	}
	return s[:i]
}

// asciiLower lowercases only ASCII A-Z so byte offsets remain identical to the input.
func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}

func portalNeedles(cfg config.PortalConfig, path string) (plain, escaped string) {
	base := cfg.PortalBase()
	plain = base + "/" + path + "?"
	escaped = strings.ReplaceAll(base, "/", `\/`) + `\/` + path + "?"
	return plain, escaped
}

// findFgtauthMagic searches text for the first occurrence of the portal's fgtauth URL
// (plain or \/-escaped, host matched case-insensitively) and returns the magic run after it.
func findFgtauthMagic(cfg config.PortalConfig, text string) (magic string, found bool) {
	plain, escaped := portalNeedles(cfg, "fgtauth")
	lower := asciiLower(text)

	bestIdx := -1
	bestLen := 0
	for _, needle := range []string{plain, escaped} {
		if idx := strings.Index(lower, needle); idx != -1 && (bestIdx == -1 || idx < bestIdx) {
			bestIdx = idx
			bestLen = len(needle)
		}
	}
	if bestIdx == -1 {
		return "", false
	}
	return magicPrefix(text[bestIdx+bestLen:]), true
}

// findPortalToken searches text for the first occurrence of {portal base}/{path}?{token}
// (plain or \/-escaped, host matched case-insensitively) with a non-empty [0-9A-Za-z] token.
func findPortalToken(cfg config.PortalConfig, text, path string) (token string, found bool) {
	if text == "" {
		return "", false
	}
	plain, escaped := portalNeedles(cfg, path)
	lower := asciiLower(text)

	bestIdx := -1
	bestToken := ""
	for _, needle := range []string{plain, escaped} {
		offset := 0
		for {
			idx := strings.Index(lower[offset:], needle)
			if idx == -1 {
				break
			}
			absIdx := offset + idx
			tok := magicPrefix(text[absIdx+len(needle):])
			if tok != "" {
				if bestIdx == -1 || absIdx < bestIdx {
					bestIdx = absIdx
					bestToken = tok
				}
				break
			}
			offset = absIdx + len(needle)
		}
	}
	if bestIdx == -1 {
		return "", false
	}
	return bestToken, true
}

func hasPasswordField(body []byte, passwordField string) bool {
	for _, f := range ParseForms(body) {
		if f.Has(passwordField) {
			return true
		}
	}
	return false
}
