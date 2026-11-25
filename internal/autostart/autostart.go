// Package autostart registers the agent to start at login for the current user only
// (never system-wide, never needs admin rights).
//   - darwin:  ~/Library/LaunchAgents/<Label>.plist (LaunchAgentPlist), loaded with
//     `launchctl bootstrap gui/<uid> <plist>` (an "already loaded" error is ignored); Uninstall
//     runs `launchctl bootout gui/<uid>/<Label>` (errors ignored) and removes the file.
//   - linux:   ~/.config/autostart/autowifi-bits.desktop (DesktopEntry); XDG_CONFIG_HOME is
//     honoured when set.
//   - windows: HKCU\Software\Microsoft\Windows\CurrentVersion\Run, value name "AutoWiFiBITS",
//     data RunValue(exe).
package autostart

// Label is the launchd label and the base name of the plist.
const Label = "io.github.autowifi-bits.agent"

// LaunchAgentPlist returns the plist XML for exe: Label, ProgramArguments [exe, "run"],
// RunAtLoad true, KeepAlive {SuccessfulExit: false}, ProcessType "Background",
// LimitLoadToSessionType "Aqua", no StandardOutPath/StandardErrorPath. exe is XML-escaped
// (& < > " ').
func LaunchAgentPlist(exe string) string {
	panic("not implemented")
}

// DesktopEntry returns the XDG autostart file for exe:
//
//	[Desktop Entry]
//	Type=Application
//	Name=AutoWiFi BITS
//	Comment=Logs in to BITS campus Wi-Fi automatically
//	Exec=<exe quoted per the Desktop Entry spec> run
//	Terminal=false
//	NoDisplay=true
//	X-GNOME-Autostart-enabled=true
//
// Quoting: always wrap exe in double quotes and backslash-escape " ` $ and \ inside it.
// Lines end with "\n", including the last.
func DesktopEntry(exe string) string {
	panic("not implemented")
}

// RunValue returns the Windows Run-key data: `"<exe>" run` (exe in double quotes).
func RunValue(exe string) string {
	panic("not implemented")
}

// Install registers exe (an absolute path; error otherwise) to start at login. Idempotent.
// home is the user's home directory (os.UserHomeDir in production; tests pass t.TempDir()).
// On windows home is unused.
func Install(home, exe string) error {
	return install(home, exe)
}

// Uninstall removes the registration. Not installed: nil.
func Uninstall(home string) error {
	return uninstall(home)
}

// Installed reports whether a registration exists (file present / Run value present).
func Installed(home string) (bool, error) {
	return installed(home)
}
