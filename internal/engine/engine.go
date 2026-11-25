// Package engine is the decision logic of PROTOCOL.md sections 7-8: what to do on each
// trigger, retry delays, pausing on rejected credentials and the hourly notification limit.
// It is single-threaded: the caller (internal/agent) serialises calls to Run and Logout.
package engine

import (
	"context"
	"time"

	"autowifi/internal/creds"
	"autowifi/internal/notify"
	"autowifi/internal/portal"
	"autowifi/internal/state"
)

// Portal is the network side; *portal.Client satisfies it.
type Portal interface {
	CheckOnce(ctx context.Context, get func() (creds.Credentials, error)) portal.Result
	Keepalive(ctx context.Context, keepaliveURL string) bool
	Logout(ctx context.Context, logoutURL string) error
}

// Trigger is why Run is called.
type Trigger string

const (
	TriggerStart        Trigger = "start"
	TriggerTimer        Trigger = "timer"
	TriggerNetwork      Trigger = "network"
	TriggerManual       Trigger = "manual"        // "Log in now"
	TriggerCredsUpdated Trigger = "creds_updated" // user saved new credentials
)

// Extra outcomes reported by Run besides the portal outcomes.
const (
	OutcomeKeepaliveOK portal.Outcome = "keepalive_ok"
	OutcomePaused      portal.Outcome = "paused" // credentials rejected, waiting for new ones
)

// Step is what one Run did.
type Step struct {
	Outcome  portal.Outcome
	Next     time.Duration // wait before the next TriggerTimer
	Notified notify.Reason // "" when no notification was sent
}

// Engine holds the session. All fields except Save and Log are required.
type Engine struct {
	Portal   Portal
	Creds    creds.Store
	Notifier notify.Notifier
	State    state.SessionState
	Now      func() time.Time
	Save     func(state.SessionState) error   // called at the end of every Run/Logout; nil = skip
	Log      func(format string, args ...any) // outcomes only, never secrets; nil = skip
}

// Run handles one trigger. Rules, in order:
//
//  1. TriggerCredsUpdated: clear CredentialsRejected and ConsecutiveFailures, then go on to 4.
//  2. CredentialsRejected and trigger is not TriggerManual → Step{OutcomePaused, Steady}, no
//     network call, no notification.
//  3. State.LoggedIn() and trigger is TriggerTimer: Portal.Keepalive(KeepaliveURL).
//     Alive → ConsecutiveFailures = 0, Step{OutcomeKeepaliveOK, backoff.Steady}.
//     Not alive → State.ClearSession() and go on to 4 (a fresh check right away).
//  4. r := Portal.CheckOnce(ctx, Creds.Get), then by r.Outcome:
//     - OffCampus: ClearSession, failures = 0, Next Steady.
//     - Online, Foreign: failures = 0, Next Steady (session kept).
//     - LoggedIn: MagicToken/KeepaliveURL/LogoutURL from r, failures = 0,
//     CredentialsRejected = false, Next Steady.
//     - Rejected: CredentialsRejected = true, ClearSession, failures = 0,
//     notify ReasonRejected, Outcome Rejected, Next Steady.
//     - NoCredentials: failures = 0, notify ReasonNoCredentials, Next Steady.
//     - FormChanged: failures++, notify ReasonFormChanged, Next backoff.Delay(failures).
//     - PortalError: failures++, notify ReasonRepeatedErrors when failures >= 3,
//     Next backoff.Delay(failures).
//  5. Save(State) if Save != nil (an error is passed to Log, never returned).
//
// "notify r" means: if notify.Allow(State.LastNotificationAt, Now()) then
// Notifier.Notify(notify.Message(r)), State.LastNotificationAt = Now() (even if Notify
// returned an error, so a broken notifier cannot cause a retry storm), Step.Notified = r.
// Otherwise nothing. Notifications never happen on LoggedIn, Online, OffCampus, Foreign or
// keepalive success.
func (e *Engine) Run(ctx context.Context, t Trigger) Step {
	panic("not implemented")
}

// Logout ends the session: when State.LoggedIn(), Portal.Logout(LogoutURL) (error only
// logged), then ClearSession and Save. Not logged in: nothing but Save.
func (e *Engine) Logout(ctx context.Context) {
	panic("not implemented")
}
