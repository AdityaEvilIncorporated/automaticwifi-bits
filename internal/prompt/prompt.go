// Package prompt asks the student for their campus ID and password: in a terminal (password
// not echoed) or, when there is no terminal (tray, double-click on Windows), with a native
// dialog. The values only ever go to the caller; nothing here logs or stores them.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"autowifi/internal/creds"
)

// ErrCancelled is returned when the user closes or cancels the prompt.
var ErrCancelled = errors.New("cancelled")

// ErrNoDialog is returned on Linux when neither zenity nor kdialog is installed.
var ErrNoDialog = errors.New("no dialog tool found (install zenity or kdialog, or run 'autowifi setup' in a terminal)")

// Title is shown on every dialog.
const Title = "AutoWiFi BITS"

// Terminal prompts on a text terminal.
type Terminal struct {
	In  io.Reader
	Out io.Writer
	// ReadPassword reads one line without echo. Default: term.ReadPassword on stdin.
	ReadPassword func() (string, error)
}

// NewTerminal returns a Terminal on os.Stdin/os.Stdout.
func NewTerminal() Terminal {
	return Terminal{In: os.Stdin, Out: os.Stdout, ReadPassword: func() (string, error) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		return string(b), err
	}}
}

// IsTerminal reports whether stdin is an interactive terminal.
func IsTerminal() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// Ask prompts for the ID (echoed, spaces trimmed) and the password (not echoed, kept exactly).
// An empty ID or password, or end of input, returns ErrCancelled.
func (t Terminal) Ask() (creds.Credentials, error) {
	fmt.Fprint(t.Out, "BITS campus ID: ")
	line, err := bufio.NewReader(t.In).ReadString('\n')
	if err != nil && line == "" {
		return creds.Credentials{}, ErrCancelled
	}
	user := strings.TrimSpace(line)
	if user == "" {
		return creds.Credentials{}, ErrCancelled
	}
	fmt.Fprint(t.Out, "Password (not shown): ")
	pw, err := t.ReadPassword()
	fmt.Fprintln(t.Out)
	if err != nil {
		return creds.Credentials{}, fmt.Errorf("reading password: %w", err)
	}
	pw = strings.TrimRight(pw, "\r\n")
	if pw == "" {
		return creds.Credentials{}, ErrCancelled
	}
	return creds.Credentials{Username: user, Password: pw}, nil
}

// Dialog asks with the platform's native dialogs (osascript, PowerShell Get-Credential,
// zenity/kdialog). Credentials come back over the child process's stdout pipe only.
func Dialog() (creds.Credentials, error) {
	return dialog()
}

// trimOne removes one trailing "\n" (and a "\r" before it): dialog tools end their output
// with a newline, but a password may legitimately end in spaces.
func trimOne(s string) string {
	s = strings.TrimSuffix(s, "\n")
	return strings.TrimSuffix(s, "\r")
}

// parseWindows parses the PowerShell helper's output "<user>\n<password>". Get-Credential may
// prefix a bare user name with a backslash (empty domain); that is removed.
// credScript (Windows) shows the standard credential dialog through
// $Host.UI.PromptForCredential, which takes no parameter sets (Get-Credential -Message makes
// -UserName mandatory in Windows PowerShell 5.1 and then prompts on the hidden console). The
// result goes to stdout (a pipe to this process only) as "<user>\n<password>" in UTF-8; Cancel
// prints nothing. No -NonInteractive: it disables the prompt. The OutputEncoding line is
// guarded in case the hidden console handle is unusable. ErrorActionPreference Stop makes a
// failing dialog exit non-zero (Windows PowerShell 5.1 would otherwise continue and exit 0).
const credScript = `$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch {}
$c = $Host.UI.PromptForCredential('AutoWiFi BITS', 'BITS campus ID and password', '', '')
if ($c) { [Console]::Out.Write($c.UserName + "` + "`n" + `" + $c.GetNetworkCredential().Password) }`

// firstLine returns the first non-empty line of s, at most 200 bytes (error context only).
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if len(l) > 200 {
				l = l[:200]
			}
			return l
		}
	}
	return ""
}

func parseWindows(out string) (creds.Credentials, error) {
	user, pw, ok := strings.Cut(out, "\n")
	if !ok {
		return creds.Credentials{}, ErrCancelled
	}
	user = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(user), `\`))
	pw = trimOne(pw)
	if user == "" || pw == "" {
		return creds.Credentials{}, ErrCancelled
	}
	return creds.Credentials{Username: user, Password: pw}, nil
}
