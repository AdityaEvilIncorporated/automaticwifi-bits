//go:build linux

package prompt

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"autowifi/internal/creds"
)

// run executes a prepared command (program names are literals at every call site).
func run(cmd *exec.Cmd) (string, error) {
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", ErrCancelled // zenity/kdialog exit 1 on Cancel
		}
		return "", err
	}
	return trimOne(string(out)), nil
}

func dialog() (creds.Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var user, pw string
	var err error
	switch {
	case lookPath("zenity"):
		if user, err = run(exec.CommandContext(ctx, "zenity", "--entry", "--title="+Title, "--text=BITS campus ID:")); err == nil {
			pw, err = run(exec.CommandContext(ctx, "zenity", "--password", "--title="+Title))
		}
	case lookPath("kdialog"):
		if user, err = run(exec.CommandContext(ctx, "kdialog", "--title", Title, "--inputbox", "BITS campus ID:")); err == nil {
			pw, err = run(exec.CommandContext(ctx, "kdialog", "--title", Title, "--password", "Password:"))
		}
	default:
		return creds.Credentials{}, ErrNoDialog
	}
	if err != nil {
		return creds.Credentials{}, err
	}
	user = strings.TrimSpace(user)
	if user == "" || pw == "" {
		return creds.Credentials{}, ErrCancelled
	}
	return creds.Credentials{Username: user, Password: pw}, nil
}

func lookPath(name string) bool { _, err := exec.LookPath(name); return err == nil }
