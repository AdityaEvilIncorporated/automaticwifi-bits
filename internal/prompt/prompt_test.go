package prompt

import (
	"errors"
	"strings"
	"testing"

	"autowifi/internal/creds"
)

func fakeTerm(in, pw string, pwErr error) (Terminal, *strings.Builder) {
	out := &strings.Builder{}
	return Terminal{In: strings.NewReader(in), Out: out, ReadPassword: func() (string, error) { return pw, pwErr }}, out
}

func TestTerminalAsk(t *testing.T) {
	tm, out := fakeTerm("  f20250001 \n", " pass word ", nil)
	got, err := tm.Ask()
	if err != nil || got != (creds.Credentials{Username: "f20250001", Password: " pass word "}) {
		t.Fatalf("%q %v", got.Password, err)
	}
	if strings.Contains(out.String(), "pass word") {
		t.Fatal("password echoed")
	}
	for _, c := range []struct{ in, pw string }{{"", "x"}, {"\n", "x"}, {"id\n", ""}, {"id\n", "\r\n"}} {
		tm, _ := fakeTerm(c.in, c.pw, nil)
		if _, err := tm.Ask(); !errors.Is(err, ErrCancelled) {
			t.Errorf("in %q pw %q: err %v", c.in, c.pw, err)
		}
	}
	tm, _ = fakeTerm("id\n", "", errors.New("not a tty"))
	if _, err := tm.Ask(); err == nil || errors.Is(err, ErrCancelled) {
		t.Fatalf("read error: %v", err)
	}
	// no trailing newline on the ID line still works
	tm, _ = fakeTerm("f2025", "pw", nil)
	if got, err := tm.Ask(); err != nil || got.Username != "f2025" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParseWindows(t *testing.T) {
	cases := []struct {
		out  string
		want creds.Credentials
		ok   bool
	}{
		{"f20250001\nhunter2", creds.Credentials{Username: "f20250001", Password: "hunter2"}, true},
		{`\f20250001` + "\npw with space \n", creds.Credentials{Username: "f20250001", Password: "pw with space "}, true},
		{"", creds.Credentials{}, false},
		{"user\n", creds.Credentials{}, false},
		{"\npw", creds.Credentials{}, false},
	}
	for _, c := range cases {
		got, err := parseWindows(c.out)
		if c.ok != (err == nil) || got != c.want {
			t.Errorf("parseWindows(%q) = %v, %v", c.out, got, err)
		}
	}
}

func TestTrimOne(t *testing.T) {
	if trimOne("pw  \n") != "pw  " || trimOne("pw\r\n") != "pw" || trimOne("pw") != "pw" || trimOne("pw\n\n") != "pw\n" {
		t.Fatal("trimOne")
	}
}
