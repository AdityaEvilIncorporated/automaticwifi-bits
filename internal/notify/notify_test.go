package notify

import (
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	cases := map[Reason]string{
		ReasonRejected:       "Campus Wi-Fi login was rejected. Update your ID and password to resume.",
		ReasonRepeatedErrors: "Campus Wi-Fi login keeps failing. Will keep retrying every 5 minutes.",
		ReasonFormChanged:    "The campus login page has changed. Run 'autowifi capture' and report it.",
		ReasonNoCredentials:  "Campus Wi-Fi needs a login. Set your ID and password in AutoWiFi BITS.",
		Reason("other"):      "AutoWiFi BITS needs attention.",
	}
	for r, want := range cases {
		title, body := Message(r)
		if title != "AutoWiFi BITS" || body != want {
			t.Errorf("Message(%q) = %q, %q", r, title, body)
		}
	}
}

func TestAllow(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		last time.Time
		want bool
	}{
		{"never", time.Time{}, true},
		{"just now", now, false},
		{"59 min ago", now.Add(-59 * time.Minute), false},
		{"exactly 1 h", now.Add(-time.Hour), true},
		{"2 h ago", now.Add(-2 * time.Hour), true},
		{"clock moved back", now.Add(3 * time.Hour), true},
	}
	for _, c := range cases {
		if got := Allow(c.last, now); got != c.want {
			t.Errorf("%s: Allow = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestNewSystemNotNil(t *testing.T) {
	if NewSystem() == nil {
		t.Fatal("NewSystem returned nil")
	}
}
