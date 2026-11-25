// Package notify sends quiet user notifications, at most one per hour (PROTOCOL.md section 8).
// Message texts are fixed and never include credentials, magic tokens or URLs.
package notify

import "time"

// Reason says why a notification is sent.
type Reason string

const (
	ReasonRejected       Reason = "rejected"        // portal rejected the stored credentials
	ReasonRepeatedErrors Reason = "repeated_errors" // 3rd consecutive portal_error
	ReasonFormChanged    Reason = "form_changed"    // portal page no longer has the expected form
	ReasonNoCredentials  Reason = "no_credentials"  // portal showing but nothing stored
)

// MinInterval is the minimum time between two notifications, across all reasons.
const MinInterval = time.Hour

// Notifier shows one notification. Implementations must not block for more than a few seconds.
type Notifier interface {
	Notify(title, body string) error
}

// Message returns the fixed title and body for r. Title is always "AutoWiFi BITS".
// Bodies (exact text, tests check them):
//   - ReasonRejected:       "Campus Wi-Fi login was rejected. Update your ID and password to resume."
//   - ReasonRepeatedErrors: "Campus Wi-Fi login keeps failing. Will keep retrying every 5 minutes."
//   - ReasonFormChanged:    "The campus login page has changed. Run 'autowifi capture' and report it."
//   - ReasonNoCredentials:  "Campus Wi-Fi needs a login. Set your ID and password in AutoWiFi BITS."
//
// An unknown reason returns the title and "AutoWiFi BITS needs attention."
func Message(r Reason) (title, body string) {
	panic("not implemented")
}

// Allow reports whether a notification may be sent at now, given the time of the last one
// (zero time = never). True when last is zero, or now-last >= MinInterval. A last in the
// future (clock moved back) also returns true, so a clock change can never silence
// notifications forever.
func Allow(last, now time.Time) bool {
	panic("not implemented")
}

// Nop is a Notifier that does nothing (used when notifications are unavailable).
type Nop struct{}

// Notify does nothing and returns nil.
func (Nop) Notify(title, body string) error { return nil }
