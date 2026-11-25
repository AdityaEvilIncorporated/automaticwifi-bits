// Package tray is the tray / menu-bar icon (fyne.io/systray). Run must be called on the main
// goroutine (a macOS requirement) and blocks until Quit.
package tray

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"fyne.io/systray"

	"autowifi/internal/portal"
)

// Handlers are called from the tray's own goroutine; they must not block for long.
type Handlers struct {
	LoginNow       func()
	Logout         func()
	SetCredentials func()
	Quit           func()
	// Autostart reports whether start-at-login is on; SetAutostart changes it.
	Autostart    func() bool
	SetAutostart func(on bool) error
}

// Tray is the running menu. Its methods are safe to call from any goroutine.
type Tray struct {
	mu        sync.Mutex
	status    *systray.MenuItem
	update    *systray.MenuItem
	updateURL string
}

// Run shows the icon and menu, calls onReady with the Tray, and blocks until Quit.
func Run(h Handlers, onReady func(*Tray)) {
	systray.Run(func() {
		t := build(h)
		if onReady != nil {
			onReady(t)
		}
	}, func() {})
}

// Quit removes the icon and makes Run return.
func Quit() { systray.Quit() }

func build(h Handlers) *Tray {
	setIcon()
	systray.SetTooltip("AutoWiFi BITS")
	t := &Tray{}
	t.status = systray.AddMenuItem("Starting…", "")
	t.status.Disable()
	systray.AddSeparator()
	login := systray.AddMenuItem("Log in now", "Check the campus portal and log in")
	logout := systray.AddMenuItem("Log out", "End the campus Wi-Fi session")
	setCreds := systray.AddMenuItem("Set ID and password…", "Store your campus ID and password")
	auto := systray.AddMenuItemCheckbox("Start at login", "Run AutoWiFi BITS when you log in", h.Autostart != nil && h.Autostart())
	t.update = systray.AddMenuItem("", "Open the release page")
	t.update.Hide()
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Stop AutoWiFi BITS")

	go func() {
		for {
			select {
			case <-login.ClickedCh:
				call(h.LoginNow)
			case <-logout.ClickedCh:
				call(h.Logout)
			case <-setCreds.ClickedCh:
				call(h.SetCredentials)
			case <-auto.ClickedCh:
				if h.SetAutostart == nil {
					continue
				}
				want := !auto.Checked()
				if err := h.SetAutostart(want); err == nil {
					if want {
						auto.Check()
					} else {
						auto.Uncheck()
					}
				}
			case <-t.update.ClickedCh:
				t.mu.Lock()
				u := t.updateURL
				t.mu.Unlock()
				_ = OpenURL(u)
			case <-quit.ClickedCh:
				call(h.Quit)
				systray.Quit()
				return
			}
		}
	}()
	return t
}

func call(f func()) {
	if f != nil {
		f()
	}
}

// SetStatus shows the outcome of the last check as the first (disabled) menu line.
func (t *Tray) SetStatus(outcome portal.Outcome) {
	text := StatusText(outcome)
	t.status.SetTitle(text)
	systray.SetTooltip("AutoWiFi BITS: " + text)
}

// SetUpdate shows "Update available: <tag>" linking to url (only github.com release pages).
func (t *Tray) SetUpdate(tag, url string) {
	if !IsReleaseURL(url) {
		return
	}
	t.mu.Lock()
	t.updateURL = url
	t.mu.Unlock()
	t.update.SetTitle("Update available: " + tag)
	t.update.Show()
}

// StatusText is the human wording for an engine/agent outcome.
func StatusText(o portal.Outcome) string {
	switch o {
	case portal.LoggedIn, "keepalive_ok":
		return "Logged in to campus Wi-Fi"
	case portal.Online:
		return "Online"
	case portal.OffCampus:
		return "Not on the campus network"
	case portal.Foreign:
		return "On another network"
	case portal.Rejected, "paused":
		return "ID or password rejected: set them again"
	case portal.NoCredentials:
		return "ID and password needed"
	case portal.FormChanged:
		return "Login page changed: run 'autowifi capture'"
	case portal.PortalError:
		return "Portal not responding: retrying"
	case "logged_out":
		return "Logged out"
	}
	return "Working…"
}

// IsReleaseURL reports whether u is an https://github.com/ page (the only kind we open).
func IsReleaseURL(u string) bool {
	return strings.HasPrefix(u, "https://github.com/") && !strings.ContainsAny(u, " \t\r\n\"'`")
}

// OpenURL opens a GitHub release page in the default browser. Any other URL is refused.
func OpenURL(u string) error {
	if !IsReleaseURL(u) {
		return errors.New("tray: refusing to open a non-GitHub URL")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("/usr/bin/open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap it: no zombie per click
	return nil
}
