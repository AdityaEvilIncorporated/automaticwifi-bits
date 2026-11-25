package portal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"autowifi/internal/config"
)

// loadCases reads protocol/testcases/<file> into out (a pointer to struct{Cases []T}).
func loadCases(t *testing.T, file string, out any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "protocol", "testcases", file))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
}

// cfgWith returns Default() with the case's config overrides applied.
func cfgWith(t *testing.T, raw json.RawMessage) config.PortalConfig {
	t.Helper()
	c := config.Default()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func testClientCfgOff() config.PortalConfig {
	c := config.Default()
	c.PortalHost = "fw.bits-pilani.invalid"
	return c
}
