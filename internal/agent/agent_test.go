package agent

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"autowifi/internal/creds"
	"autowifi/internal/engine"
	"autowifi/internal/mockportal"
	"autowifi/internal/notify"
	"autowifi/internal/portal"
)

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// fast compresses time 1000x: the 5-minute keepalive runs every 300 ms.
func fast(d time.Duration) <-chan time.Time { return time.After(d / 1000) }

type rec struct {
	mu    sync.Mutex
	steps []engine.Step
	trigs []engine.Trigger
}

func (r *rec) add(t engine.Trigger, s engine.Step) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trigs = append(r.trigs, t)
	r.steps = append(r.steps, s)
}

func (r *rec) count(t engine.Trigger) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, x := range r.trigs {
		if x == t {
			n++
		}
	}
	return n
}

type countNotifier struct{ n atomic.Int32 }

func (c *countNotifier) Notify(string, string) error { c.n.Add(1); return nil }

func rig(t *testing.T, password string) (*mockportal.Server, *Agent, *rec, *countNotifier) {
	t.Helper()
	s := mockportal.Start(mockportal.Options{})
	t.Cleanup(s.Close)
	c := &portal.Client{
		Cfg:      s.Config(),
		Probe:    &http.Client{Timeout: 5 * time.Second, CheckRedirect: noRedirect},
		Portal:   &http.Client{Timeout: 5 * time.Second, CheckRedirect: noRedirect, Transport: &http.Transport{TLSClientConfig: s.TLSClientConfig()}},
		Resolver: net.DefaultResolver,
	}
	st := creds.NewMemory()
	st.Set(creds.Credentials{Username: mockportal.DefaultUsername, Password: password})
	n := &countNotifier{}
	r := &rec{}
	a := &Agent{
		Engine:       &engine.Engine{Portal: c, Creds: st, Notifier: n, Now: time.Now},
		Addrs:        func() (string, error) { return "fixed", nil },
		OnStep:       r.add,
		After:        fast,
		PollInterval: 5 * time.Millisecond,
		Jitter:       func() time.Duration { return 0 },
	}
	return s, a, r, n
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

func start(t *testing.T, a *Agent) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return cancel
}

// Flow 3 of PLAN.md: login, keepalive every 5 min, fresh login after the session expires.
func TestSilentLoginKeepaliveAndRelogin(t *testing.T) {
	s, a, _, n := rig(t, mockportal.DefaultPassword)
	start(t, a)
	eventually(t, "first login", func() bool { return s.Snapshot().LoginsOK == 1 })
	eventually(t, "two keepalives", func() bool { return s.Snapshot().Keepalives >= 2 })
	s.Expire()
	eventually(t, "relogin after expiry", func() bool { return s.Snapshot().LoginsOK == 2 })
	a.Logout()
	eventually(t, "logout", func() bool { return s.Snapshot().Logouts == 1 })
	if n.n.Load() != 0 {
		t.Fatal("notified on success")
	}
}

// Flow 5: wrong password is tried once, notifies once, then pauses until credentials change.
func TestRejectedPausesUntilCredsUpdated(t *testing.T) {
	s, a, r, n := rig(t, "wrong")
	start(t, a)
	eventually(t, "rejection", func() bool { return s.Snapshot().LoginsFailed == 1 })
	eventually(t, "a few paused timer runs", func() bool { return r.count(engine.TriggerTimer) >= 3 })
	if got := s.Snapshot().LoginsFailed; got != 1 {
		t.Fatalf("retried a rejected password: %d attempts", got)
	}
	if n.n.Load() != 1 {
		t.Fatalf("notifications = %d, want 1", n.n.Load())
	}
	a.Engine.Creds.Set(creds.Credentials{Username: mockportal.DefaultUsername, Password: mockportal.DefaultPassword})
	a.Kick(engine.TriggerCredsUpdated)
	eventually(t, "login with new password", func() bool { return s.Snapshot().LoginsOK == 1 })
}

// Flow 5: portal errors retry with backoff (5 s, 15 s, 60 s ... compressed) and notify on the 3rd.
func TestPortalErrorsBackOff(t *testing.T) {
	s, a, _, n := rig(t, mockportal.DefaultPassword)
	s.FailNext(3)
	start(t, a)
	eventually(t, "login after 3 failures", func() bool { return s.Snapshot().LoginsOK == 1 })
	if n.n.Load() != 1 {
		t.Fatalf("notifications = %d, want 1 (3rd consecutive error)", n.n.Load())
	}
}

func TestNetworkChangeTriggersOneCheck(t *testing.T) {
	s, a, r, _ := rig(t, mockportal.DefaultPassword)
	s.SetAuthenticated(true) // already online: checks are cheap probes
	var fp atomic.Value
	fp.Store("wifi-a")
	a.Addrs = func() (string, error) { return fp.Load().(string), nil }
	a.After = func(d time.Duration) <-chan time.Time {
		if d >= time.Minute { // keep the periodic timer out of the way
			return nil
		}
		return time.After(d / 100) // debounce 30 ms
	}
	a.PollInterval = 2 * time.Millisecond
	start(t, a)
	eventually(t, "start check", func() bool { return r.count(engine.TriggerStart) == 1 })
	// several quick changes (addresses coming up one by one, 5 ms apart) → a single network check
	for _, v := range []string{"wifi-b", "wifi-b,v6", "wifi-b,v6,tmp"} {
		fp.Store(v)
		time.Sleep(5 * time.Millisecond)
	}
	eventually(t, "network check", func() bool { return r.count(engine.TriggerNetwork) >= 1 })
	time.Sleep(100 * time.Millisecond)
	if got := r.count(engine.TriggerNetwork); got != 1 {
		t.Fatalf("network checks = %d, want 1 (debounced)", got)
	}
}

func TestKickCoalescesAndNeverBlocks(t *testing.T) {
	a := &Agent{}
	for i := 0; i < 100; i++ {
		a.Kick(engine.TriggerManual)
	}
	a.Kick(engine.TriggerCredsUpdated)
	a.Logout()
	a.Logout()
	if got := a.drain(<-a.kicks); got != engine.TriggerManual && got != engine.TriggerCredsUpdated {
		t.Fatalf("drain = %s", got)
	}
	a2 := &Agent{}
	a2.Kick(engine.TriggerManual)
	a2.Kick(engine.TriggerCredsUpdated)
	a2.Kick(engine.TriggerManual)
	if got := a2.drain(<-a2.kicks); got != engine.TriggerCredsUpdated {
		t.Fatalf("drain = %s, want creds_updated", got)
	}
}

func TestInterfaceFingerprint(t *testing.T) {
	if _, err := InterfaceFingerprint(); err != nil {
		t.Fatal(err)
	}
}

var _ notify.Notifier = (*countNotifier)(nil)
