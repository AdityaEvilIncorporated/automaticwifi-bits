package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"v1.2.3", "v1.2.4", true},
		{"1.2.3", "v1.3.0", true},
		{"v1.2.3", "v2.0.0", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.10.0", "v1.9.9", false},
		{"v2.0.0", "v1.99.99", false},
		{"dev", "v9.0.0", false},
		{"v1.0.0", "garbage", false},
		{"v1.0.0", "v2.0.0-rc1", false},
		{"v1.0.0-rc1", "v1.0.0", false},
		{"v1.0.0-rc1", "v1.0.1", true},
		{"v1.2", "v1.3.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q,%q) = %v, want %v", c.cur, c.latest, got, c.want)
		}
	}
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" || r.Header.Get("User-Agent") != "autowifi-bits" {
			http.Error(w, "bad headers", 400)
			return
		}
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte(`{"tag_name":"v1.4.0","html_url":"https://github.com/o/r/releases/tag/v1.4.0","assets":[]}`))
		case "/empty":
			w.Write([]byte(`{"tag_name":""}`))
		case "/bad":
			w.Write([]byte(`{`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	rel, err := Latest(ctx, srv.Client(), srv.URL+"/ok")
	if err != nil || rel.Tag != "v1.4.0" || rel.URL != "https://github.com/o/r/releases/tag/v1.4.0" {
		t.Fatalf("Latest = %+v, %v", rel, err)
	}
	if _, err := Latest(ctx, srv.Client(), srv.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("404: err = %v", err)
	}
	if _, err := Latest(ctx, srv.Client(), srv.URL+"/empty"); err == nil {
		t.Fatal("empty tag: want error")
	}
	if _, err := Latest(ctx, srv.Client(), srv.URL+"/bad"); err == nil {
		t.Fatal("bad json: want error")
	}
	if _, err := Latest(ctx, nil, srv.URL+"/ok"); err != nil {
		t.Fatalf("nil client: %v", err)
	}
}
