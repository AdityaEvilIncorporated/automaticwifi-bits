// Command autowifi is the desktop client (macOS, Windows, Linux). HARD task T14.
//
//	autowifi run                 background agent (tray icon unless --no-tray); default when
//	                             started with no arguments by the login item
//	autowifi setup               ask for campus ID and password (terminal, or a native dialog
//	                             when there is no terminal) and store them in the OS store
//	autowifi login-now           one check + login in the foreground, prints the outcome
//	autowifi logout              log out of the current portal session
//	autowifi capture             print the portal form action, field names and TLS facts
//	                             (never values)
//	autowifi status              print on-campus, logged-in and last outcome
//	autowifi autostart on|off    register / remove the login item
//	autowifi forget              delete stored credentials
//	autowifi version
//
// Global flags: --config <path> (default config.DefaultPath()).
package main

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

// repo is OWNER/NAME for the update check, set at build time from RELEASE_REPO.
var repo = ""

func main() {
	panic("not implemented")
}
