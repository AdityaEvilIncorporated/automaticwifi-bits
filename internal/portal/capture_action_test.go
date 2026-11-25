package portal

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCaptureStripsActionQuery(t *testing.T) {
	page := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<form action="/login?magic=abc123#x" method="post"><input name=username><input type=password name=password><input type=hidden name=magic value=abc123></form>`)
	}))
	defer page.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(page.URL, "https://"))
	c := &Client{Cfg: testClientCfgOff(), Resolver: okResolver{}, Portal: page.Client()}
	c.Cfg.PortalHost = host
	c.Cfg.PortalPort, _ = strconv.Atoi(port)
	c.Portal.CheckRedirect = noRedirect
	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, c.Cfg.PortalBase()+"/fgtauth?abc123", http.StatusSeeOther)
	}))
	defer probe.Close()
	c.Cfg.CheckURL = probe.URL
	c.Probe = &http.Client{CheckRedirect: noRedirect}

	r, err := c.Capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.FormAction != "/login" {
		t.Fatalf("FormAction = %q, want /login", r.FormAction)
	}
	if out := FormatCapture(r); strings.Contains(out, "abc123") {
		t.Fatalf("report leaks the magic:\n%s", out)
	}
}
