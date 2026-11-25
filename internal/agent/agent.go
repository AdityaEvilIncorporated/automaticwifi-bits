// Package agent runs the engine in the background: one goroutine owns the Engine and serves
// triggers (timer, network change, manual, credentials updated) one at a time.
// HARD task T14 (supervisor).
package agent

import (
	"context"
	"time"

	"autowifi/internal/engine"
)

// NetworkPollInterval is how often local interface addresses are compared to detect a
// network change (no OS-specific APIs; SSIDs are never read).
const NetworkPollInterval = 15 * time.Second

// Agent serialises all engine work.
type Agent struct {
	Engine *engine.Engine
	// Addrs returns a fingerprint of the current interface addresses (net.InterfaceAddrs,
	// sorted, loopback and link-local removed). Injected for tests.
	Addrs func() (string, error)
	// OnStep is called after every Run with the trigger and the Step (tray status text).
	OnStep func(engine.Trigger, engine.Step)
}

// Run blocks until ctx is done. It runs TriggerStart at once, then waits for the earliest of:
// the Step.Next timer (TriggerTimer), an address fingerprint change (TriggerNetwork, debounced
// 3 s so a Wi-Fi join that brings up several addresses yields one check), or a Trigger sent
// through Kick. Triggers arriving while a Run is in progress are coalesced into one.
func (a *Agent) Run(ctx context.Context) error {
	panic("not implemented")
}

// Kick asks for a Run with trigger t (TriggerManual, TriggerCredsUpdated); never blocks.
func (a *Agent) Kick(t engine.Trigger) {
	panic("not implemented")
}

// Logout asks the agent goroutine to run Engine.Logout; never blocks.
func (a *Agent) Logout() {
	panic("not implemented")
}
