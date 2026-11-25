package creds

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
	panic("not implemented")
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
	panic("not implemented")
}

// KeyringProbe is the real Probe using go-keyring.
func KeyringProbe() error {
	panic("not implemented")
}
