// Package docscheck checks the user-facing documents for required content (T30, T31).
package docscheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustContain(t *testing.T, doc, name string, needles ...string) {
	t.Helper()
	low := strings.ToLower(doc)
	for _, n := range needles {
		if !strings.Contains(low, strings.ToLower(n)) {
			t.Errorf("%s must mention %q", name, n)
		}
	}
}

// Assets are the release file names (scripts/build-release.sh produces exactly these).
var Assets = []string{
	"autowifi-macos.zip",
	"autowifi-windows-amd64.zip",
	"autowifi-windows-arm64.zip",
	"autowifi-linux-amd64.tar.gz",
	"autowifi-linux-arm64.tar.gz",
	"autowifi-bits.apk",
	"SHA256SUMS.txt",
}

func TestShortcutGuide(t *testing.T) {
	g := read(t, "ios/SHORTCUT_GUIDE.md")
	mustContain(t, g, "ios/SHORTCUT_GUIDE.md",
		"Personal Automation", "Wi-Fi", "Run Immediately",
		"http://neverssl.com", "fgtauth", "Get Contents of URL", "POST", "Form",
		"username", "password", "magic", "4Tredir", "https://fw.bits-pilani.ac.in:8090/",
		"Match Text", "[0-9A-Za-z]+",
		"Text action", "iCloud", "Advanced Data Protection",
		"SSID", "capture", "keepalive")
	// No real-looking credentials: the only allowed values are the placeholders.
	if regexp.MustCompile(`(?i)\bf20\d{2}\d{4}\b`).MatchString(g) {
		t.Error("guide contains something that looks like a real BITS ID; use YOUR_ID")
	}
	mustContain(t, g, "ios/SHORTCUT_GUIDE.md", "YOUR_ID", "YOUR_PASSWORD")
}

func TestDownloadPage(t *testing.T) {
	page := read(t, "site/index.html")
	repo := strings.TrimSpace(read(t, "RELEASE_REPO"))
	// Links name the tag in RELEASE_TAG: /releases/latest/ skips pre-releases, so it would 404
	// while only beta builds exist. release.yml refuses a tag that differs from RELEASE_TAG.
	tag := strings.TrimSpace(read(t, "RELEASE_TAG"))
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`).MatchString(tag) {
		t.Fatalf("RELEASE_TAG %q is not vMAJOR.MINOR.PATCH", tag)
	}
	if strings.Contains(page, "/releases/latest/") {
		t.Error("site/index.html must link the tag in RELEASE_TAG, not /releases/latest/")
	}
	// While RELEASE_BETA exists the page must carry its label and point at the checklist.
	if beta, err := os.ReadFile(filepath.Join("..", "..", "RELEASE_BETA")); err == nil {
		mustContain(t, page, "site/index.html", tag+" pre-release", strings.TrimSpace(string(beta)),
			`href="#campus-checklist"`, `id="campus-checklist"`)
	}
	for _, a := range Assets {
		want := "https://github.com/" + repo + "/releases/download/" + tag + "/" + a
		if !strings.Contains(page, want) {
			t.Errorf("site/index.html must link %s", want)
		}
	}
	mustContain(t, page, "site/index.html",
		"<title>AutoWiFi BITS</title>", `name="viewport"`,
		"Gatekeeper", "Open", "SmartScreen", "More info", "Run anyway",
		"Install unknown apps", "battery",
		"SHORTCUT_GUIDE", "autowifi setup", "autowifi capture",
		"Campus verification checklist", "portal fields", "SSIDs", "certificate",
		"https://github.com/"+repo)
	for _, bad := range []string{"<script src=\"http", "password=", "value=\"f20"} {
		if strings.Contains(page, bad) {
			t.Errorf("site/index.html must not contain %q", bad)
		}
	}
	if strings.Contains(page, "<script") {
		t.Error("site/index.html must be static (no scripts)")
	}
}
