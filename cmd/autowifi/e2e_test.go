package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"autowifi/internal/config"
	"autowifi/internal/creds"
	"autowifi/internal/mockportal"
	"autowifi/internal/portal"
	"autowifi/internal/prompt"
	"autowifi/internal/state"
	"autowifi/internal/tlsconf"
)

// syncBuf is a goroutine-safe output buffer.
type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type countNotifier struct{ n atomic.Int32 }

func (c *countNotifier) Notify(string, string) error { c.n.Add(1); return nil }

type fixture struct {
	mock     *mockportal.Server
	env      *env
	out, err *syncBuf
	notes    *countNotifier
	autoOn   atomic.Int32
	autoOff  atomic.Int32
	asked    atomic.Int32
}

// newFixture writes a config pointing at the mock portal. The portal certificate is NOT in
// any OS trust store, so every portal request goes through the real tlsconf pin fallback.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	m := mockportal.Start(mockportal.Options{})
	t.Cleanup(m.Close)
	dir := t.TempDir()
	cfg := m.Config()
	cfg.PinnedCertSHA256 = tlsconf.SPKISHA256(m.Portal.Certificate())
	if err := config.Save(filepath.Join(dir, "config.json"), cfg); err != nil {
		t.Fatal(err)
	}
	f := &fixture{mock: m, out: &syncBuf{}, err: &syncBuf{}, notes: &countNotifier{}}
	f.env = &env{
		ctx: context.Background(), stdout: f.out, stderr: f.err, configDir: dir, home: t.TempDir(),
		exe: "/opt/autowifi", store: creds.NewMemory(), notifier: f.notes, interactive: true,
		askTerminal: func() (creds.Credentials, error) {
			f.asked.Add(1)
			return creds.Credentials{Username: mockportal.DefaultUsername, Password: mockportal.DefaultPassword}, nil
		},
		askDialog: func() (creds.Credentials, error) {
			t.Error("dialog used in an interactive test")
			return creds.Credentials{}, nil
		},
		newClient: portal.NewClient,
		autostartOn: func(home, exe string) error {
			if exe != "/opt/autowifi" {
				t.Errorf("autostart exe %q", exe)
			}
			f.autoOn.Add(1)
			return nil
		},
		autostartOff:  func(string) error { f.autoOff.Add(1); return nil },
		autostartIsOn: func(string) (bool, error) { return f.autoOn.Load() > f.autoOff.Load(), nil },
		after:         func(d time.Duration) <-chan time.Time { return time.After(d / 1000) },
		pollInterval:  5 * time.Millisecond,
		kickInterval:  5 * time.Millisecond,
	}
	return f
}

func (f *fixture) run(args ...string) int {
	f.out.mu.Lock()
	f.out.b.Reset()
	f.out.mu.Unlock()
	return realMain(args, f.env)
}

