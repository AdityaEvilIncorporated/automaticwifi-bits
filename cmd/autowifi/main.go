// Command autowifi is the desktop client (macOS, Windows, Linux).
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

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

// repo is OWNER/NAME for the update check, set at build time from RELEASE_REPO.
var repo = ""

func init() {
	// The tray (Cocoa on macOS) must run on the process's main thread.
	runtime.LockOSThread()
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] != "run" {
		attachConsole() // Windows GUI build: print to the terminal that started us
	}
	ctx := context.Background()
	if len(args) == 0 || args[0] == "run" || strings.HasPrefix(args[0], "-") {
		// Only the agent traps Ctrl-C/SIGTERM (to log out of nothing and exit cleanly); other
		// commands keep the default so Ctrl-C at the setup prompt exits at once.
		var stop context.CancelFunc
		ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
	}
	e, err := osEnv(ctx)
	if err != nil {
		os.Stderr.WriteString("autowifi: " + err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(realMain(args, e))
}
