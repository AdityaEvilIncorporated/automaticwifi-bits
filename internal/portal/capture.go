package portal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"autowifi/internal/tlsconf"
)

// CaptureReport is what the `autowifi capture` diagnostic prints. It never holds input
// values, the magic, or any credential.
type CaptureReport struct {
	OnCampus    bool
	ProbeKind   ProbeKind
	ProbeStatus int
	PortalHost  string   // host of the fgtauth URL seen, "" if none
	PortalPort  int      // port of the fgtauth URL seen, 0 if none
	FormAction  string   // action attribute of the login form, "" if none
	FormMethod  string   // lower-cased method attribute
	Fields      []Field  // every input of every form on the login page, in order
	TLSVerified bool     // the portal certificate passed OS trust-store verification
	CertSPKI    string   // lowercase hex SHA-256 of the leaf SPKI (for pinned_cert_sha256)
	CertSubject string   // leaf subject common name
	CertIssuer  string   // leaf issuer common name
	Notes       []string // human hints, e.g. "username_field 'username' not found"
}

// Field is a form input name and type, never its value.
type Field struct {
	Form int // index of the form on the page, from 0
	Name string
	Type string
}

// Capture inspects the portal:
//  1. OnCampus; false → report with OnCampus false and Note "portal host does not resolve:
//     not on the campus network", nil error.
//  2. ProbeOnce (error → return it). ProbeStatus is the HTTP status of the probe reply.
//  3. TLS facts: chain from tlsconf.Inspect(ctx, "<portal_host>:<portal_port>", portal_host)
//     (a handshake only; no HTTP request is sent on that connection). TLSVerified =
//     tlsconf.VerifySystem(chain, portal_host, c.Roots) == nil. CertSPKI/Subject/Issuer from
//     chain[0]. Inspect error → Note "TLS handshake failed: <err>".
//  4. When the probe is captive: PortalHost/PortalPort from the magic URL, GET it with
//     c.Portal (the verified client), ParseForms, FormAction/FormMethod from the first form,
//     Fields = every input of every form. If that GET fails, Note
//     "could not load the login page: <err>" and, when !TLSVerified, also Note
//     "if you trust this certificate, set pinned_cert_sha256 to <spki>".
//  5. Notes "<key> '<name>' not found" (key = username_field, password_field, magic_field,
//     redirect_field) for each configured field missing from every form, only when a login
//     page was parsed.
//
// Values, the magic and query strings never enter the report.
func (c *Client) Capture(ctx context.Context) (CaptureReport, error) {
	if !c.OnCampus(ctx) {
		return CaptureReport{
			OnCampus: false,
			Notes:    []string{"portal host does not resolve: not on the campus network"},
		}, nil
	}

	r := CaptureReport{OnCampus: true}

	probe, status, err := c.probeWithStatus(ctx)
	if err != nil {
		return CaptureReport{}, err
	}
	r.ProbeKind = probe.Kind
	r.ProbeStatus = status

	addr := net.JoinHostPort(c.Cfg.PortalHost, strconv.Itoa(c.Cfg.PortalPort))
	chain, err := tlsconf.Inspect(ctx, addr, c.Cfg.PortalHost)
	if err != nil {
		r.Notes = append(r.Notes, fmt.Sprintf("TLS handshake failed: %v", err))
	} else if len(chain) > 0 && chain[0] != nil {
		r.TLSVerified = tlsconf.VerifySystem(chain, c.Cfg.PortalHost, c.Roots) == nil
		r.CertSPKI = tlsconf.SPKISHA256(chain[0])
		r.CertSubject = chain[0].Subject.CommonName
		r.CertIssuer = chain[0].Issuer.CommonName
	}

	if probe.Kind == ProbeCaptive && OnPortal(c.Cfg, probe.MagicURL) {
		if u, err := url.Parse(probe.MagicURL); err == nil {
			r.PortalHost = u.Hostname()
			if p := u.Port(); p != "" {
				r.PortalPort, _ = strconv.Atoi(p)
			} else if u.Scheme == "https" {
				r.PortalPort = 443
			}
		}

		body, err := c.fetchLoginPage(ctx, probe.MagicURL)
		if err != nil {
			r.Notes = append(r.Notes, fmt.Sprintf("could not load the login page: %v", err))
			if !r.TLSVerified && r.CertSPKI != "" {
				r.Notes = append(r.Notes, "if you trust this certificate, set pinned_cert_sha256 to "+r.CertSPKI)
			}
			return r, nil
		}

		forms := ParseForms(body)
		if len(forms) > 0 {
			// The action is shown without its query string or fragment: those may carry the magic.
			action := forms[0].Action
			if i := strings.IndexAny(action, "?#"); i >= 0 {
				action = action[:i]
			}
			r.FormAction = action
			r.FormMethod = forms[0].Method
		}
		for fi, f := range forms {
			for _, in := range f.Inputs {
				typ := in.Type
				if typ == "" {
					typ = "text"
				}
				r.Fields = append(r.Fields, Field{
					Form: fi,
					Name: in.Name,
					Type: typ,
				})
			}
		}

		expected := []struct {
			key  string
			name string
		}{
			{"username_field", c.Cfg.UsernameField},
			{"password_field", c.Cfg.PasswordField},
			{"magic_field", c.Cfg.MagicField},
			{"redirect_field", c.Cfg.RedirectField},
		}
		for _, exp := range expected {
			if exp.name == "" {
				continue
			}
			found := false
			for _, f := range forms {
				if f.Has(exp.name) {
					found = true
					break
				}
			}
			if !found {
				r.Notes = append(r.Notes, fmt.Sprintf("%s '%s' not found", exp.key, exp.name))
			}
		}
	}

	return r, nil
}