func (f *fixture) storeGood() {
	f.env.store.Set(creds.Credentials{Username: mockportal.DefaultUsername, Password: mockportal.DefaultPassword})
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestVersionAndUsage(t *testing.T) {
	f := newFixture(t)
	if f.run("version") != 0 || strings.TrimSpace(f.out.String()) != version {
		t.Fatalf("version: %q", f.out.String())
	}
	if f.run("nope") != 2 || !strings.Contains(f.err.String(), "unknown command") {
		t.Fatal("unknown command")
	}
	if f.run("autostart", "maybe") != 2 {
		t.Fatal("autostart arg")
	}
}

// Flow 1: setup stores the credentials and leaves a kick for a running agent.
func TestSetup(t *testing.T) {
	f := newFixture(t)
	if code := f.run("setup"); code != 0 {
		t.Fatalf("setup exit %d: %s", code, f.err.String())
	}
	if c, err := f.env.store.Get(); err != nil || c.Username != mockportal.DefaultUsername {
		t.Fatalf("stored %v %v", c, err)
	}
	if _, err := os.Stat(filepath.Join(f.env.configDir, "kick-creds")); err != nil {
		t.Fatal("no kick-creds marker")
	}
	if strings.Contains(f.out.String()+f.err.String(), mockportal.DefaultPassword) {
		t.Fatal("password printed")
	}
}

func TestAutostartAndForget(t *testing.T) {
	f := newFixture(t)
	if f.run("autostart", "on") != 0 || f.autoOn.Load() != 1 {
		t.Fatal("autostart on")
	}
	if f.run("autostart", "off") != 0 || f.autoOff.Load() != 1 {
		t.Fatal("autostart off")
	}
	f.storeGood()
	if f.run("forget") != 0 {
		t.Fatal("forget")
	}
	if _, err := f.env.store.Get(); err == nil {
		t.Fatal("credentials still stored")
	}
}

// login-now and logout in the foreground (no agent running), through the pinned TLS path.
func TestLoginNowAndLogout(t *testing.T) {
	f := newFixture(t)
	if code := f.run("login-now"); code != 1 || !strings.Contains(f.out.String(), "no_credentials") {
		t.Fatalf("without creds: exit %d out %q", code, f.out.String())
	}
	f.storeGood()
	if code := f.run("login-now"); code != 0 || !strings.Contains(f.out.String(), "logged_in") {
		t.Fatalf("login-now exit %d out %q err %q", code, f.out.String(), f.err.String())
	}
	st, _ := state.Load(filepath.Join(f.env.configDir, "state.json"))
	if !st.LoggedIn() || f.mock.Snapshot().LoginsOK != 1 {
		t.Fatalf("state %+v", st)
	}
	if code := f.run("status"); code != 0 || !strings.Contains(f.out.String(), "Portal session: yes") || !strings.Contains(f.out.String(), "Agent running: no") {
		t.Fatalf("status: %s", f.out.String())
	}
	if code := f.run("logout"); code != 0 || f.mock.Snapshot().Logouts != 1 {
		t.Fatalf("logout exit %d, logouts %d", code, f.mock.Snapshot().Logouts)
	}
}

func TestWrongPinIsRefused(t *testing.T) {
	f := newFixture(t)
	p := filepath.Join(f.env.configDir, "config.json")
	cfg, _ := config.Load(p)
	cfg.PinnedCertSHA256 = strings.Repeat("ab", 32)
	config.Save(p, cfg)
	f.storeGood()
	if code := f.run("login-now"); code != 1 || !strings.Contains(f.out.String(), "portal_error") {
		t.Fatalf("exit %d out %q", code, f.out.String())
	}
	if st := f.mock.Snapshot(); st.LoginsOK+st.LoginsFailed != 0 {
		t.Fatal("credentials sent over an unverified connection")
	}
}

func TestCapture(t *testing.T) {
	f := newFixture(t)
	if code := f.run("capture"); code != 0 {
		t.Fatalf("capture exit %d: %s", code, f.err.String())
	}
	out := f.out.String()
	for _, want := range []string{"On campus: yes", "Probe: captive (HTTP 303)", "form 0: username (text)", "TLS verified by OS trust store: no"} {
		if !strings.Contains(out, want) {
			t.Errorf("capture output missing %q:\n%s", want, out)
		}
	}
	if st := f.mock.Snapshot(); st.LoginsOK+st.LoginsFailed != 0 {
		t.Fatal("capture logged in")
	}
}

// Flows 1+3: first run asks once, enables start at login, logs in, keeps alive, re-logs after
// expiry; other commands talk to the running agent through kick files; one agent only.
func TestRunAgentEndToEnd(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.env.ctx = ctx
	done := make(chan int, 1)
	go func() { done <- realMain([]string{"run", "--no-tray"}, f.env) }()

	eventually(t, "first login", func() bool { return f.mock.Snapshot().LoginsOK == 1 })
	if f.asked.Load() != 1 || f.autoOn.Load() != 1 {
		t.Fatalf("first run: asked %d, autostart %d", f.asked.Load(), f.autoOn.Load())
	}
	eventually(t, "keepalives", func() bool { return f.mock.Snapshot().Keepalives >= 2 })
	f.mock.Expire()
	eventually(t, "relogin", func() bool { return f.mock.Snapshot().LoginsOK == 2 })

	// a second agent refuses to start
	second := *f.env
	second.stdout = &syncBuf{}
	if code := realMain([]string{"run", "--no-tray"}, &second); code != 0 || !strings.Contains(second.stdout.(*syncBuf).String(), "already running") {
		t.Fatalf("second agent: exit %d", code)
	}
	// logout from another process goes through the agent
	if code := f.run("logout"); code != 0 || !strings.Contains(f.out.String(), "running") {
		t.Fatalf("logout via agent: %d %q", code, f.out.String())
	}
	eventually(t, "logout via kick", func() bool { return f.mock.Snapshot().Logouts == 1 })
	if code := f.run("login-now"); code != 0 {
		t.Fatal("login-now via agent")
	}
	eventually(t, "login via kick", func() bool { return f.mock.Snapshot().LoginsOK == 3 })
	if code := f.run("status"); code != 0 || !strings.Contains(f.out.String(), "Agent running: yes") {
		t.Fatalf("status while running: %s", f.out.String())
	}

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("run exit %d: %s", code, f.err.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not stop on cancel")
	}

	logb, err := os.ReadFile(filepath.Join(f.env.configDir, "autowifi.log"))
	if err != nil || !strings.Contains(string(logb), "logged_in") {
		t.Fatalf("log: %v %s", err, logb)
	}
	for _, secret := range []string{mockportal.DefaultPassword, mockportal.DefaultUsername, f.mock.Token(), "fgtauth?", "keepalive?"} {
		if secret != "" && strings.Contains(string(logb), secret) {
			t.Fatalf("log contains %q:\n%s", secret, logb)
		}
	}
	if f.notes.n.Load() != 0 {
		t.Fatal("notified on success")
	}
	if fi, _ := os.Stat(filepath.Join(f.env.configDir, "state.json")); fi.Mode().Perm() != 0o600 {
		t.Fatal("state.json mode")
	}
}

func TestRotWriter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.log")
	w, err := newRotWriter(p, 10)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("0123456789"))
	w.Write([]byte("abc"))
	w.Close()
	old, _ := os.ReadFile(p + ".1")
	cur, _ := os.ReadFile(p)
	if string(old) != "0123456789" || string(cur) != "abc" {
		t.Fatalf("old %q cur %q", old, cur)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatal("log mode")
	}
}

