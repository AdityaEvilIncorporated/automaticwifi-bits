package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileCreatesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "creds.json")
	if err := WriteFile(path, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != `{"a":1}` {
		t.Fatalf("got %q, %v", b, err)
	}
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
			t.Errorf("mode %v, want 0600", st.Mode().Perm())
		}
		if st, _ := os.Stat(filepath.Dir(path)); st.Mode().Perm() != 0o700 {
			t.Errorf("dir mode %v, want 0700", st.Mode().Perm())
		}
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp*"))
	if len(left) != 0 {
		t.Errorf("temp files left: %v", left)
	}
}

// A planted path+".tmp" (wide mode, or a symlink to somewhere else) must not receive the data.
func TestWriteFileIgnoresPlantedTmp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	victim := filepath.Join(dir, "victim")
	if err := os.WriteFile(victim, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, path+".tmp"); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("secret")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(victim); len(b) != 0 {
		t.Errorf("data written through the planted symlink: %q", b)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", st.Mode().Perm())
	}
}

func TestWriteFileReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	for _, v := range []string{"one", "two"} {
		if err := WriteFile(path, []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	if b, _ := os.ReadFile(path); string(b) != "two" {
		t.Errorf("got %q", b)
	}
}
