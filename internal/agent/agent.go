// Package agent runs the engine in the background: one goroutine owns the Engine and serves
// triggers (timer, network change, manual, credentials updated) one at a time.
package agent

import (
	"context"
	"math/rand/v2"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"autowifi/internal/backoff"
	"autowifi/internal/engine"
	"autowifi/internal/portal"
)

// NetworkPollInterval is how often local interface addresses are compared to detect a
// network change (no OS-specific APIs; SSIDs are never read). Reading the interface list is a
// cheap syscall, and polling must be faster than Debounce for the debounce to merge anything.
const NetworkPollInterval = 2 * time.Second

// Debounce is how long the address set must stay unchanged before a network check runs, so a
// Wi-Fi join that brings up IPv4, IPv6 and temporary addresses one by one yields one check.
const Debounce = 3 * time.Second

// MaxJitter spreads steady 5-minute checks so many clients never hit the portal in lockstep.
const MaxJitter = 20 * time.Second

// Agent serialises all engine work. Engine is required; every other field has a default.
type Agent struct {
	Engine *engine.Engine
	// Addrs returns a fingerprint of the current interface addresses (net.InterfaceAddrs,
	// sorted, loopback and link-local removed). Injected for tests. Default: InterfaceFingerprint.
	Addrs func() (string, error)
	// OnStep is called after every Run with the trigger and the Step (tray status text).
	OnStep func(engine.Trigger, engine.Step)

	// After and PollInterval exist so tests can run the loop fast; zero = real time.
	After        func(time.Duration) <-chan time.Time
	PollInterval time.Duration
	// Jitter returns the extra delay added to a steady (5 min) wait. Default: uniform ±MaxJitter.
	Jitter func() time.Duration

	once   sync.Once
	kicks  chan engine.Trigger
	logout chan struct{}
}

func (a *Agent) init() {
	a.once.Do(func() {
		a.kicks = make(chan engine.Trigger, 8)
		a.logout = make(chan struct{}, 1)
		if a.Addrs == nil {
			a.Addrs = InterfaceFingerprint
		}
		if a.After == nil {
			a.After = time.After
		}
		if a.PollInterval <= 0 {
			a.PollInterval = NetworkPollInterval
		}
		if a.Jitter == nil {
			a.Jitter = func() time.Duration {
				return time.Duration(rand.Int64N(int64(2*MaxJitter))) - MaxJitter
			}
		}
	})
}

// Run blocks until ctx is done. It runs TriggerStart at once, then waits for the earliest of:
// the Step.Next timer (TriggerTimer), an address fingerprint change (TriggerNetwork, debounced
// 3 s so a Wi-Fi join that brings up several addresses yields one check), or a Trigger sent
// through Kick. Triggers arriving while a Run is in progress are coalesced into one.
func (a *Agent) Run(ctx context.Context) error {
	a.init()
	lastAddrs, _ := a.Addrs()
	timer := a.wait(a.step(ctx, engine.TriggerStart))
	poll := time.NewTicker(a.PollInterval)
	defer poll.Stop()
	var debounce <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer:
			timer = a.wait(a.step(ctx, engine.TriggerTimer))
		case <-poll.C:
			if fp, err := a.Addrs(); err == nil && fp != lastAddrs {
				lastAddrs = fp
				debounce = a.After(Debounce) // restart the debounce on every change
			}
		case <-debounce:
			debounce = nil
			timer = a.wait(a.step(ctx, engine.TriggerNetwork))
		case t := <-a.kicks:
			timer = a.wait(a.step(ctx, a.drain(t)))
		case <-a.logout:
			a.Engine.Logout(ctx)
			if a.OnStep != nil {
				a.OnStep(engine.TriggerManual, engine.Step{Outcome: OutcomeLoggedOut, Next: backoff.Steady})
			}
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

// OutcomeLoggedOut is reported to OnStep after a Logout.
const OutcomeLoggedOut portal.Outcome = "logged_out"

// drain coalesces every queued kick into one trigger. Credentials-updated wins (it also
// un-pauses), then manual, then whatever came first.
func (a *Agent) drain(first engine.Trigger) engine.Trigger {
	best := first
	for {
		select {
		case t := <-a.kicks:
			if rank(t) > rank(best) {
				best = t
			}
		default:
			return best
		}
	}
}

func rank(t engine.Trigger) int {
	switch t {
	case engine.TriggerCredsUpdated:
		return 2
	case engine.TriggerManual:
		return 1
	}
	return 0
}

func (a *Agent) step(ctx context.Context, t engine.Trigger) engine.Step {
	s := a.Engine.Run(ctx, t)
	if a.OnStep != nil {
		a.OnStep(t, s)
	}
	return s
}

func (a *Agent) wait(s engine.Step) <-chan time.Time {
	d := s.Next
	if d == backoff.Steady {
		d += a.Jitter()
	}
	if d <= 0 {
		d = time.Second
	}
	return a.After(d)
}

// Kick asks for a Run with trigger t (TriggerManual, TriggerCredsUpdated); never blocks.
func (a *Agent) Kick(t engine.Trigger) {
	a.init()
	select {
	case a.kicks <- t:
	default: // queue full: a run is pending anyway and drain keeps the highest-ranked trigger
	}
}

// Logout asks the agent goroutine to run Engine.Logout; never blocks.
func (a *Agent) Logout() {
	a.init()
	select {
	case a.logout <- struct{}{}:
	default:
	}
}

// InterfaceFingerprint returns the sorted list of non-loopback, non-link-local interface
// addresses, joined by commas. Any change (Wi-Fi join, cable, wake from sleep with a new
// lease) changes the fingerprint.
func InterfaceFingerprint() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	var out []string
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipn.IP
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		out = append(out, ipn.String())
	}
	sort.Strings(out)
	return strings.Join(out, ","), nil
}
