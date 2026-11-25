package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	want := PortalConfig{
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
	if c != want {
		t.Fatalf("Default() = %+v, want %+v", c, want)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v", err)
	}
}

func TestPortalBase(t *testing.T) {
	c := Default()
	if got := c.PortalBase(); got != "https://fw.bits-pilani.ac.in:8090" {
		t.Fatalf("PortalBase = %q", got)
	}
	c.PortalHost, c.PortalPort = "FW.Example.ORG", 443
	if got := c.PortalBase(); got != "https://fw.example.org:443" {
		t.Fatalf("PortalBase = %q", got)
	}
}

func TestValidateRejects(t *testing.T) {
	pin := strings.Repeat("a", 64)
	cases := map[string]func(*PortalConfig){
		"check_url relative":       func(c *PortalConfig) { c.CheckURL = "neverssl.com" },
		"check_url ftp":            func(c *PortalConfig) { c.CheckURL = "ftp://neverssl.com" },
		"check_url no host":        func(c *PortalConfig) { c.CheckURL = "http://" },
		"fallback empty":           func(c *PortalConfig) { c.FallbackCheckURL = "" },
		"host empty":               func(c *PortalConfig) { c.PortalHost = "" },
		"host with scheme":         func(c *PortalConfig) { c.PortalHost = "https://fw.bits-pilani.ac.in" },
		"host with port":           func(c *PortalConfig) { c.PortalHost = "fw.bits-pilani.ac.in:8090" },
		"host with path":           func(c *PortalConfig) { c.PortalHost = "fw.bits-pilani.ac.in/x" },
		"host with userinfo":       func(c *PortalConfig) { c.PortalHost = "a@fw.bits-pilani.ac.in" },
		"host with space":          func(c *PortalConfig) { c.PortalHost = "fw bits" },
		"port zero":                func(c *PortalConfig) { c.PortalPort = 0 },
		"port too big":             func(c *PortalConfig) { c.PortalPort = 65536 },
		"login_path no slash":      func(c *PortalConfig) { c.LoginPath = "login" },
		"username_field empty":     func(c *PortalConfig) { c.UsernameField = "" },
		"password_field empty":     func(c *PortalConfig) { c.PasswordField = "" },
		"magic_field empty":        func(c *PortalConfig) { c.MagicField = "" },
		"redirect_field empty":     func(c *PortalConfig) { c.RedirectField = "" },
		"duplicate field names":    func(c *PortalConfig) { c.PasswordField = "username" },
		"pin too short":            func(c *PortalConfig) { c.PinnedCertSHA256 = "abcd" },
		"pin uppercase":            func(c *PortalConfig) { c.PinnedCertSHA256 = strings.ToUpper(pin) },
		"pin not hex":              func(c *PortalConfig) { c.PinnedCertSHA256 = strings.Repeat("g", 64) },
		"pin with colons (len 64)": func(c *PortalConfig) { c.PinnedCertSHA256 = strings.Repeat("a:", 32) },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			c := Default()
			mut(&c)
			err := c.Validate()
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestValidateAccepts(t *testing.T) {
	c := Default()
	c.PinnedCertSHA256 = strings.Repeat("0f", 32)
	c.PortalHost = "127.0.0.1"
	c.CheckURL = "https://example.com/x?y=1"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateErrorNamesKey(t *testing.T) {
	c := Default()
	c.PortalPort = -1
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "portal_port") {
		t.Fatalf("error %v should name portal_port", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil || c != Default() {
		t.Fatalf("Load(missing) = %+v, %v", c, err)
	}
}

func TestLoadOverlay(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(`{"username_field":"uid","portal_port":9443,"unknown_key":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.UsernameField, want.PortalPort = "uid", 9443
	if c != want {
		t.Fatalf("got %+v want %+v", c, want)
	}
}

func TestLoadBadJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"portal_port":`), 0o600)
	if _, err := Load(p); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"portal_port":70000}`), 0o600)
	if _, err := Load(p); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestSaveRoundTripAndMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	p := filepath.Join(dir, "config.json")
	c := Default()
	c.MagicField = "tok"
	if err := Save(p, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil || got != c {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file left behind: %v", err)
	}
	if runtime.GOOS != "windows" {
		st, _ := os.Stat(p)
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", st.Mode().Perm())
		}
		dst, _ := os.Stat(dir)
		if dst.Mode().Perm() != 0o700 {
			t.Fatalf("dir mode = %v, want 0700", dst.Mode().Perm())
		}
	}
}

func TestSaveRefusesInvalid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	c := Default()
	c.PortalHost = ""
	if err := Save(p, c); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("file written despite invalid config")
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Skip("no user config dir:", err)
	}
	if filepath.Base(p) != "config.json" || filepath.Base(filepath.Dir(p)) != "autowifi-bits" {
		t.Fatalf("DefaultPath = %q", p)
	}
	d, _ := Dir()
	if filepath.Dir(p) != d {
		t.Fatalf("Dir = %q, DefaultPath = %q", d, p)
	}
}
