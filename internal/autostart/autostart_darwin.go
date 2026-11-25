//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// launchctl runs /usr/bin/launchctl with args; a failure includes its stderr. Tests replace
// it so they never load a real agent. install bootstraps the job only when
// "print gui/<uid>/<Label>" says it is not loaded. Nothing ever calls bootout: the tray's
// "Start at login" toggle runs inside the launchd-started agent, and bootout would kill it.
var launchctl = func(args ...string) error {
	var stderr strings.Builder
	cmd := exec.Command("/usr/bin/launchctl", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launchctl %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func plistPath(home string) string {
	return filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
}

// plistPath is <home>/Library/LaunchAgents/<Label>.plist.
func install(home, exe string) error {
	if !filepath.IsAbs(exe) {
		return fmt.Errorf("autostart: exe must be an absolute path: %q", exe)
	}
	p := plistPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(LaunchAgentPlist(exe)), 0o644); err != nil {
		return err
	}
	uid := strconv.Itoa(os.Getuid())
	if launchctl("print", "gui/"+uid+"/"+Label) == nil {
		return nil // already loaded (e.g. we are that agent); launchd rereads the plist at login
	}
	// Starts the agent now (RunAtLoad). If one is already running it exits 0 at once, and
	// KeepAlive{SuccessfulExit:false} does not restart it.
	return launchctl("bootstrap", "gui/"+uid, p)
}

// uninstall removes the plist only: a running agent keeps running until logout or Quit, and
// is not started at the next login.
func uninstall(home string) error {
	if err := os.Remove(plistPath(home)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func installed(home string) (bool, error) {
	_, err := os.Stat(plistPath(home))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
