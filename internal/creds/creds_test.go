package creds

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestRedaction(t *testing.T) {
	c := Credentials{Username: "f20250001", Password: "hunter2-secret"}
	for _, s := range []string{c.String(), c.GoString(), fmt.Sprint(c), fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), fmt.Sprintf("%s", c)} {
		if strings.Contains(s, "hunter2") {
			t.Fatalf("password leaked: %s", s)
		}
	}
	if c.String() != "Credentials{Username: f20250001, Password: [redacted]}" {
		t.Fatalf("String = %q", c.String())
	}
	if (Credentials{Username: "a"}).String() != "Credentials{Username: a, Password: []}" {
		t.Fatal("empty password form")
	}
	p := &c
	if strings.Contains(fmt.Sprintf("%v %+v", p, p), "hunter2") {
		t.Fatal("pointer formatting leaked")
	}
}

func storeContract(t *testing.T, s Store) {
	t.Helper()
	if _, err := s.Get(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty Get err = %v", err)
	}
	if err := s.Delete(); err != nil {
		t.Fatalf("Delete when empty: %v", err)
	}
	for _, bad := range []Credentials{{"", "p"}, {"   ", "p"}, {"u", ""}} {
		if err := s.Set(bad); !errors.Is(err, ErrEmpty) {
			t.Fatalf("Set(%v) err = %v", bad, err)
		}
	}
	if err := s.Set(Credentials{"  f20250001 ", " pass with spaces "}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get()
	if err != nil || got != (Credentials{"f20250001", " pass with spaces "}) {
		t.Fatalf("Get = %q/%q, %v", got.Username, got.Password, err)
	}
	if err := s.Set(Credentials{"f2", "new"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(); got != (Credentials{"f2", "new"}) {
		t.Fatalf("overwrite: %v", got)
	}
	if err := s.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after Delete err = %v", err)
	}
}

func TestMemoryStore(t *testing.T) { storeContract(t, NewMemory()) }

func TestKeyringStore(t *testing.T) {
	keyring.MockInit()
	storeContract(t, NewKeyring())
}

func TestKeyringCorruptSecret(t *testing.T) {
	keyring.MockInit()
	keyring.Set(Service, Account, "not json")
	if _, err := NewKeyring().Get(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestFileStore(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg", "credentials")
	storeContract(t, NewFile(p))
	s := NewFile(p)
	if err := s.Set(Credentials{"u", "p"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("tmp left")
	}
	if runtime.GOOS == "windows" {
		return
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
	dst, _ := os.Stat(filepath.Dir(p))
	if dst.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %v", dst.Mode().Perm())
	}
	os.Chmod(p, 0o644)
	if got, err := s.Get(); err != nil || got.Username != "u" {
		t.Fatalf("Get after chmod: %v %v", got, err)
	}
	st, _ = os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("Get did not tighten mode: %v", st.Mode().Perm())
	}
}

func TestNewDefault(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	fb := filepath.Join(dir, "credentials")
	var warnings []string
	warn := func(s string) { warnings = append(warnings, s) }

	s := NewDefault(func() error { return nil }, fb, warn)
	s.Set(Credentials{"k", "v"})
	if _, err := os.Stat(fb); !os.IsNotExist(err) || len(warnings) != 0 {
		t.Fatal("keyring path used the file or warned")
	}
	if got, _ := NewKeyring().Get(); got.Username != "k" {
		t.Fatal("keyring store not used")
	}

	broken := func() error { return errors.New("org.freedesktop.secrets not provided") }
	s = NewDefault(broken, fb, warn)
	if err := s.Set(Credentials{"f", "w"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fb); err != nil {
		t.Fatal("fallback file not written")
	}
	want := "No Secret Service keyring found; storing credentials in " + fb + " (mode 600)."
	if len(warnings) != 1 || warnings[0] != want {
		t.Fatalf("warnings %q", warnings)
	}
	NewDefault(broken, fb, warn)
	if len(warnings) != 1 {
		t.Fatal("warned twice")
	}
	if _, err := os.Stat(fb + ".warned"); err != nil {
		t.Fatal("marker missing")
	}
}

func TestKeyringProbe(t *testing.T) {
	keyring.MockInit()
	if err := KeyringProbe(); err != nil {
		t.Fatal(err)
	}
	keyring.MockInitWithError(errors.New("no dbus"))
	if err := KeyringProbe(); err == nil {
		t.Fatal("probe should fail")
	}
}
