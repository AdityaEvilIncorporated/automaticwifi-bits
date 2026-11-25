package portal

import "context"

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
	panic("not implemented")
}

// FormatCapture renders the report as plain text lines for the terminal, in this order:
// "On campus: yes|no", "Probe: <kind> (HTTP <status>)", "Portal: <host>:<port>",
// "Form action: <action> (method <method>)", "Fields:" then one line per field
// "  form <n>: <name> (<type>)", "TLS verified by OS trust store: yes|no",
// "Certificate: CN=<subject>, issuer CN=<issuer>", "pinned_cert_sha256 if needed: <spki>",
// then "Note: <note>" lines. Lines whose data is empty (no portal seen, no cert) are omitted.
// It never prints input values.
func FormatCapture(r CaptureReport) string {
	panic("not implemented")
}
