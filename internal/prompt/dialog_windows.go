//go:build windows

package prompt

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"autowifi/internal/creds"
)

func dialog() (creds.Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", credScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// Cancel is not an error (empty output); a non-zero exit means the dialog failed.
		return creds.Credentials{}, fmt.Errorf("prompt: credential dialog failed: %v: %s", err, firstLine(string(ee.Stderr)))
	}
	if err != nil {
		return creds.Credentials{}, err
	}
	return parseWindows(string(out))
}
