package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"autowifi/internal/backoff"
	"autowifi/internal/creds"
	"autowifi/internal/notify"
	"autowifi/internal/portal"
	"autowifi/internal/state"
)

type fakePortal struct {
	results    []portal.Result // CheckOnce returns these in order; last one repeats
	checks     int
	alive      bool
	keepalives []string
	logouts    []string
	credsSeen  []creds.Credentials
}

func (f *fakePortal) CheckOnce(ctx context.Context, get func() (creds.Credentials, error)) portal.Result {
	f.checks++
	if c, err := get(); err == nil {
		f.credsSeen = append(f.credsSeen, c)
	}
	i := f.checks - 1
	if i >= len(f.results) {
		i = len(f.results) - 1
	}
	return f.results[i]
}
func (f *fakePortal) Keepalive(ctx context.Context, u string) bool {
	f.keepalives = append(f.keepalives, u)
	return f.alive
}
func (f *fakePortal) Logout(ctx context.Context, u string) error {
	f.logouts = append(f.logouts, u)
	return errors.New("ignored")
}

type fakeNotifier struct {
	msgs []string
	err  error
}

func (n *fakeNotifier) Notify(title, body string) error {
	n.msgs = append(n.msgs, body)
	return n.err
}

type rig struct {
	e     *Engine
	p     *fakePortal
	n     *fakeNotifier
	now   time.Time
	saves int
}

func newRig(results ...portal.Result) *rig {
	r := &rig{p: &fakePortal{results: results}, n: &fakeNotifier{}, now: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}
	st := creds.NewMemory()
	st.Set(creds.Credentials{Username: "f20250001", Password: "pw"})
	r.e = &Engine{
		Portal: r.p, Creds: st, Notifier: r.n,
		Now:  func() time.Time { return r.now },
		Save: func(state.SessionState) error { r.saves++; return nil },
	}
	return r
}

func (r *rig) run(t Trigger) Step { return r.e.Run(context.Background(), t) }

const ka = "https://fw.bits-pilani.ac.in:8090/keepalive?aa"
const lo = "https://fw.bits-pilani.ac.in:8090/logout?aa"

var loggedIn = portal.Result{Outcome: portal.LoggedIn, Magic: "m1", KeepaliveURL: ka, LogoutURL: lo}

func TestLoginThenKeepalive(t *testing.T) {
	r := newRig(loggedIn)
	s := r.run(TriggerStart)
	if s != (Step{Outcome: portal.LoggedIn, Next: backoff.Steady}) {
		t.Fatalf("step %+v", s)
	}
	st := r.e.State
	if st.KeepaliveURL != ka || st.LogoutURL != lo || st.MagicToken != "m1" || st.ConsecutiveFailures != 0 {
		t.Fatalf("state %+v", st)
	}
	if len(r.n.msgs) != 0 {
		t.Fatal("notified on success")
	}
	r.p.alive = true
	s = r.run(TriggerTimer)
	if s != (Step{Outcome: OutcomeKeepaliveOK, Next: backoff.Steady}) || r.p.checks != 1 || len(r.p.keepalives) != 1 || r.p.keepalives[0] != ka {
		t.Fatalf("keepalive step %+v checks %d", s, r.p.checks)
	}
	if r.saves != 2 {
		t.Fatalf("saves %d", r.saves)
	}
}

func TestKeepaliveFailureTriggersFreshLogin(t *testing.T) {
	r := newRig(loggedIn)
	r.run(TriggerStart)
	r.p.alive = false
	r.p.results = []portal.Result{loggedIn, {Outcome: portal.LoggedIn, Magic: "m2", KeepaliveURL: ka + "2", LogoutURL: lo + "2"}}
	s := r.run(TriggerTimer)
	if s.Outcome != portal.LoggedIn || r.p.checks != 2 || r.e.State.KeepaliveURL != ka+"2" {
		t.Fatalf("step %+v checks %d state %+v", s, r.p.checks, r.e.State)
	}
}

func TestKeepaliveFailureThenOnlineClearsSession(t *testing.T) {
	r := newRig(loggedIn, portal.Result{Outcome: portal.Online})
	r.run(TriggerStart)
	s := r.run(TriggerTimer) // alive=false
	if s.Outcome != portal.Online || r.e.State.LoggedIn() {
		t.Fatalf("step %+v state %+v", s, r.e.State)
	}
}

