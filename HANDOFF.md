# Handoff

## Working agreement (from the user, 2026-11-25)
Do routine review actions (merge/reject/take, small follow-up fixes with tests, commits, pushes to
dev) without asking; report afterwards. Stop only for what CLAUDE.md reserves for a human.

## Current task
T14 done (2026-11-25): desktop agent, CLI, tray, prompts, lock, e2e. Next HARD: T24 (Android app).
Earlier: review rounds 1-3 merged everything except T24/T32; T07 done by me (17d553a).

## Done
- intake read; PLAN.md, ARCHITECTURE.md, SCHEDULE.md, DECISIONS.md written.
- protocol/PROTOCOL.md + protocol/testcases/*.json (probe, login_page, login_reply, keepalive, scenarios).
- internal/mockportal + cmd/mockportal implemented and tested (test infrastructure).
- Go contracts (panic stubs) + failing tests for every package; Kotlin core contracts (TODO stubs)
  + failing tests; docscheck tests for the iOS guide and site; scripts/check-dist.sh.
- verify.sh: Go (gofmt, vet x3 OS, builds x3 OS, go test -race) + Kotlin via scripts/test-android.sh.
- TASKS.json (22 tasks) and tasks/<ID>/BRIEF.md for every EASY/MEDIUM task.

## Half-done
- T07 second opinion pending (see DECISIONS.md); do it via a task branch or at release.
- T06 brief updated: unwrap *url.Error (URLs carry magic/tokens); new TestErrorsNeverContainMagicOrToken.

## Supervisor fixes after review (commit "review fixes")
- creds file store: corrupt JSON -> ErrNotFound, never the json error (it can quote file bytes). Test added.
- Windows toast uses PowerShell's registered AppUserModelID (unregistered ids are dropped). Untested on Windows: campus/Windows checklist.
- PROTOCOL 3.1: redirect magic must be [0-9A-Za-z]+ else malformed; new probe.json case; Kotlin fixed. Go T05 must pass it.
- Kotlin Engine: notifier exceptions caught so state is still saved. Test added.
- site: macOS 15+ Gatekeeper path (System Settings > Privacy & Security > Open Anyway).

## Decisions
See DECISIONS.md.

## Next 3 steps
1. bs-task.py queue: review T32 (release script/workflows; unblocked by T14).
2. T24 Android app (HARD, mine): app module, PortalTls.kt, Keystore store, APK via Docker
   android-sdk image; run second-opinion on a task/T24 branch before merging.
3. Then Definition of done (security-reviewer, e2e-runner, TEST_REPORT, SHIPPING, deploy).

## Known problems
- Desktop GUI paths (tray, osascript/Get-Credential/zenity dialogs, toasts, Run key, LaunchAgent
  load) are build-checked and unit-tested but not exercised on real Windows/Linux desktops; the
  macOS tray was smoke-run here (starts, off_campus, clean SIGTERM exit). Release checklist item.
- bs-task.py merge conflicts when dev already has tasks/<ID>/REVIEW_L1.md (rejected tasks) and then
  deletes task/<ID>. Do not commit REVIEW_L1.md of tasks still in flight; recover with git fsck.
- ParseForms (T04) relies on the html5 tree: a <form> placed directly inside <table> loses its
  inputs (html5 foster-parenting). Standard FortiGate pages nest the table inside the form, so OK
  for now; if `capture` on campus shows no fields, switch to a tokenizer-based pass.
- No Android SDK locally; APK build planned via Docker image in T24.
- Campus facts (fields, cert trust, SSIDs) unconfirmed by design: campus checklist in PLAN.md.

## Verify commands
bash scripts/verify.sh               # everything (needs Docker for Kotlin)
SKIP_TESTS=1 bash scripts/verify.sh  # what the gate runs before task tests
go test ./...                        # Go only
bash scripts/test-android.sh         # Kotlin core only
