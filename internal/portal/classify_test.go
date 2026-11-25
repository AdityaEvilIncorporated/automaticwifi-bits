package portal

import (
	"encoding/json"
	"errors"
	"testing"

	"autowifi/internal/config"
)

func TestClassifyProbeSharedCases(t *testing.T) {
	var f struct {
		Cases []struct {
			Name     string          `json:"name"`
			Config   json.RawMessage `json:"config"`
			Status   int             `json:"status"`
			Location string          `json:"location"`
			Body     string          `json:"body"`
			Expect   struct {
				Kind  ProbeKind `json:"kind"`
				Magic string    `json:"magic"`
			} `json:"expect"`
		} `json:"cases"`
	}
	loadCases(t, "probe.json", &f)
	if len(f.Cases) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			cfg := cfgWith(t, c.Config)
			got := ClassifyProbe(cfg, c.Status, c.Location, []byte(c.Body))
			if got.Kind != c.Expect.Kind || got.Magic != c.Expect.Magic {
				t.Fatalf("got kind=%s magic=%q, want kind=%s magic=%q", got.Kind, got.Magic, c.Expect.Kind, c.Expect.Magic)
			}
			wantURL := ""
			if got.Kind == ProbeCaptive {
				wantURL = cfg.PortalBase() + "/fgtauth?" + c.Expect.Magic
			}
			if got.MagicURL != wantURL {
				t.Fatalf("MagicURL = %q, want %q", got.MagicURL, wantURL)
			}
		})
	}
}

func TestParseLoginPageSharedCases(t *testing.T) {
	var f struct {
		Cases []struct {
			Name       string          `json:"name"`
			Config     json.RawMessage `json:"config"`
			ProbeMagic string          `json:"probe_magic"`
			Body       string          `json:"body"`
			Expect     struct {
				OK       bool    `json:"ok"`
				Magic    string  `json:"magic"`
				Redirect string  `json:"redirect"`
				Error    *string `json:"error"`
			} `json:"expect"`
		} `json:"cases"`
	}
	loadCases(t, "login_page.json", &f)
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			lf, err := ParseLoginPage(cfgWith(t, c.Config), []byte(c.Body), c.ProbeMagic)
			if !c.Expect.OK {
				if !errors.Is(err, ErrFormChanged) {
					t.Fatalf("err = %v, want ErrFormChanged", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if lf.Magic != c.Expect.Magic || lf.Redirect != c.Expect.Redirect {
				t.Fatalf("got %+v, want magic=%q redirect=%q", lf, c.Expect.Magic, c.Expect.Redirect)
			}
		})
	}
}

func TestParseLoginPageAction(t *testing.T) {
	lf, err := ParseLoginPage(config.Default(), []byte(`<form action="/fgtauth" method=post><input name=magic value=1><input name=username><input name=password></form>`), "x")
	if err != nil || lf.Action != "/fgtauth" {
		t.Fatalf("%+v %v", lf, err)
	}
}

func TestClassifyLoginReplySharedCases(t *testing.T) {
	var f struct {
		Cases []struct {
			Name     string          `json:"name"`
			Config   json.RawMessage `json:"config"`
			Status   int             `json:"status"`
			Location string          `json:"location"`
			Body     string          `json:"body"`
			Expect   struct {
				Outcome      Outcome `json:"outcome"`
				KeepaliveURL string  `json:"keepalive_url"`
				LogoutURL    string  `json:"logout_url"`
			} `json:"expect"`
		} `json:"cases"`
	}
	loadCases(t, "login_reply.json", &f)
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			got := ClassifyLoginReply(cfgWith(t, c.Config), c.Status, c.Location, []byte(c.Body))
			want := LoginReply{c.Expect.Outcome, c.Expect.KeepaliveURL, c.Expect.LogoutURL}
			if got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestKeepaliveSharedCases(t *testing.T) {
	var f struct {
		Cases []struct {
			Name   string `json:"name"`
			Status int    `json:"status"`
			Body   string `json:"body"`
			Expect struct {
				Alive bool `json:"alive"`
			} `json:"expect"`
		} `json:"cases"`
	}
	loadCases(t, "keepalive.json", &f)
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if got := KeepaliveAlive(config.Default(), c.Status, []byte(c.Body)); got != c.Expect.Alive {
				t.Fatalf("alive = %v, want %v", got, c.Expect.Alive)
			}
		})
	}
}

func TestOnPortal(t *testing.T) {
	cfg := config.Default()
	cases := map[string]bool{
		"https://fw.bits-pilani.ac.in:8090/keepalive?1":             true,
		"https://FW.bits-pilani.ac.in:8090/fgtauth?1":               true,
		"http://fw.bits-pilani.ac.in:8090/keepalive?1":              false,
		"https://fw.bits-pilani.ac.in/keepalive?1":                  false,
		"https://fw.bits-pilani.ac.in:8091/keepalive?1":             false,
		"https://fw.bits-pilani.ac.in.evil.example:8090/x":          false,
		"https://evil.example/?h=https://fw.bits-pilani.ac.in:8090": false,
		"https://user@fw.bits-pilani.ac.in:8090/x":                  true,
		"not a url": false,
		"":          false,
	}
	for u, want := range cases {
		if got := OnPortal(cfg, u); got != want {
			t.Errorf("OnPortal(%q) = %v, want %v", u, got, want)
		}
	}
	cfg.PortalPort = 443
	if !OnPortal(cfg, "https://fw.bits-pilani.ac.in/x") || !OnPortal(cfg, "https://fw.bits-pilani.ac.in:443/x") {
		t.Error("port 443 may be implicit or explicit")
	}
}
