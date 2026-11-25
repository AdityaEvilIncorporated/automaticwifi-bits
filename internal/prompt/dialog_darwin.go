//go:build darwin

package prompt

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"autowifi/internal/creds"
)

// The prompt texts are passed as arguments (argv), never spliced into the AppleScript source.
const askScript = `on run argv
	if item 2 of argv is "hidden" then
		return text returned of (display dialog (item 1 of argv) default answer "" with title "AutoWiFi BITS" with hidden answer)
	end if
	return text returned of (display dialog (item 1 of argv) default answer "" with title "AutoWiFi BITS")
end run`

func ask(text, mode string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", askScript, text, mode).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && strings.Contains(string(ee.Stderr), "-128") {
			return "", ErrCancelled
		}
		return "", err
	}
	return trimOne(string(out)), nil
}

func dialog() (creds.Credentials, error) {
	user, err := ask("BITS campus ID:", "plain")
	if err != nil {
		return creds.Credentials{}, err
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return creds.Credentials{}, ErrCancelled
	}
	pw, err := ask("Password for "+user+":", "hidden")
	if err != nil {
		return creds.Credentials{}, err
	}
	if pw == "" {
		return creds.Credentials{}, ErrCancelled
	}
	return creds.Credentials{Username: user, Password: pw}, nil
}
