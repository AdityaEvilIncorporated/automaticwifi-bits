//go:build darwin

package notify

// newSystem: see NewSystem in system.go for the exact command on darwin.
func newSystem() Notifier {
	panic("not implemented")
}
