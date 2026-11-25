package portal

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"autowifi/internal/config"
	"autowifi/internal/creds"
	"autowifi/internal/tlsconf"
)

// MaxBody is the most bytes read from any response (PROTOCOL.md section 2).
const MaxBody = 256 << 10

const (
	dnsTimeout    = 5 * time.Second
	probeTimeout  = 10 * time.Second
	portalTimeout = 15 * time.Second
)

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

func readLimitedBody(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, MaxBody))
}

func safePortalErr(op, host string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return fmt.Errorf("%s to %s: %w", op, host, err)
}

// NewClient builds a Client with net.DefaultResolver, a probe client (timeout 10 s, no
// redirects, default transport, proxy from environment disabled) and a portal client
// (timeout 15 s, no redirects, TLS config from tlsconf.PortalTLSConfig(cfg.PortalHost,
// cfg.PinnedCertSHA256, nil), proxy disabled). Errors come from tlsconf.
func NewClient(cfg config.PortalConfig) (*Client, error) {
	tlsCfg, err := tlsconf.PortalTLSConfig(cfg.PortalHost, cfg.PinnedCertSHA256, nil)
	if err != nil {
		return nil, err
	}
	probeTransport := http.DefaultTransport.(*http.Transport).Clone()
	probeTransport.Proxy = nil
	portalTransport := http.DefaultTransport.(*http.Transport).Clone()
	portalTransport.Proxy = nil
	portalTransport.TLSClientConfig = tlsCfg

	noRedirect := func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &Client{
		Cfg: cfg,
		Probe: &http.Client{
			Timeout:       probeTimeout,
			CheckRedirect: noRedirect,
			Transport:     probeTransport,
		},
		Portal: &http.Client{
			Timeout:       portalTimeout,
			CheckRedirect: noRedirect,
			Transport:     portalTransport,
		},
		Resolver: net.DefaultResolver,
	}, nil
}

// OnCampus implements PROTOCOL.md section 2 step 1: true when cfg.PortalHost is an IP literal
// or Resolver.LookupHost returns at least one address without error.
func (c *Client) OnCampus(ctx context.Context) bool {
	if net.ParseIP(c.Cfg.PortalHost) != nil {
		return true
	}
	resolver := c.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	lookupCtx, cancel := context.WithTimeout(ctx, dnsTimeout)
	defer cancel()
	addrs, err := resolver.LookupHost(lookupCtx, c.Cfg.PortalHost)
	return err == nil && len(addrs) > 0
}

func (c *Client) doProbe(ctx context.Context, rawURL string) (ProbeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ProbeResult{}, err
	}
	resp, err := c.Probe.Do(req)
	if err != nil {
		return ProbeResult{}, err
	}
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return ProbeResult{}, err
	}
	return ClassifyProbe(c.Cfg, resp.StatusCode, resp.Header.Get("Location"), body), nil
}

// ProbeOnce implements section 2 steps 2-3: GET check_url (on a network error retry once with
// fallback_check_url), read at most MaxBody bytes, ClassifyProbe. A non-nil error means both
// probes had network errors. An HTTP status is never an error here.
func (c *Client) ProbeOnce(ctx context.Context) (ProbeResult, error) {
	res, err := c.doProbe(ctx, c.Cfg.CheckURL)
	if err == nil {
		return res, nil
	}
	if c.Cfg.FallbackCheckURL == "" {
		return ProbeResult{}, err
	}
	return c.doProbe(ctx, c.Cfg.FallbackCheckURL)
}

