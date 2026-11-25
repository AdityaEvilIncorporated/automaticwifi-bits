package main

import "testing"

func TestCommandOf(t *testing.T) {
	cases := map[string][]string{
		"run":     {},
		"setup":   {"--config", "x.json", "setup"},
		"capture": {"-config=x.json", "capture"},
		"status":  {"status", "--config", "x"},
	}
	for want, args := range cases {
		if got := commandOf(args); got != want {
			t.Errorf("commandOf(%q) = %q, want %q", args, got, want)
		}
	}
	if got := commandOf([]string{"--config", "x.json"}); got != "run" {
		t.Errorf("flags only = %q, want run", got)
	}
	if got := commandOf([]string{"--bogus"}); got != "" {
		t.Errorf("bad flag = %q, want \"\" (realMain prints the error)", got)
	}
}
