package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func sample() SessionState {
	return SessionState{
		MagicToken:          "abc",
		KeepaliveURL:        "https://fw.bits-pilani.ac.in:8090/keepalive?01",
		LogoutURL:           "https://fw.bits-pilani.ac.in:8090/logout?01",
		ConsecutiveFailures: 2,
		LastNotificationAt:  time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC),
		CredentialsRejected: true,
	}
}

func TestLoggedInAndClear(t *testing.T) {
	s := sample()
	if !s.LoggedIn() {
		t.Fatal("LoggedIn false with keepalive URL")
	}
	s.ClearSession()
	if s.LoggedIn() || s.MagicToken != "" || s.LogoutURL != "" {
		t.Fatalf("ClearSession left session data: %+v", s)
	}
	if s.ConsecutiveFailures != 2 || !s.CredentialsRejected || s.LastNotificationAt.IsZero() {
		t.Fatalf("ClearSession touched other fields: %+v", s)
	}
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d", "state.json")
	if err := Save(p, sample()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	want := sample()
	if !got.LastNotificationAt.Equal(want.LastNotificationAt) {
		t.Fatalf("time %v != %v", got.LastNotificationAt, want.LastNotificationAt)
	}
	got.LastNotificationAt, want.LastNotificationAt = time.Time{}, time.Time{}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if runtime.GOOS != "windows" {
		st, _ := os.Stat(p)
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v", st.Mode().Perm())
		}
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("tmp file left")
	}
}

func TestJSONKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	Save(p, sample())
	b, _ := os.ReadFile(p)
	for _, k := range []string{"magic_token", "keepalive_url", "logout_url", "consecutive_failures", "last_notification_at", "credentials_rejected"} {
		if !strings.Contains(string(b), `"`+k+`"`) {
			t.Errorf("key %s missing in %s", k, b)
		}
	}
}

func TestLoadMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(filepath.Join(dir, "none.json"))
	if err != nil || s != (SessionState{}) {
		t.Fatalf("missing: %+v %v", s, err)
	}
	p := filepath.Join(dir, "bad.json")
	os.WriteFile(p, []byte("{not json"), 0o600)
	s, err = Load(p)
	if err != nil || s != (SessionState{}) {
		t.Fatalf("corrupt: %+v %v", s, err)
	}
}