func TestHelpAndConfigFlag(t *testing.T) {
	f := newFixture(t)
	for _, h := range []string{"-h", "--help", "help"} {
		if code := f.run(h); code != 0 || !strings.Contains(f.out.String(), "usage: autowifi") {
			t.Fatalf("%s: exit %d", h, code)
		}
	}
	// --config elsewhere: state and markers live next to it
	other := t.TempDir()
	src, _ := os.ReadFile(filepath.Join(f.env.configDir, "config.json"))
	os.WriteFile(filepath.Join(other, "config.json"), src, 0o600)
	f.storeGood()
	if code := f.run("--config", filepath.Join(other, "config.json"), "login-now"); code != 0 {
		t.Fatalf("--config login-now exit %d: %s", code, f.err.String())
	}
	if _, err := os.Stat(filepath.Join(other, "state.json")); err != nil {
		t.Fatal("state not written next to --config")
	}
	if _, err := os.Stat(filepath.Join(f.env.configDir, "state.json")); err == nil {
		t.Fatal("state written to the default dir despite --config")
	}
}

func TestSetupCancelledAndDialogPath(t *testing.T) {
	f := newFixture(t)
	f.env.askTerminal = func() (creds.Credentials, error) { return creds.Credentials{}, prompt.ErrCancelled }
	if code := f.run("setup"); code != 1 || !strings.Contains(f.err.String(), "Cancelled") {
		t.Fatalf("cancelled: exit %d", code)
	}
	if _, err := f.env.store.Get(); err == nil {
		t.Fatal("stored after cancel")
	}
	f.env.interactive = false
	used := false
	f.env.askDialog = func() (creds.Credentials, error) {
		used = true
		return creds.Credentials{Username: "u", Password: "p"}, nil
	}
	if code := f.run("setup"); code != 0 || !used {
		t.Fatalf("dialog path: exit %d used %v", code, used)
	}
}

// A running agent paused on rejected credentials resumes when `autowifi setup` saves new ones.
func TestSetupUnpausesRunningAgent(t *testing.T) {
	f := newFixture(t)
	f.env.store.Set(creds.Credentials{Username: mockportal.DefaultUsername, Password: "wrong"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.env.ctx = ctx
	done := make(chan int, 1)
	go func() { done <- realMain([]string{"run", "--no-tray"}, f.env) }()
	eventually(t, "rejection", func() bool { return f.mock.Snapshot().LoginsFailed == 1 })
	time.Sleep(50 * time.Millisecond)
	if f.mock.Snapshot().LoginsFailed != 1 {
		t.Fatal("retried a rejected password")
	}
	if code := f.run("setup"); code != 0 { // askTerminal returns the right password
		t.Fatalf("setup exit %d", code)
	}
	eventually(t, "login with new password", func() bool { return f.mock.Snapshot().LoginsOK == 1 })
	if f.notes.n.Load() != 1 {
		t.Fatalf("notifications %d, want 1 (the rejection)", f.notes.n.Load())
	}
	cancel()
	<-done
}
