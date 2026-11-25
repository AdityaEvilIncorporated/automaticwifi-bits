// Package state persists the small SessionState that must survive a restart (rejected
// credentials, the hourly notification limit, the current keepalive session).
// The file is <config.Dir()>/state.json, mode 0600, because the keepalive URL carries a
// session token.
package state

import "time"

// SessionState is the persisted session. JSON keys match the spec's data model.
type SessionState struct {
	MagicToken          string    `json:"magic_token"`
	KeepaliveURL        string    `json:"keepalive_url"`
	LogoutURL           string    `json:"logout_url"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastNotificationAt  time.Time `json:"last_notification_at"`
	CredentialsRejected bool      `json:"credentials_rejected"`
}

// LoggedIn reports whether a keepalive session is held (KeepaliveURL != "").
func (s SessionState) LoggedIn() bool {
	panic("not implemented")
}

// ClearSession empties MagicToken, KeepaliveURL and LogoutURL and keeps everything else.
func (s *SessionState) ClearSession() {
	panic("not implemented")
}

// Load reads the state file. A missing file returns the zero SessionState and nil.
// A corrupt file (bad JSON) also returns the zero SessionState and nil: losing this state is
// harmless and must never stop the agent. Other read errors (e.g. permission) are returned.
func Load(path string) (SessionState, error) {
	panic("not implemented")
}

// Save writes s as JSON atomically (path+".tmp" then rename), mode 0600, creating the parent
// directory with mode 0700 if needed.
func Save(path string, s SessionState) error {
	panic("not implemented")
}
