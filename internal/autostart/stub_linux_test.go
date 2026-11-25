//go:build linux

package autostart

import "testing"

func stubLaunchctl(t *testing.T) func() { return func() {} }
