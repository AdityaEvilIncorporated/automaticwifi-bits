//go:build windows

package autostart

import (
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	runValueName = "AutoWiFiBITS"
)

func install(_ string, exe string) error {
	if !filepath.IsAbs(exe) {
		return fmt.Errorf("autostart: exe path must be absolute: %q", exe)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.SetStringValue(runValueName, RunValue(exe))
}

func uninstall(_ string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer k.Close()

	err = k.DeleteValue(runValueName)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

func installed(_ string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer k.Close()

	_, _, err = k.GetStringValue(runValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
