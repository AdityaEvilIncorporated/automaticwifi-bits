//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

func desktopPath(home string) string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "autowifi-bits.desktop")
}

func install(home, exe string) error {
	if !filepath.IsAbs(exe) {
		return fmt.Errorf("autostart: exe must be an absolute path: %q", exe)
	}
	p := desktopPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(DesktopEntry(exe)), 0o644)
}

func uninstall(home string) error {
	if err := os.Remove(desktopPath(home)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func installed(home string) (bool, error) {
	_, err := os.Stat(desktopPath(home))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
