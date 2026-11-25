// Package state persists the small SessionState that must survive a restart (rejected
// credentials, the hourly notification limit, the current keepalive session).
// The file is <config.Dir()>/state.json, mode 0600, because the keepalive URL carries a
// session token.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

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
	return s.KeepaliveURL != ""
}

// ClearSession empties MagicToken, KeepaliveURL and LogoutURL and keeps everything else.
func (s *SessionState) ClearSession() {
	s.MagicToken = ""
	s.KeepaliveURL = ""
	s.LogoutURL = ""
}

// Load reads the state file. A missing file returns the zero SessionState and nil.
// A corrupt file (bad JSON) also returns the zero SessionState and nil: losing this state is
// harmless and must never stop the agent. Other read errors (e.g. permission) are returned.
func Load(path string) (SessionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SessionState{}, nil
		}
		return SessionState{}, err
	}
	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return SessionState{}, nil
	}
	return s, nil
}

// Save writes s as JSON atomically (path+".tmp" then rename), mode 0600, creating the parent
// directory with mode 0700 if needed.
func Save(path string, s SessionState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
