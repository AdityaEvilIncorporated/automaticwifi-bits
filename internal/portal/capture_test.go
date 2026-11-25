package portal

import (
	"context"
	"strings"
	"testing"

	"autowifi/internal/mockportal"
	"autowifi/internal/tlsconf"
)

func TestCaptureOnMock(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	s.SetFields(mockportal.Fields{Username: "user_id"})
	c := testClient(s)
	r, err := c.Capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	host, port := s.PortalHostPort()
	if !r.OnCampus || r.ProbeKind != ProbeCaptive || r.ProbeStatus != 303 || r.PortalHost != host || r.PortalPort != port {
		t.Fatalf("report %+v", r)
	}
	if r.FormAction != "/" || r.FormMethod != "post" {
		t.Fatalf("form %q %q", r.FormAction, r.FormMethod)
	}
	want := []Field{{0, "4Tredir", "hidden"}, {0, "magic", "hidden"}, {0, "user_id", "text"}, {0, "password", "password"}, {0, "", "submit"}}
	if len(r.Fields) != len(want) {
		t.Fatalf("fields %+v", r.Fields)
	}
	for i := range want {
		if r.Fields[i] != want[i] {
			t.Fatalf("field %d = %+v, want %+v", i, r.Fields[i], want[i])
		}
	}
	if !r.TLSVerified {
		t.Fatal("mock CA is in Roots: TLSVerified should be true")
	}
	if r.CertSPKI != tlsconf.SPKISHA256(s.Portal.Certificate()) {
		t.Fatalf("CertSPKI %q", r.CertSPKI)
	}
	foundNote := false
	for _, n := range r.Notes {
		if n == "username_field 'username' not found" {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("notes %v", r.Notes)
	}

	out := FormatCapture(r)
	for _, line := range []string{
		"On campus: yes",
		"Probe: captive (HTTP 303)",
		"Form action: / (method post)",
		"Fields:",
		"  form 0: user_id (text)",
		"  form 0: magic (hidden)",
		"TLS verified by OS trust store: yes",
		"pinned_cert_sha256 if needed: " + r.CertSPKI,
		"Note: username_field 'username' not found",
	} {
		if !strings.Contains(out, line+"\n") {
			t.Errorf("output missing line %q:\n%s", line, out)
		}
	}
	if strings.Contains(out, "neverssl.com/") || strings.Contains(out, "fgtauth?") {
		t.Fatalf("output leaks values or magic:\n%s", out)
	}
}

func TestCaptureUntrustedCertStillReportsPin(t *testing.T) {
	s := mockportal.Start(mockportal.Options{})
	defer s.Close()
	c := testClient(s)
	c.Roots = nil // OS trust store: the mock CA is not in it
	c.Portal.Transport = nil
	r, err := c.Capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.TLSVerified || r.CertSPKI != tlsconf.SPKISHA256(s.Portal.Certificate()) {
		t.Fatalf("report %+v", r)
	}
	hint := "if you trust this certificate, set pinned_cert_sha256 to " + r.CertSPKI
	ok := false
	for _, n := range r.Notes {
		ok = ok || n == hint
	}
	if !ok || len(r.Fields) != 0 {
		t.Fatalf("notes %v fields %v", r.Notes, r.Fields)
	}
	if st := s.Snapshot(); st.LoginsOK+st.LoginsFailed != 0 {
		t.Fatal("capture must never log in")
	}
}

func TestCaptureOffCampus(t *testing.T) {
	c := &Client{Cfg: testClientCfgOff(), Resolver: &failResolver{}}
	r, err := c.Capture(context.Background())
	if err != nil || r.OnCampus {
		t.Fatalf("%+v %v", r, err)
	}
	if !strings.Contains(FormatCapture(r), "On campus: no\n") {
		t.Fatal(FormatCapture(r))
	}
}

func TestFormatCaptureStripsControlCharacters(t *testing.T) {
	r := CaptureReport{
		FormAction:  "/\x1b]0;pwned\x07login",
		FormMethod:  "POST\r",
		CertSubject: "fw\x1b[2J.example",
		CertIssuer:  strings.Repeat("A", 500),
		Fields:      []Field{{Form: 0, Name: "user\x00name", Type: "text\x7f"}},
	}
	out := FormatCapture(r)
	for _, c := range out {
		if c != '\n' && (c < 0x20 || c == 0x7f) {
			t.Fatalf("control character %q in output:\n%q", c, out)
		}
	}
	if !strings.Contains(out, "/]0;pwnedlogin") || !strings.Contains(out, "username (text)") {
		t.Errorf("printable text should survive:\n%s", out)
	}
	if strings.Contains(out, strings.Repeat("A", 300)) {
		t.Error("long values must be capped")
	}
}
