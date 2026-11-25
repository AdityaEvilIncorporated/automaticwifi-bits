// Package portal implements the FortiGate protocol in protocol/PROTOCOL.md: pure classifiers
// (this file), the network client (client.go) and the capture diagnostic (capture.go).
package portal

import (
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
	panic("not implemented")
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
	panic("not implemented")
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
	panic("not implemented")
}

// KeepaliveAlive implements PROTOCOL.md section 7 "Alive". Must pass
// protocol/testcases/keepalive.json.
func KeepaliveAlive(cfg config.PortalConfig, status int, body []byte) bool {
	panic("not implemented")
}

// OnPortal reports whether rawURL is https, on cfg.PortalHost (case-insensitive) and on
// cfg.PortalPort (explicit port required, except that port 443 may be implicit). It is the
// host rule from PROTOCOL.md section 2 and guards every portal request.
func OnPortal(cfg config.PortalConfig, rawURL string) bool {
	panic("not implemented")
}
