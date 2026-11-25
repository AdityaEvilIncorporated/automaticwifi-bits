// Package atomicfile writes private files (credentials fallback, config, state) atomically.
package atomicfile

import (
	"os"
	"path/filepath"
)

// WriteFile writes data to path with mode 0600 (os.CreateTemp): a new uniquely named temp file in the same
// directory (O_EXCL, so a planted file or symlink is never written through), fsync, rename.
// The parent directory is created with mode 0700 if missing.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
