//go:build darwin

package notify

import (
	"context"
	"os/exec"
	"time"
)

type darwinNotifier struct{}

// newSystem: see NewSystem in system.go for the exact command on darwin.
func newSystem() Notifier {
	return darwinNotifier{}
}

func (darwinNotifier) Notify(title, body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"/usr/bin/osascript",
		"-e", "on run argv",
		"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
		"-e", "end run",
		title,
		body,
	)
	return cmd.Run()
}
