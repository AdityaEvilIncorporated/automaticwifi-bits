//go:build windows

package notify

// newSystem: see NewSystem in system.go for the exact command on windows.
func newSystem() Notifier {
	panic("not implemented")
}