func (c *Client) probeWithStatus(ctx context.Context) (ProbeResult, int, error) {
	res, status, err := c.doProbeWithStatus(ctx, c.Cfg.CheckURL)
	if err == nil {
		return res, status, nil
	}
	if c.Cfg.FallbackCheckURL == "" {
		return ProbeResult{}, 0, err
	}
	return c.doProbeWithStatus(ctx, c.Cfg.FallbackCheckURL)
}

func (c *Client) doProbeWithStatus(ctx context.Context, rawURL string) (ProbeResult, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ProbeResult{}, 0, err
	}
	resp, err := c.Probe.Do(req)
	if err != nil {
		return ProbeResult{}, 0, err
	}
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return ProbeResult{}, 0, err
	}
	return ClassifyProbe(c.Cfg, resp.StatusCode, resp.Header.Get("Location"), body), resp.StatusCode, nil
}

func (c *Client) fetchLoginPage(ctx context.Context, magicURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, magicURL, nil)
	if err != nil {
		return nil, safePortalErr("login page GET", c.Cfg.PortalHost, err)
	}
	resp, err := c.Portal.Do(req)
	if err != nil {
		return nil, safePortalErr("login page GET", c.Cfg.PortalHost, err)
	}
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return nil, safePortalErr("login page read", c.Cfg.PortalHost, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("login page GET to %s: status %d", c.Cfg.PortalHost, resp.StatusCode)
	}
	return body, nil
}

// FormatCapture renders the report as plain text lines for the terminal, in this order:
// "On campus: yes|no", "Probe: <kind> (HTTP <status>)", "Portal: <host>:<port>",
// "Form action: <action> (method <method>)", "Fields:" then one line per field
// "  form <n>: <name> (<type>)", "TLS verified by OS trust store: yes|no",
// "Certificate: CN=<subject>, issuer CN=<issuer>", "pinned_cert_sha256 if needed: <spki>",
// then "Note: <note>" lines. Lines whose data is empty (no portal seen, no cert) are omitted.
// It never prints input values.
func FormatCapture(r CaptureReport) string {
	var lines []string
	if r.OnCampus {
		lines = append(lines, "On campus: yes")
	} else {
		lines = append(lines, "On campus: no")
	}
	if r.ProbeKind != "" {
		lines = append(lines, fmt.Sprintf("Probe: %s (HTTP %d)", r.ProbeKind, r.ProbeStatus))
	}
	if r.PortalHost != "" {
		lines = append(lines, "Portal: "+net.JoinHostPort(r.PortalHost, strconv.Itoa(r.PortalPort)))
	}
	if r.FormAction != "" || r.FormMethod != "" {
		lines = append(lines, fmt.Sprintf("Form action: %s (method %s)", r.FormAction, r.FormMethod))
	}
	if len(r.Fields) > 0 {
		lines = append(lines, "Fields:")
		for _, f := range r.Fields {
			lines = append(lines, fmt.Sprintf("  form %d: %s (%s)", f.Form, f.Name, f.Type))
		}
	}
	if r.CertSPKI != "" || r.TLSVerified {
		if r.TLSVerified {
			lines = append(lines, "TLS verified by OS trust store: yes")
		} else {
			lines = append(lines, "TLS verified by OS trust store: no")
		}
	}
	if r.CertSubject != "" || r.CertIssuer != "" {
		lines = append(lines, fmt.Sprintf("Certificate: CN=%s, issuer CN=%s", r.CertSubject, r.CertIssuer))
	}
	if r.CertSPKI != "" {
		lines = append(lines, "pinned_cert_sha256 if needed: "+r.CertSPKI)
	}
	for _, n := range r.Notes {
		lines = append(lines, "Note: "+n)
	}
	return strings.Join(lines, "\n") + "\n"
}
