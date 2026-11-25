package main

import (
	"os"
	"path/filepath"
	"sync"
)

// rotWriter appends to path (mode 0600) and, past max bytes, renames it to path+".1" (replacing
// the previous one) and starts a new file. Outcomes only are written to it; never secrets.
type rotWriter struct {
	mu   sync.Mutex
	path string
	max  int64
	f    *os.File
	size int64
}

func newRotWriter(path string, max int64) (*rotWriter, error) {
	w := &rotWriter{path: path, max: max}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return w, w.open()
}

func (w *rotWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.f, w.size = f, st.Size()
	return nil
}

func (w *rotWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+int64(len(b)) > w.max && w.size > 0 {
		w.f.Close()
		_ = os.Rename(w.path, w.path+".1")
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	n, err := w.f.Write(b)
	w.size += int64(n)
	return n, err
}

func (w *rotWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}
