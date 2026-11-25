# Handoff

## Working agreement (from the user, 2026-11-25)
Do routine review actions (merge/reject/take, small follow-up fixes with tests, commits, pushes to
dev) without asking; report afterwards. Stop only for what CLAUDE.md reserves for a human.

## Current task
Deploy v0.1.0 (pre-release, "beta: not yet verified on the real BITS portal"). Definition of done
is complete (reports/TEST_REPORT.md, SHIPPING.md). BLOCKED on two owner-only GitHub settings
(HUMAN_NEEDED.md, human notified 2026-11-26): gh token lacks Secrets write (HTTP 403), Pages not
enabled. Review queue empty, no NEEDS_SUPERVISOR, all HARD tasks done.

## Release state (2026-11-26)
- Signing key: ~/.agent-secrets provider `android-automaticwifi-bits` (4 keys). Not in the repo.
- Local dry run with the real key: signed APK + 6 desktop assets + SHA256SUMS, check-dist OK.
- Security review: fixed/accepted (DECISIONS.md). Second opinion release: round 1 block (8 fixed),
  round 2 concerns (1 fixed). govulncheck clean with toolchain go1.26.11.
- RELEASE_TAG=v0.1.0; RELEASE_BETA holds the label; release.yml checks tag format, RELEASE_TAG
  and that the commit is on dev before the signing key is used.
- GitHub default branch is dev; main exists only locally (never push it). So the playbook's
  `gh pr create --base main` cannot run; DEPLOYED.md records this.

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
1. When the human replies done: `bs-secret.sh run android-automaticwifi-bits -- bash scripts/set-gh-secrets.sh`
   (expect 4 ANDROID_* secrets listed).
2. `git tag v0.1.0 <tested dev commit> && git push origin v0.1.0`; watch the release run
   (gh run list/watch if permitted, else the release page) and the pages run (re-run pages via a
   dev push or workflow_dispatch after Pages is enabled).
3. Verify: release is a pre-release with 7 assets; Pages URL shows the banner and links resolve;
   write DEPLOYED.md, notify LIVE with both URLs.

## Known problems
- Desktop GUI paths (tray, osascript/Get-Credential/zenity dialogs, toasts, Run key, LaunchAgent
  load) are build-checked and unit-tested but not exercised on real Windows/Linux desktops; the
  macOS tray was smoke-run here (starts, off_campus, clean SIGTERM exit). Release checklist item.
- bs-task.py merge conflicts when dev already has tasks/<ID>/REVIEW_L1.md (rejected tasks) and then
  deletes task/<ID>. Do not commit REVIEW_L1.md of tasks still in flight; recover with git fsck.
- ParseForms (T04) relies on the html5 tree: a <form> placed directly inside <table> loses its
  inputs (html5 foster-parenting). Standard FortiGate pages nest the table inside the form, so OK
  for now; if `capture` on campus shows no fields, switch to a tokenizer-based pass.
- APK not run on a device or emulator here (no virtualization in the amd64 container): built,
  unit-tested and linted only. Install-and-login on a real phone is on the campus checklist.
- Campus facts (fields, cert trust, SSIDs) unconfirmed by design: campus checklist in PLAN.md.

## Verify commands
bash scripts/verify.sh               # everything (needs Docker for Kotlin)
SKIP_TESTS=1 bash scripts/verify.sh  # what the gate runs before task tests
go test ./...                        # Go only
bash scripts/test-android.sh         # Kotlin core only
