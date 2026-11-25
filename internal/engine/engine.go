// Package engine is the decision logic of PROTOCOL.md sections 7-8: what to do on each
// trigger, retry delays, pausing on rejected credentials and the hourly notification limit.
// It is single-threaded: the caller (internal/agent) serialises calls to Run and Logout.
package engine

import (
	"context"
	"time"

	"autowifi/internal/backoff"
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
	var step Step

	switch {
	case t == TriggerCredsUpdated:
		e.State.CredentialsRejected = false
		e.State.ConsecutiveFailures = 0
	case e.State.CredentialsRejected && t != TriggerManual:
		step = Step{Outcome: OutcomePaused, Next: backoff.Steady}
		e.logf("check %s: %s", t, step.Outcome)
		e.save()
		return step
	case e.State.LoggedIn() && t == TriggerTimer:
		if e.Portal.Keepalive(ctx, e.State.KeepaliveURL) {
			e.State.ConsecutiveFailures = 0
			step = Step{Outcome: OutcomeKeepaliveOK, Next: backoff.Steady}
			e.logf("check %s: %s", t, step.Outcome)
			e.save()
			return step
		}
		e.State.ClearSession()
	}

	r := e.Portal.CheckOnce(ctx, e.Creds.Get)
	step.Outcome = r.Outcome

	switch r.Outcome {
	case portal.OffCampus:
		e.State.ClearSession()
		e.State.ConsecutiveFailures = 0
		step.Next = backoff.Steady
	case portal.Online, portal.Foreign:
		e.State.ConsecutiveFailures = 0
		step.Next = backoff.Steady
	case portal.LoggedIn:
		e.State.MagicToken = r.Magic
		e.State.KeepaliveURL = r.KeepaliveURL
		e.State.LogoutURL = r.LogoutURL
		e.State.ConsecutiveFailures = 0
		e.State.CredentialsRejected = false
		step.Next = backoff.Steady
	case portal.Rejected:
		e.State.CredentialsRejected = true
		e.State.ClearSession()
		e.State.ConsecutiveFailures = 0
		step.Notified = e.notify(notify.ReasonRejected)
		step.Next = backoff.Steady
	case portal.NoCredentials:
		e.State.ConsecutiveFailures = 0
		step.Notified = e.notify(notify.ReasonNoCredentials)
		step.Next = backoff.Steady
	case portal.FormChanged:
		e.State.ConsecutiveFailures++
		step.Notified = e.notify(notify.ReasonFormChanged)
		step.Next = backoff.Delay(e.State.ConsecutiveFailures)
	case portal.PortalError:
		e.State.ConsecutiveFailures++
		if e.State.ConsecutiveFailures >= 3 {
			step.Notified = e.notify(notify.ReasonRepeatedErrors)
		}
		step.Next = backoff.Delay(e.State.ConsecutiveFailures)
	default:
		step.Next = backoff.Steady
	}

	if r.Outcome == portal.PortalError && r.Err != nil {
		e.logf("check %s: %s (%v)", t, step.Outcome, r.Err)
	} else {
		e.logf("check %s: %s", t, step.Outcome)
	}
	e.save()
	return step
}

// Logout ends the session: when State.LoggedIn(), Portal.Logout(LogoutURL) (error only
// logged), then ClearSession and Save. Not logged in: nothing but Save.
func (e *Engine) Logout(ctx context.Context) {
	if e.State.LoggedIn() {
		if err := e.Portal.Logout(ctx, e.State.LogoutURL); err != nil {
			e.logf("logout: %v", err)
		}
		e.State.ClearSession()
	}
	e.save()
}

func (e *Engine) notify(r notify.Reason) notify.Reason {
	now := e.Now()
	if !notify.Allow(e.State.LastNotificationAt, now) {
		return ""
	}
	title, body := notify.Message(r)
	if err := e.Notifier.Notify(title, body); err != nil {
		e.logf("notify %s: %v", r, err)
	}
	e.State.LastNotificationAt = now
	return r
}

func (e *Engine) save() {
	if e.Save == nil {
		return
	}
	if err := e.Save(e.State); err != nil {
		e.logf("save state: %v", err)
	}
}

func (e *Engine) logf(format string, args ...any) {
	if e.Log != nil {
		e.Log(format, args...)
	}
}
