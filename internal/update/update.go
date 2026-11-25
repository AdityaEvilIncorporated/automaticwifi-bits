// Package update checks GitHub Releases for a newer version (informational only: it never
// downloads or replaces anything).
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "autowifi-bits")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return Release{}, err
	}
	if rel.Tag == "" {
		return Release{}, errors.New("github releases: empty tag_name")
	}
	return rel, nil
}

// Newer reports whether latest is a strictly greater semantic version than current.
// Both may have a leading "v". Only MAJOR.MINOR.PATCH numbers are compared; a pre-release
// suffix ("-rc1") on latest makes it NOT newer (we only announce stable releases); a
// pre-release suffix on current is ignored for the comparison. Anything that does not parse
// (e.g. current "dev") returns false.
// Examples: ("v1.2.3","v1.2.4") true; ("1.2.3","v1.3.0") true; ("v1.2.3","v1.2.3") false;
// ("v1.10.0","v1.9.9") false; ("dev","v9.0.0") false; ("v1.0.0","v2.0.0-rc1") false.
func Newer(current, latest string) bool {
	latest = strings.TrimPrefix(latest, "v")
	if strings.Contains(latest, "-") {
		return false
	}
	current = strings.TrimPrefix(current, "v")
	if base, _, ok := strings.Cut(current, "-"); ok {
		current = base
	}

	c, ok := parseSemver(current)
	if !ok {
		return false
	}
	l, ok := parseSemver(latest)
	if !ok {
		return false
	}

	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseSemver(s string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
