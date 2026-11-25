package creds

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// NewKeyring returns a Store backed by github.com/zalando/go-keyring under Service/Account.
// The secret is the JSON encoding of Credentials. keyring.ErrNotFound maps to ErrNotFound.
// A stored secret that is not valid JSON is treated as ErrNotFound.
// Tests call keyring.MockInit() first.
func NewKeyring() Store {
	return keyringStore{}
}

type keyringStore struct{}

func (keyringStore) Get() (Credentials, error) {
	raw, err := keyring.Get(Service, Account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Credentials{}, fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return Credentials{}, err
	}
	var c Credentials
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return Credentials{}, fmt.Errorf("%w: invalid json", ErrNotFound)
	}
	return c, nil
}

func (keyringStore) Set(c Credentials) error {
	norm, err := validateAndNormalize(c)
	if err != nil {
		return err
	}
	b, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return keyring.Set(Service, Account, string(b))
}

func (keyringStore) Delete() error {
	err := keyring.Delete(Service, Account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
