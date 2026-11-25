//go:build darwin

package autostart

import (
	"errors"
	"strings"
	"testing"
)

// stubLaunchctl replaces launchctl: "print" reports the job as not loaded, everything else
// succeeds. On cleanup it checks install bootstrapped the job and nothing ever booted it out
// (bootout from inside the launchd-started agent would kill the running app).
func stubLaunchctl(t *testing.T) func() {
	old := launchctl
	var calls []string
	launchctl = func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "print" {
			return errors.New("Could not find service")
		}
		return nil
	}
	return func() {
		launchctl = old
		joined := strings.Join(calls, "\n")
		if !strings.Contains(joined, "bootstrap gui/") || !strings.Contains(joined, Label) {
			t.Errorf("launchctl calls: %q", calls)
		}
		if strings.Contains(joined, "bootout") {
			t.Errorf("bootout must never be called: %q", calls)
		}
	}
}

func TestInstallSkipsBootstrapWhenLoaded(t *testing.T) {
	old := launchctl
	defer func() { launchctl = old }()
	var calls []string
	launchctl = func(args ...string) error { calls = append(calls, strings.Join(args, " ")); return nil }
	if err := install(t.TempDir(), "/Applications/autowifi"); err != nil {
		t.Fatal(err)
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "bootstrap") || strings.HasPrefix(c, "bootout") {
			t.Errorf("loaded job must be left alone, got %q", calls)
		}
	}
}
