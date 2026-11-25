package tray

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"

	"autowifi/internal/portal"
)

func TestStatusTextCoversEveryOutcome(t *testing.T) {
	for _, o := range []portal.Outcome{portal.OffCampus, portal.Online, portal.Foreign, portal.LoggedIn,
		portal.Rejected, portal.FormChanged, portal.PortalError, portal.NoCredentials, "keepalive_ok", "paused", "logged_out"} {
		if StatusText(o) == "Working…" {
			t.Errorf("no wording for %s", o)
		}
	}
}

func TestIsReleaseURL(t *testing.T) {
	ok := []string{"https://github.com/o/r/releases/tag/v1.2.3"}
	bad := []string{"", "http://github.com/o/r", "https://github.com.evil.example/x", "https://evil.example/https://github.com/",
		"file:///etc/passwd", "https://github.com/o/r\" & calc", "javascript:alert(1)",
		"https://github.com/login", "https://github.com/o/r", "https://github.com/o/r/settings",
		"https://github.com/o/r/releases/../../settings", "https://github.com/o/r/releases/tag/v1?x=1"}
	for _, u := range ok {
		if !IsReleaseURL(u) {
			t.Errorf("%q refused", u)
		}
	}
	for _, u := range bad {
		if IsReleaseURL(u) {
			t.Errorf("%q accepted", u)
		}
		if OpenURL(u) == nil {
			t.Errorf("OpenURL(%q) did not refuse", u)
		}
	}
}

func TestIcon(t *testing.T) {
	p := iconPNG()
	img, err := png.Decode(bytes.NewReader(p))
	if err != nil || img.Bounds().Dx() != 32 {
		t.Fatalf("icon png: %v", err)
	}
	ico := icoFromPNG(p, 32)
	if binary.LittleEndian.Uint16(ico[2:]) != 1 || binary.LittleEndian.Uint32(ico[18:]) != 22 || !bytes.Equal(ico[22:], p) {
		t.Fatal("ico header")
	}
}
