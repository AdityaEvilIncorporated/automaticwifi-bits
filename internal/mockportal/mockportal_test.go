package mockportal

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMockFlow(t *testing.T) {
	s := Start(Options{})
	defer s.Close()
	noRedir := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	probe := &http.Client{CheckRedirect: noRedir}
	portal := &http.Client{CheckRedirect: noRedir, Transport: &http.Transport{TLSClientConfig: s.TLSClientConfig()}}

	resp, err := probe.Get(s.Probe.URL + "/")
	if err != nil || resp.StatusCode != 303 {
		t.Fatal(err, resp)
	}
	loc := resp.Header.Get("Location")
	magic := strings.TrimPrefix(loc, s.PortalBase()+"/fgtauth?")
	if magic == loc || magic == "" {
		t.Fatalf("location %q", loc)
	}
	resp, _ = portal.Get(loc)
	page, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(page), `name="magic" value="`+magic+`"`) {
		t.Fatalf("page %s", page)
	}
	form := url.Values{"username": {DefaultUsername}, "password": {DefaultPassword}, "magic": {magic}, "4Tredir": {"x"}}
	resp, _ = portal.PostForm(s.PortalBase()+"/", form)
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "/keepalive?"+s.Token()) {
		t.Fatalf("login reply %s", body)
	}
	resp, _ = portal.PostForm(s.PortalBase()+"/", form)
	if resp.StatusCode != 400 {
		t.Fatal("magic reuse accepted")
	}
	resp, _ = probe.Get(s.Probe.URL + "/generate_204")
	if resp.StatusCode != 204 {
		t.Fatal("authenticated 204 probe")
	}
	resp, _ = probe.Get(s.Probe.URL + "/__mock/state")
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), `"logins_ok":1`) || !strings.Contains(string(b), `"authenticated":true`) {
		t.Fatalf("state %s", b)
	}
	probe.Get(s.Probe.URL + "/__mock/expire")
	resp, _ = portal.Get(s.PortalBase() + "/keepalive?x")
	if resp.StatusCode != 303 {
		t.Fatal("keepalive after expire")
	}
	resp, _ = probe.Get(s.Probe.URL + "/__mock/style?probe=bogus")
	if resp.StatusCode != 400 {
		t.Fatal("bad style accepted")
	}
}
