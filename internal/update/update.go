// Package update checks GitHub Releases for a newer version (informational only: it never
// downloads or replaces anything).
package update

import (
	"context"
	"net/http"
)

// Release is the part of the GitHub "latest release" API reply we use.
type Release struct {
	Tag string `json:"tag_name"`
	URL string `json:"html_url"`
}

// Latest GETs apiURL (e.g. https://api.github.com/repos/OWNER/REPO/releases/latest) with
// header "Accept: application/vnd.github+json" and a User-Agent "autowifi-bits", reads at most
// 1 MiB, and decodes Release. Non-200 status → error mentioning the status code. An empty
// tag_name → error. client nil = http.DefaultClient.
func Latest(ctx context.Context, client *http.Client, apiURL string) (Release, error) {
	panic("not implemented")
}

// Newer reports whether latest is a strictly greater semantic version than current.
// Both may have a leading "v". Only MAJOR.MINOR.PATCH numbers are compared; a pre-release
// suffix ("-rc1") on latest makes it NOT newer (we only announce stable releases); a
// pre-release suffix on current is ignored for the comparison. Anything that does not parse
// (e.g. current "dev") returns false.
// Examples: ("v1.2.3","v1.2.4") true; ("1.2.3","v1.3.0") true; ("v1.2.3","v1.2.3") false;
// ("v1.10.0","v1.9.9") false; ("dev","v9.0.0") false; ("v1.0.0","v2.0.0-rc1") false.
func Newer(current, latest string) bool {
	panic("not implemented")
}
