package tlsconf

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func server(t *testing.T) *httptest.Server {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, srv *httptest.Server, cfg *tls.Config) error {
	t.Helper()
	c := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}
	resp, err := c.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func TestSPKISHA256(t *testing.T) {
	srv := server(t)
	cert := srv.Certificate()
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	if got := SPKISHA256(cert); got != hex.EncodeToString(sum[:]) || len(got) != 64 || strings.ToLower(got) != got {
		t.Fatalf("SPKISHA256 = %q", got)
	}
}

func TestTrustedByRoots(t *testing.T) {
	srv := server(t)
	cfg, err := PortalTLSConfig("127.0.0.1", "", srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion < tls.VersionTLS12 || cfg.ServerName != "127.0.0.1" {
		t.Fatalf("cfg %+v", cfg)
	}
	if err := get(t, srv, cfg); err != nil {
		t.Fatalf("trusted cert refused: %v", err)
	}
}

func TestUntrustedWithoutPinIsRefused(t *testing.T) {
	srv := server(t)
	cfg, err := PortalTLSConfig("127.0.0.1", "", nil) // OS store: mock CA not trusted
	if err != nil {
		t.Fatal(err)
	}
	if err := get(t, srv, cfg); err == nil {
		t.Fatal("untrusted cert accepted without pin")
	}
}

func TestUntrustedWithMatchingPinIsAccepted(t *testing.T) {
	srv := server(t)
	cfg, err := PortalTLSConfig("127.0.0.1", SPKISHA256(srv.Certificate()), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := get(t, srv, cfg); err != nil {
		t.Fatalf("pinned cert refused: %v", err)
	}
}

func TestUntrustedWithWrongPinIsRefused(t *testing.T) {
	srv := server(t)
	cfg, err := PortalTLSConfig("127.0.0.1", strings.Repeat("ab", 32), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := get(t, srv, cfg); err == nil {
		t.Fatal("wrong pin accepted")
	}
}

func TestConfigBoundToHost(t *testing.T) {
	srv := server(t)
	// pin matches, but the config is for another host: the dial uses ServerName from cfg,
	// so force a different ServerName to simulate reuse for another host.
	cfg, err := PortalTLSConfig("fw.bits-pilani.ac.in", SPKISHA256(srv.Certificate()), nil)
	if err != nil {
		t.Fatal(err)
	}
	other := cfg.Clone()
	other.ServerName = "example.com"
	if err := get(t, srv, other); err == nil {
		t.Fatal("config reused for another host was accepted")
	}
}

func TestBadPin(t *testing.T) {
	for _, pin := range []string{"abc", strings.Repeat("G", 64), strings.Repeat("AB", 32)} {
		if _, err := PortalTLSConfig("h", pin, nil); err == nil {
			t.Errorf("pin %q accepted", pin)
		}
	}
}

func TestVerifySystem(t *testing.T) {
	srv := server(t)
	roots := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	chain, err := Inspect(context.Background(), strings.TrimPrefix(srv.URL, "https://"), "127.0.0.1")
	if err != nil || len(chain) == 0 {
		t.Fatalf("Inspect: %v %d", err, len(chain))
	}
	if !chain[0].Equal(srv.Certificate()) {
		t.Fatal("Inspect returned a different leaf")
	}
	if err := VerifySystem(chain, "127.0.0.1", roots); err != nil {
		t.Fatalf("VerifySystem with roots: %v", err)
	}
	if err := VerifySystem(chain, "127.0.0.1", nil); err == nil {
		t.Fatal("VerifySystem with OS store should fail for the test CA")
	}
	if err := VerifySystem(chain, "wrong.example", roots); err == nil {
		t.Fatal("VerifySystem wrong host should fail")
	}
	if err := VerifySystem(nil, "127.0.0.1", roots); err == nil {
		t.Fatal("empty chain should fail")
	}
}

func TestInspectSendsNoRequest(t *testing.T) {
	hits := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	if _, err := Inspect(context.Background(), strings.TrimPrefix(srv.URL, "https://"), "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if hits != 0 {
		t.Fatal("Inspect sent an HTTP request")
	}
}

func TestDNSHostRequiresMatchingServerName(t *testing.T) {
	srv := server(t)
	// A DNS-named config whose ServerName was blanked (as an IP dial would do) must not pass the
	// binding check, even with a matching pin.
	cfg, err := PortalTLSConfig("fw.bits-pilani.ac.in", SPKISHA256(srv.Certificate()), nil)
	if err != nil {
		t.Fatal(err)
	}
	blank := cfg.Clone()
	blank.ServerName = ""
	if err := get(t, srv, blank); err == nil {
		t.Fatal("empty server name accepted for a DNS host")
	}
}

func TestResumedSessionsAreStillVerified(t *testing.T) {
	srv := server(t)
	cfg, err := PortalTLSConfig("127.0.0.1", SPKISHA256(srv.Certificate()), nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ClientSessionCache = tls.NewLRUClientSessionCache(4)
	tr := &http.Transport{TLSClientConfig: cfg, DisableKeepAlives: true}
	c := &http.Client{Transport: tr}
	for i := 0; i < 2; i++ { // second request resumes the session
		resp, err := c.Get(srv.URL)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		resp.Body.Close()
	}
	// Same cache, now a config whose pin does not match: resumption must not skip the check.
	bad, _ := PortalTLSConfig("127.0.0.1", strings.Repeat("ab", 32), nil)
	bad.ClientSessionCache = cfg.ClientSessionCache
	c2 := &http.Client{Transport: &http.Transport{TLSClientConfig: bad, DisableKeepAlives: true}}
	if resp, err := c2.Get(srv.URL); err == nil {
		resp.Body.Close()
		t.Fatal("resumed session bypassed verification")
	}
}
