//go:build darwin

package autostart

import (
	"strings"
	"testing"
)

func stubLaunchctl(t *testing.T) func() {
	old := launchctl
	var calls []string
	launchctl = func(args ...string) error { calls = append(calls, strings.Join(args, " ")); return nil }
	return func() {
		launchctl = old
		joined := strings.Join(calls, "\n")
		if !strings.Contains(joined, "bootstrap gui/") || !strings.Contains(joined, "bootout gui/") || !strings.Contains(joined, Label) {
			t.Errorf("launchctl calls: %q", calls)
		}
	}
}
