package creds

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// NewFile returns a Store that keeps the JSON-encoded Credentials in the file at path.
//   - Set writes atomically (path+".tmp", then rename) with mode 0600 and creates the parent
//     directory with mode 0700.
//   - Get on a missing file returns ErrNotFound. When the file's permission bits allow any
//     group or other access (mode & 0o077 != 0) Get first chmods it to 0600; if that fails it
//     returns an error and does not read the file.
//   - Delete removes the file (missing file: nil).
//
// Path for the Linux fallback: ~/.config/autowifi-bits/credentials.
func NewFile(path string) Store {
	return &fileStore{path: path}
}

type fileStore struct {
	path string
}

func (f *fileStore) Get() (Credentials, error) {
	st, err := os.Stat(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrNotFound
		}
		return Credentials{}, err
	}
	if st.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(f.path, 0o600); err != nil {
			return Credentials{}, err
		}
	}
	b, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrNotFound
		}
		return Credentials{}, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		// Never return the JSON error: its message can quote bytes of the file (the password).
		return Credentials{}, fmt.Errorf("%w: credentials file is not valid JSON", ErrNotFound)
	}
	return c, nil
}

func (f *fileStore) Set(c Credentials) error {
	norm, err := validateAndNormalize(c)
	if err != nil {
		return err
	}
	b, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (f *fileStore) Delete() error {
	err := os.Remove(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Probe reports whether the OS keyring works: it Sets, Gets and Deletes a throwaway secret
// under Service and account "probe". Used by NewDefault.
type Probe func() error

// NewDefault picks the store for this machine: the keyring when probe() returns nil;
// otherwise (only expected on Linux without Secret Service) NewFile(fallbackPath), and it
// calls warn exactly once per machine with the message
// "No Secret Service keyring found; storing credentials in <fallbackPath> (mode 600)."
// "Once per machine" is tracked by the marker file fallbackPath+".warned" (mode 0600):
// warn is called only if the marker does not exist, then the marker is created.
func NewDefault(probe Probe, fallbackPath string, warn func(string)) Store {
	if probe == nil {
		probe = KeyringProbe
	}
	if probe() == nil {
		return NewKeyring()
	}
	marker := fallbackPath + ".warned"
	if _, err := os.Stat(marker); errors.Is(err, os.ErrNotExist) {
		if warn != nil {
			warn(fmt.Sprintf("No Secret Service keyring found; storing credentials in %s (mode 600).", fallbackPath))
		}
		_ = os.MkdirAll(filepath.Dir(marker), 0o700)
		_ = os.WriteFile(marker, nil, 0o600)
	}
	return NewFile(fallbackPath)
}

// KeyringProbe is the real Probe using go-keyring.
func KeyringProbe() error {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return err
	}
	val := hex.EncodeToString(buf[:])
	if err := keyring.Set(Service, "probe", val); err != nil {
		return err
	}
	if _, err := keyring.Get(Service, "probe"); err != nil {
		_ = keyring.Delete(Service, "probe")
		return err
	}
	if err := keyring.Delete(Service, "probe"); err != nil {
		return err
	}
	return nil
}
