// Package config holds the portal configuration (protocol/PROTOCOL.md section 1).
//
// Every value is configurable so the campus portal can be confirmed with the
// `capture` command without a code change. The file lives at
// <os.UserConfigDir()>/autowifi-bits/config.json and only holds non-secret settings.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

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
	return PortalConfig{
		CheckURL:         "http://neverssl.com",
		FallbackCheckURL: "http://connectivitycheck.gstatic.com/generate_204",
		PortalHost:       "fw.bits-pilani.ac.in",
		PortalPort:       8090,
		LoginPath:        "/",
		UsernameField:    "username",
		PasswordField:    "password",
		MagicField:       "magic",
		RedirectField:    "4Tredir",
	}
}

func validateHTTPURL(key, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w: %s must be an absolute http:// or https:// URL with a host", ErrInvalid, key)
	}
	return nil
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
	if err := validateHTTPURL("check_url", c.CheckURL); err != nil {
		return err
	}
	if err := validateHTTPURL("fallback_check_url", c.FallbackCheckURL); err != nil {
		return err
	}
	if c.PortalHost == "" {
		return fmt.Errorf("%w: portal_host must not be empty", ErrInvalid)
	}
	for _, r := range c.PortalHost {
		if r == '/' || r == ':' || r == '@' || r == '?' || r == '#' || unicode.IsSpace(r) {
			return fmt.Errorf("%w: portal_host contains invalid character", ErrInvalid)
		}
	}
	if c.PortalPort < 1 || c.PortalPort > 65535 {
		return fmt.Errorf("%w: portal_port must be between 1 and 65535", ErrInvalid)
	}
	if !strings.HasPrefix(c.LoginPath, "/") {
		return fmt.Errorf("%w: login_path must start with /", ErrInvalid)
	}

	fields := []struct {
		key string
		val string
	}{
		{"username_field", c.UsernameField},
		{"password_field", c.PasswordField},
		{"magic_field", c.MagicField},
		{"redirect_field", c.RedirectField},
	}
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if f.val == "" {
			return fmt.Errorf("%w: %s must not be empty", ErrInvalid, f.key)
		}
		if _, dup := seen[f.val]; dup {
			return fmt.Errorf("%w: %s duplicates another field name", ErrInvalid, f.key)
		}
		seen[f.val] = struct{}{}
	}

	if c.PinnedCertSHA256 != "" {
		if len(c.PinnedCertSHA256) != 64 {
			return fmt.Errorf("%w: pinned_cert_sha256 must be 64 lowercase hex characters", ErrInvalid)
		}
		for _, r := range c.PinnedCertSHA256 {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return fmt.Errorf("%w: pinned_cert_sha256 must be 64 lowercase hex characters", ErrInvalid)
			}
		}
	}
	return nil
}

// PortalBase returns "https://{host}:{port}" (port always written, host lower-cased),
// e.g. "https://fw.bits-pilani.ac.in:8090".
func (c PortalConfig) PortalBase() string {
	return fmt.Sprintf("https://%s:%d", strings.ToLower(c.PortalHost), c.PortalPort)
}

// Load reads the JSON config at path, starting from Default() and overlaying the keys present
// in the file (missing keys keep their defaults; unknown keys are ignored).
// A missing file returns Default() and nil error. Malformed JSON returns an error wrapping
// ErrInvalid. The merged result is validated; a validation failure is returned as is.
func Load(path string) (PortalConfig, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return PortalConfig{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return PortalConfig{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := cfg.Validate(); err != nil {
		return PortalConfig{}, err
	}
	return cfg, nil
}

// Save writes cfg as indented JSON to path atomically (write to path+".tmp" in the same
// directory, then rename), creating the parent directory with mode 0700 if needed. The file
// mode is 0600. It validates first and writes nothing on a validation error.
func Save(path string, cfg PortalConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Dir returns <os.UserConfigDir()>/autowifi-bits. It does not create it.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "autowifi-bits"), nil
}

// DefaultPath returns filepath.Join(Dir(), "config.json").
func DefaultPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