func TestNetworkTriggerChecksEvenWhenLoggedIn(t *testing.T) {
	r := newRig(loggedIn, portal.Result{Outcome: portal.Online})
	r.run(TriggerStart)
	r.p.alive = true
	s := r.run(TriggerNetwork)
	if r.p.checks != 2 || len(r.p.keepalives) != 0 || s.Outcome != portal.Online {
		t.Fatalf("network trigger: checks %d keepalives %d step %+v", r.p.checks, len(r.p.keepalives), s)
	}
	if !r.e.State.LoggedIn() {
		t.Fatal("online must keep the session")
	}
}

func TestOffCampusClearsSessionQuietly(t *testing.T) {
	r := newRig(loggedIn, portal.Result{Outcome: portal.OffCampus})
	r.run(TriggerStart)
	r.e.State.ConsecutiveFailures = 2
	s := r.run(TriggerNetwork)
	if s != (Step{Outcome: portal.OffCampus, Next: backoff.Steady}) || r.e.State.LoggedIn() || r.e.State.ConsecutiveFailures != 0 || len(r.n.msgs) != 0 {
		t.Fatalf("step %+v state %+v msgs %v", s, r.e.State, r.n.msgs)
	}
}

func TestForeignResetsFailures(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.Foreign})
	r.e.State.ConsecutiveFailures = 3
	s := r.run(TriggerTimer)
	if s.Next != backoff.Steady || r.e.State.ConsecutiveFailures != 0 || len(r.n.msgs) != 0 {
		t.Fatalf("%+v %+v", s, r.e.State)
	}
}

func TestPortalErrorBackoffAndNotifyOnThird(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.PortalError, Err: errors.New("timeout")})
	want := []time.Duration{5 * time.Second, 15 * time.Second, 60 * time.Second, 5 * time.Minute, 5 * time.Minute}
	for i, d := range want {
		s := r.run(TriggerTimer)
		if s.Outcome != portal.PortalError || s.Next != d {
			t.Fatalf("attempt %d: %+v, want next %v", i+1, s, d)
		}
		if r.e.State.ConsecutiveFailures != i+1 {
			t.Fatalf("failures %d", r.e.State.ConsecutiveFailures)
		}
		wantNotified := notify.Reason("")
		if i == 2 {
			wantNotified = notify.ReasonRepeatedErrors
		}
		if s.Notified != wantNotified {
			t.Fatalf("attempt %d notified %q", i+1, s.Notified)
		}
		r.now = r.now.Add(d)
	}
	_, body := notify.Message(notify.ReasonRepeatedErrors)
	if len(r.n.msgs) != 1 || r.n.msgs[0] != body {
		t.Fatalf("msgs %v", r.n.msgs)
	}
	// an hour later the still-failing check notifies again
	r.now = r.now.Add(time.Hour)
	if s := r.run(TriggerTimer); s.Notified != notify.ReasonRepeatedErrors {
		t.Fatalf("after 1h: %+v", s)
	}
	// success resets
	r.p.results = []portal.Result{loggedIn}
	r.p.checks = 0
	r.run(TriggerTimer)
	if r.e.State.ConsecutiveFailures != 0 {
		t.Fatal("success did not reset failures")
	}
}

func TestRejectedPausesUntilCredsUpdated(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.Rejected})
	s := r.run(TriggerStart)
	if s != (Step{Outcome: portal.Rejected, Next: backoff.Steady, Notified: notify.ReasonRejected}) || !r.e.State.CredentialsRejected {
		t.Fatalf("%+v %+v", s, r.e.State)
	}
	for _, tr := range []Trigger{TriggerTimer, TriggerNetwork, TriggerStart} {
		s = r.run(tr)
		if s != (Step{Outcome: OutcomePaused, Next: backoff.Steady}) {
			t.Fatalf("%s while paused: %+v", tr, s)
		}
	}
	if r.p.checks != 1 || len(r.n.msgs) != 1 {
		t.Fatalf("checks %d msgs %d", r.p.checks, len(r.n.msgs))
	}
	// manual tries once, still rejected, stays paused, no second notification within the hour
	s = r.run(TriggerManual)
	if s.Outcome != portal.Rejected || s.Notified != "" || r.p.checks != 2 || !r.e.State.CredentialsRejected {
		t.Fatalf("manual: %+v", s)
	}
	// new credentials: unpaused, logs in
	r.e.Creds.Set(creds.Credentials{Username: "f20250001", Password: "new"})
	r.p.results = []portal.Result{loggedIn}
	r.p.checks = 0
	s = r.run(TriggerCredsUpdated)
	if s.Outcome != portal.LoggedIn || r.e.State.CredentialsRejected {
		t.Fatalf("creds updated: %+v %+v", s, r.e.State)
	}
	if last := r.p.credsSeen[len(r.p.credsSeen)-1]; last.Password != "new" {
		t.Fatal("did not use the new credentials")
	}
}