// Login implements section 2 steps 5-6 for an already-classified captive probe.
// It refuses (returns Result{Outcome: Foreign}) when !OnPortal(Cfg, probe.MagicURL).
// Steps: GET probe.MagicURL with Portal (non-2xx or network error → PortalError);
// ParseLoginPage (ErrFormChanged → FormChanged); POST the four fields in protocol order to
// Cfg.PortalBase()+Cfg.LoginPath as application/x-www-form-urlencoded; ClassifyLoginReply.
// Result.Err is set for PortalError and never contains the request body.
func (c *Client) Login(ctx context.Context, probe ProbeResult, cr creds.Credentials) Result {
	if !OnPortal(c.Cfg, probe.MagicURL) {
		return Result{Outcome: Foreign}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probe.MagicURL, nil)
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login page GET", c.Cfg.PortalHost, err)}
	}
	resp, err := c.Portal.Do(req)
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login page GET", c.Cfg.PortalHost, err)}
	}
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login page read", c.Cfg.PortalHost, err)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{Outcome: PortalError, Err: fmt.Errorf("login page GET to %s: status %d", c.Cfg.PortalHost, resp.StatusCode)}
	}

	form, err := ParseLoginPage(c.Cfg, body, probe.Magic)
	if err != nil {
		if errors.Is(err, ErrFormChanged) {
			return Result{Outcome: FormChanged}
		}
		return Result{Outcome: PortalError, Err: err}
	}

	pairs := [][2]string{
		{c.Cfg.UsernameField, cr.Username},
		{c.Cfg.PasswordField, cr.Password},
		{c.Cfg.MagicField, form.Magic},
		{c.Cfg.RedirectField, form.Redirect},
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = url.QueryEscape(p[0]) + "=" + url.QueryEscape(p[1])
	}
	formBody := strings.Join(parts, "&")

	postURL := c.Cfg.PortalBase() + c.Cfg.LoginPath
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, strings.NewReader(formBody))
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login POST", c.Cfg.PortalHost, err)}
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postResp, err := c.Portal.Do(postReq)
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login POST", c.Cfg.PortalHost, err)}
	}
	postBody, err := readLimitedBody(postResp.Body)
	if err != nil {
		return Result{Outcome: PortalError, Err: safePortalErr("login POST read", c.Cfg.PortalHost, err)}
	}

	reply := ClassifyLoginReply(c.Cfg, postResp.StatusCode, postResp.Header.Get("Location"), postBody)
	res := Result{
		Outcome:      reply.Outcome,
		KeepaliveURL: reply.KeepaliveURL,
		LogoutURL:    reply.LogoutURL,
	}
	if reply.Outcome == LoggedIn {
		res.Magic = form.Magic
	} else if reply.Outcome == PortalError {
		res.Err = fmt.Errorf("login POST to %s: status %d", c.Cfg.PortalHost, postResp.StatusCode)
	}
	return res
}

// CheckOnce runs the whole of PROTOCOL.md section 2 and returns exactly one outcome.
// get is called only when the probe is captive; it returns the stored credentials, or an
// error wrapping creds.ErrNotFound (→ NoCredentials). Any other get error → PortalError
// with Err set. Must pass protocol/testcases/scenarios.json against the mock portal.
func (c *Client) CheckOnce(ctx context.Context, get func() (creds.Credentials, error)) Result {
	if !c.OnCampus(ctx) {
		return Result{Outcome: OffCampus}
	}
	probe, err := c.ProbeOnce(ctx)
	if err != nil {
		return Result{Outcome: PortalError, Err: err}
	}
	switch probe.Kind {
	case ProbeOnline:
		return Result{Outcome: Online}
	case ProbeForeign:
		return Result{Outcome: Foreign}
	case ProbeMalformed:
		return Result{Outcome: FormChanged}
	case ProbeCaptive:
		cr, err := get()
		if err != nil {
			if errors.Is(err, creds.ErrNotFound) {
				return Result{Outcome: NoCredentials}
			}
			return Result{Outcome: PortalError, Err: err}
		}
		return c.Login(ctx, probe, cr)
	default:
		return Result{Outcome: FormChanged}
	}
}

// Keepalive GETs keepaliveURL with Portal and returns KeepaliveAlive. It returns false
// without any request when !OnPortal(Cfg, keepaliveURL). Network errors return false.
func (c *Client) Keepalive(ctx context.Context, keepaliveURL string) bool {
	if !OnPortal(c.Cfg, keepaliveURL) {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, keepaliveURL, nil)
	if err != nil {
		return false
	}
	resp, err := c.Portal.Do(req)
	if err != nil {
		return false
	}
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return false
	}
	return KeepaliveAlive(c.Cfg, resp.StatusCode, body)
}

// Logout GETs logoutURL with Portal (only when OnPortal) and ignores the reply. It returns
// the network error, if any, for logging.
func (c *Client) Logout(ctx context.Context, logoutURL string) error {
	if !OnPortal(c.Cfg, logoutURL) {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logoutURL, nil)
	if err != nil {
		return safePortalErr("logout GET", c.Cfg.PortalHost, err)
	}
	resp, err := c.Portal.Do(req)
	if err != nil {
		return safePortalErr("logout GET", c.Cfg.PortalHost, err)
	}
	_, _ = readLimitedBody(resp.Body)
	return nil
}
