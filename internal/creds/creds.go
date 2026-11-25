// Package creds stores the campus ID and password in the OS secure store (go-keyring:
// macOS Keychain, Windows Credential Manager, Linux Secret Service), with a mode-600 file
// fallback on Linux when no Secret Service is running.
package creds

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrNotFound is returned (possibly wrapped) by Get when nothing is stored.
var ErrNotFound = errors.New("credentials not found")

// ErrEmpty is returned by Set when username or password is empty after trimming spaces
// from the username (the password is stored exactly as given, never trimmed).
var ErrEmpty = errors.New("username and password are required")

// Service and Account are the go-keyring service and user names.
const (
	Service = "autowifi-bits"
	Account = "campus-login"
)

// Credentials is a campus ID and password. Its String, GoString and Format methods redact
// the password so it can never reach a log by accident.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// String returns "Credentials{Username: <username>, Password: [redacted]}"; when the password
// is empty it is "Credentials{Username: <username>, Password: []}".
func (c Credentials) String() string {
	if c.Password == "" {
		return fmt.Sprintf("Credentials{Username: %s, Password: []}", c.Username)
	}
	return fmt.Sprintf("Credentials{Username: %s, Password: [redacted]}", c.Username)
}

// GoString returns the same as String (so %#v is redacted too).
func (c Credentials) GoString() string {
	return c.String()
}

// Store is a credential store.
type Store interface {
	// Get returns the stored credentials or an error wrapping ErrNotFound.
	Get() (Credentials, error)
	// Set validates (ErrEmpty), trims the username and stores both.
	Set(Credentials) error
	// Delete removes them; deleting when nothing is stored returns nil.
	Delete() error
}

// NewMemory returns an in-memory Store (tests and the "no secure store" error path).
// Same validation as every Store. Safe for concurrent use.
func NewMemory() Store {
	return &memoryStore{}
}

func validateAndNormalize(c Credentials) (Credentials, error) {
	u := strings.TrimSpace(c.Username)
	if u == "" || c.Password == "" {
		return Credentials{}, ErrEmpty
	}
	return Credentials{Username: u, Password: c.Password}, nil
}

type memoryStore struct {
	mu    sync.Mutex
	creds *Credentials
}

func (m *memoryStore) Get() (Credentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.creds == nil {
		return Credentials{}, ErrNotFound
	}
	return *m.creds, nil
}

func (m *memoryStore) Set(c Credentials) error {
	norm, err := validateAndNormalize(c)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.creds = &norm
	return nil
}

func (m *memoryStore) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.creds = nil
	return nil
}
