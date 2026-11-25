//go:build windows

package prompt

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"

	"autowifi/internal/creds"
)

// Get-Credential shows the standard Windows credential dialog. The result is written to stdout
// (a pipe to this process only) as "<user>\n<password>" in UTF-8; nothing is echoed or saved.
// No -NonInteractive: it disables Get-Credential's prompt. The OutputEncoding line is guarded
// in case the hidden console handle is unusable.
const credScript = `try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch {}
$c = Get-Credential -Message 'BITS campus ID and password for AutoWiFi BITS'
if ($c) { [Console]::Out.Write($c.UserName + "` + "`n" + `" + $c.GetNetworkCredential().Password) }`

func dialog() (creds.Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", credScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return creds.Credentials{}, ErrCancelled // Cancel in Get-Credential ends PowerShell with an error
	}
	if err != nil {
		return creds.Credentials{}, err
	}
	return parseWindows(string(out))
}
