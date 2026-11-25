package portal

import (
	"context"
	"crypto/x509"
	"net/http"

	"autowifi/internal/config"
	"autowifi/internal/creds"
)

// MaxBody is the most bytes read from any response (PROTOCOL.md section 2).
const MaxBody = 256 << 10

// Resolver is the DNS lookup used for the campus test; net.DefaultResolver satisfies it.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Client talks to the probe URLs and the portal. Build it with NewClient in production;
// tests fill the fields directly.
type Client struct {
	Cfg config.PortalConfig
	// Probe is used for check_url / fallback_check_url. It must not follow redirects.
	Probe *http.Client
	// Portal is used for every portal request (TLS per tlsconf). It must not follow redirects.
	Portal *http.Client
	// Resolver does the campus DNS test.
	Resolver Resolver
	// Roots is the trust store for the capture diagnostic's "TLS verified" check
	// (nil = OS trust store). Tests set the mock CA.
	Roots *x509.CertPool
}

// Result is the outcome of one CheckOnce plus the session data on LoggedIn.
type Result struct {
	Outcome      Outcome
	Magic        string // the magic used, LoggedIn only (kept in state, never logged)
	KeepaliveURL string // LoggedIn only
	LogoutURL    string // LoggedIn only
	// Err is the underlying error for PortalError (for logs; must not contain credentials).
	Err error
}

// NewClient builds a Client with net.DefaultResolver, a probe client (timeout 10 s, no
// redirects, default transport, proxy from environment disabled) and a portal client
// (timeout 15 s, no redirects, TLS config from tlsconf.PortalTLSConfig(cfg.PortalHost,
// cfg.PinnedCertSHA256, nil), proxy disabled). Errors come from tlsconf.
func NewClient(cfg config.PortalConfig) (*Client, error) {
	panic("not implemented")
}

// OnCampus implements PROTOCOL.md section 2 step 1: true when cfg.PortalHost is an IP literal
// or Resolver.LookupHost returns at least one address without error.
func (c *Client) OnCampus(ctx context.Context) bool {
	panic("not implemented")
}

// ProbeOnce implements section 2 steps 2-3: GET check_url (on a network error retry once with
// fallback_check_url), read at most MaxBody bytes, ClassifyProbe. A non-nil error means both
// probes had network errors. An HTTP status is never an error here.
func (c *Client) ProbeOnce(ctx context.Context) (ProbeResult, error) {
	panic("not implemented")
}

// Login implements section 2 steps 5-6 for an already-classified captive probe.
// It refuses (returns Result{Outcome: Foreign}) when !OnPortal(Cfg, probe.MagicURL).
// Steps: GET probe.MagicURL with Portal (non-2xx or network error → PortalError);
// ParseLoginPage (ErrFormChanged → FormChanged); POST the four fields in protocol order to
// Cfg.PortalBase()+Cfg.LoginPath as application/x-www-form-urlencoded; ClassifyLoginReply.
// Result.Err is set for PortalError and never contains the request body.
func (c *Client) Login(ctx context.Context, probe ProbeResult, cr creds.Credentials) Result {
	panic("not implemented")
}

// CheckOnce runs the whole of PROTOCOL.md section 2 and returns exactly one outcome.
// get is called only when the probe is captive; it returns the stored credentials, or an
// error wrapping creds.ErrNotFound (→ NoCredentials). Any other get error → PortalError
// with Err set. Must pass protocol/testcases/scenarios.json against the mock portal.
func (c *Client) CheckOnce(ctx context.Context, get func() (creds.Credentials, error)) Result {
	panic("not implemented")
}

// Keepalive GETs keepaliveURL with Portal and returns KeepaliveAlive. It returns false
// without any request when !OnPortal(Cfg, keepaliveURL). Network errors return false.
func (c *Client) Keepalive(ctx context.Context, keepaliveURL string) bool {
	panic("not implemented")
}

// Logout GETs logoutURL with Portal (only when OnPortal) and ignores the reply. It returns
// the network error, if any, for logging.
func (c *Client) Logout(ctx context.Context, logoutURL string) error {
	panic("not implemented")
}
