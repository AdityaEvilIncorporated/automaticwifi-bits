# Handoff

## Current task
Review round 1 done (2026-11-25). Merged: T01 T02 T03 T04 T08 T10 T11 T12 T21 T23 T31.
T30: rejected once, resubmission good but merge conflicted (REVIEW_L1.md on both sides) and the
merge script dropped the branch; recovered commit dcc95c8, took it, added the Get Group step, done.
A redundant re-run of T30 (branch task/T30, based on stale dev, lacks the Get Group action) was NOT
merged; dev keeps the reviewed guide. The branch can be deleted.
Round 2 merged: T10W, T22. Round 3 merged: T05, T09. T07 (tlsconf) done by me: 17d553a.

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
1. bs-task.py queue: review T06, then T13 (both unblocked by T07).
2. Start T14 (agent/CLI/tray) in parallel; it needs T06/T13 merged before its e2e test.
3. T14 agent/CLI/tray once T06, T09, T13 are merged. In T14: open update URLs only if they start with https://github.com/.

## Known problems
- bs-task.py merge conflicts when dev already has tasks/<ID>/REVIEW_L1.md (rejected tasks) and then
  deletes task/<ID>. Do not commit REVIEW_L1.md of tasks still in flight; recover with git fsck.
- ParseForms (T04) relies on the html5 tree: a <form> placed directly inside <table> loses its
  inputs (html5 foster-parenting). Standard FortiGate pages nest the table inside the form, so OK
  for now; if `capture` on campus shows no fields, switch to a tokenizer-based pass.
- internal/deps pins modules in go.mod; remove imports there as real packages import them (T14).
- No Android SDK locally; APK build planned via Docker image in T24.
- Campus facts (fields, cert trust, SSIDs) unconfirmed by design: campus checklist in PLAN.md.

## Verify commands
bash scripts/verify.sh               # everything (needs Docker for Kotlin)
SKIP_TESTS=1 bash scripts/verify.sh  # what the gate runs before task tests
go test ./...                        # Go only
bash scripts/test-android.sh         # Kotlin core only
