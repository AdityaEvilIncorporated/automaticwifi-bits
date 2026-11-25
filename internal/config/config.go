// Package config holds the portal configuration (protocol/PROTOCOL.md section 1).
//
// Every value is configurable so the campus portal can be confirmed with the
// `capture` command without a code change. The file lives at
// <os.UserConfigDir()>/autowifi-bits/config.json and only holds non-secret settings.
package config

import "errors"

// ErrInvalid is wrapped by every validation error returned by Validate and Load.
var ErrInvalid = errors.New("invalid config")

// PortalConfig is the JSON-serialisable portal configuration. JSON keys match PROTOCOL.md.
type PortalConfig struct {
	CheckURL         string `json:"check_url"`
	FallbackCheckURL string `json:"fallback_check_url"`
	PortalHost       string `json:"portal_host"`
	PortalPort       int    `json:"portal_port"`
	LoginPath        string `json:"login_path"`
	UsernameField    string `json:"username_field"`
	PasswordField    string `json:"password_field"`
	MagicField       string `json:"magic_field"`
	RedirectField    string `json:"redirect_field"`
	// PinnedCertSHA256 is lowercase hex SHA-256 of the portal leaf SubjectPublicKeyInfo, or "".
	PinnedCertSHA256 string `json:"pinned_cert_sha256"`
}

// Default returns the standard FortiGate defaults from PROTOCOL.md section 1:
// check_url http://neverssl.com, fallback http://connectivitycheck.gstatic.com/generate_204,
// host fw.bits-pilani.ac.in, port 8090, login_path "/", fields username/password/magic/4Tredir,
// no pin.
func Default() PortalConfig {
	panic("not implemented")
}

// Validate checks the config. It returns an error wrapping ErrInvalid (use fmt.Errorf("%w: ...")
// and name the offending JSON key in the message) when:
//   - check_url or fallback_check_url is not an absolute http:// or https:// URL with a host
//   - portal_host is empty, or contains any of / : @ ? # or whitespace (so no scheme, port,
//     path or userinfo; IPv6 literals are not supported)
//   - portal_port is outside 1..65535
//   - login_path does not start with "/"
//   - any of the four field names is empty, or two of them are equal
//   - pinned_cert_sha256 is non-empty and not exactly 64 lowercase hex characters
func (c PortalConfig) Validate() error {
	panic("not implemented")
}

// PortalBase returns "https://{host}:{port}" (port always written, host lower-cased),
// e.g. "https://fw.bits-pilani.ac.in:8090".
func (c PortalConfig) PortalBase() string {
	panic("not implemented")
}

// Load reads the JSON config at path, starting from Default() and overlaying the keys present
// in the file (missing keys keep their defaults; unknown keys are ignored).
// A missing file returns Default() and nil error. Malformed JSON returns an error wrapping
// ErrInvalid. The merged result is validated; a validation failure is returned as is.
func Load(path string) (PortalConfig, error) {
	panic("not implemented")
}

// Save writes cfg as indented JSON to path atomically (write to path+".tmp" in the same
// directory, then rename), creating the parent directory with mode 0700 if needed. The file
// mode is 0600. It validates first and writes nothing on a validation error.
func Save(path string, cfg PortalConfig) error {
	panic("not implemented")
}

// Dir returns <os.UserConfigDir()>/autowifi-bits. It does not create it.
func Dir() (string, error) {
	panic("not implemented")
}

// DefaultPath returns filepath.Join(Dir(), "config.json").
func DefaultPath() (string, error) {
	panic("not implemented")
}
