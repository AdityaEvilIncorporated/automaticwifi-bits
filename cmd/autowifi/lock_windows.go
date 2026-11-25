//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// acquireLock uses a named mutex in the session namespace, one per lock path. ok=false means
// another process holds it. Windows releases it if the process dies.
func acquireLock(path string) (release func(), ok bool, err error) {
	sum := sha256.Sum256([]byte(strings.ToLower(path)))
	name, err := windows.UTF16PtrFromString(`Local\AutoWiFiBITS-` + hex.EncodeToString(sum[:8]))
	if err != nil {
		return nil, false, err
	}
	h, err := windows.CreateMutex(nil, true, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return func() { _ = windows.ReleaseMutex(h); _ = windows.CloseHandle(h) }, true, nil
}

// attachConsole makes a -H windowsgui build print to the terminal that started it (cmd.exe,
// PowerShell). Started from Explorer there is no parent console and nothing changes.
func attachConsole() {
	const attachParentProcess = ^uintptr(0) // (DWORD)-1
	r, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole").Call(attachParentProcess)
	if r == 0 {
		return
	}
	if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = out, out
	}
	if in, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = in
	}
}
