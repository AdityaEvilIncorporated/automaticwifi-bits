package creds

// NewKeyring returns a Store backed by github.com/zalando/go-keyring under Service/Account.
// The secret is the JSON encoding of Credentials. keyring.ErrNotFound maps to ErrNotFound.
// A stored secret that is not valid JSON is treated as ErrNotFound.
// Tests call keyring.MockInit() first.
func NewKeyring() Store {
	panic("not implemented")
}
