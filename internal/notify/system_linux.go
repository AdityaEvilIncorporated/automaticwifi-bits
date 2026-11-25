//go:build linux

package notify

import (
	"context"
	"os/exec"
	"time"
)

type linuxNotifier struct{}

// newSystem: see NewSystem in system.go for the exact command on linux.
func newSystem() Notifier {
	return linuxNotifier{}
}

func (linuxNotifier) Notify(title, body string) error {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		path,
		"--app-name=AutoWiFi-BITS",
		"--urgency=low",
		title,
		body,
	)
	return cmd.Run()
}