func TestRejectedClearsSession(t *testing.T) {
	r := newRig(loggedIn, portal.Result{Outcome: portal.Rejected})
	r.run(TriggerStart)
	r.run(TriggerNetwork)
	if r.e.State.LoggedIn() {
		t.Fatal("session kept after rejection")
	}
}

func TestFormChangedNotifiesAndBacksOff(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.FormChanged})
	s := r.run(TriggerStart)
	if s != (Step{Outcome: portal.FormChanged, Next: 5 * time.Second, Notified: notify.ReasonFormChanged}) {
		t.Fatalf("%+v", s)
	}
	r.now = r.now.Add(5 * time.Second)
	s = r.run(TriggerTimer)
	if s.Next != 15*time.Second || s.Notified != "" {
		t.Fatalf("second: %+v (hourly limit)", s)
	}
}

func TestNoCredentials(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.NoCredentials})
	r.e.State.ConsecutiveFailures = 2
	s := r.run(TriggerStart)
	if s != (Step{Outcome: portal.NoCredentials, Next: backoff.Steady, Notified: notify.ReasonNoCredentials}) || r.e.State.ConsecutiveFailures != 0 {
		t.Fatalf("%+v %+v", s, r.e.State)
	}
}

func TestHourlyLimitAcrossReasonsAndPersisted(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.FormChanged})
	r.e.State.LastNotificationAt = r.now.Add(-30 * time.Minute) // from a previous run
	if s := r.run(TriggerStart); s.Notified != "" || len(r.n.msgs) != 0 {
		t.Fatalf("notified within the hour: %+v", s)
	}
	r.now = r.now.Add(31 * time.Minute)
	r.p.results = []portal.Result{{Outcome: portal.Rejected}}
	r.p.checks = 0
	if s := r.run(TriggerTimer); s.Notified != notify.ReasonRejected {
		t.Fatalf("after the hour: %+v", s)
	}
	if !r.e.State.LastNotificationAt.Equal(r.now) {
		t.Fatal("LastNotificationAt not updated")
	}
}

func TestBrokenNotifierStillCountsAsSent(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.FormChanged})
	r.n.err = errors.New("no notification daemon")
	s := r.run(TriggerStart)
	if s.Notified != notify.ReasonFormChanged || !r.e.State.LastNotificationAt.Equal(r.now) {
		t.Fatalf("%+v %+v", s, r.e.State)
	}
}

func TestSaveErrorIsLoggedNotFatal(t *testing.T) {
	r := newRig(portal.Result{Outcome: portal.Online})
	var logged []string
	r.e.Save = func(state.SessionState) error { return errors.New("disk full") }
	r.e.Log = func(f string, a ...any) { logged = append(logged, f) }
	if s := r.run(TriggerStart); s.Outcome != portal.Online {
		t.Fatal(s)
	}
	if len(logged) == 0 {
		t.Fatal("save error not logged")
	}
	r.e.Save, r.e.Log = nil, nil
	r.run(TriggerStart) // nil Save/Log must not panic
}

func TestLogout(t *testing.T) {
	r := newRig(loggedIn)
	r.run(TriggerStart)
	saves := r.saves
	r.e.Logout(context.Background())
	if len(r.p.logouts) != 1 || r.p.logouts[0] != lo || r.e.State.LoggedIn() || r.saves != saves+1 {
		t.Fatalf("logouts %v state %+v", r.p.logouts, r.e.State)
	}
	r.e.Logout(context.Background())
	if len(r.p.logouts) != 1 {
		t.Fatal("logout called when not logged in")
	}
}
