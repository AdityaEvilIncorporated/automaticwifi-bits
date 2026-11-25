//go:build darwin

package autostart

// launchctl runs /usr/bin/launchctl with args. Tests replace it so they never load a real
// agent. install calls launchctl("bootstrap", "gui/<uid>", plistPath) after writing the file;
// uninstall calls launchctl("bootout", "gui/<uid>/"+Label) before removing it.
var launchctl = func(args ...string) error { panic("not implemented") }

// plistPath is <home>/Library/LaunchAgents/<Label>.plist.
func install(home, exe string) error { panic("not implemented") }

func uninstall(home string) error { panic("not implemented") }

func installed(home string) (bool, error) { panic("not implemented") }
