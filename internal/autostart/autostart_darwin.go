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

// launchctl runs /usr/bin/launchctl with args. Tests replace it so they never load a real
// agent. install calls launchctl("bootstrap", "gui/<uid>", plistPath) after writing the file;
// uninstall calls launchctl("bootout", "gui/<uid>/"+Label) before removing it.
var launchctl = func(args ...string) error {
	return exec.Command("/usr/bin/launchctl", args...).Run()
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
	_ = launchctl("bootout", "gui/"+uid+"/"+Label)
	if err := launchctl("bootstrap", "gui/"+uid, p); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already") {
		return err
	}
	return nil
}

func uninstall(home string) error {
	uid := strconv.Itoa(os.Getuid())
	_ = launchctl("bootout", "gui/"+uid+"/"+Label)
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
