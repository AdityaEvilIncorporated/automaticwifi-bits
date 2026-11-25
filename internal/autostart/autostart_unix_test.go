//go:build darwin || linux

package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func regFile(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
	}
	return filepath.Join(home, ".config", "autostart", "autowifi-bits.desktop")
}

func TestInstallUninstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	restore := stubLaunchctl(t)
	defer restore()

	if err := Install(home, "relative/autowifi"); err == nil {
		t.Fatal("relative exe accepted")
	}
	if ok, err := Installed(home); err != nil || ok {
		t.Fatalf("Installed before install = %v %v", ok, err)
	}
	if err := Install(home, "/opt/autowifi"); err != nil {
		t.Fatal(err)
	}
	if err := Install(home, "/opt/autowifi"); err != nil {
		t.Fatalf("second install: %v", err)
	}
	b, err := os.ReadFile(regFile(home))
	if err != nil {
		t.Fatal(err)
	}
	want := DesktopEntry("/opt/autowifi")
	if runtime.GOOS == "darwin" {
		want = LaunchAgentPlist("/opt/autowifi")
	}
	if string(b) != want {
		t.Fatalf("file content:\n%s", b)
	}
	if ok, _ := Installed(home); !ok {
		t.Fatal("Installed false after install")
	}
	if err := Uninstall(home); err != nil {
		t.Fatal(err)
	}
	if ok, _ := Installed(home); ok {
		t.Fatal("still installed")
	}
	if err := Uninstall(home); err != nil {
		t.Fatalf("second uninstall: %v", err)
	}
}

func TestXDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if err := Install(home, "/opt/autowifi"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(xdg, "autostart", "autowifi-bits.desktop")); err != nil {
		t.Fatal(err)
	}
}
