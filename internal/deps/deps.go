// Package deps pins the third-party modules the contracts need in go.mod, so workers never
// edit go.mod. Reasons are in ARCHITECTURE.md "Dependencies". Delete an import here once a
// real package imports that module.
package deps

import (
	_ "fyne.io/systray"
	_ "github.com/zalando/go-keyring"
	_ "golang.org/x/net/html"
	_ "golang.org/x/term"
)
