# Handoff

## Current task
Planning done; waiting for "approve automaticwifi-bits" (.agent/PLAN_APPROVED).

## Done
- intake read; PLAN.md, ARCHITECTURE.md, SCHEDULE.md, DECISIONS.md written.
- protocol/PROTOCOL.md + protocol/testcases/*.json (probe, login_page, login_reply, keepalive, scenarios).
- internal/mockportal + cmd/mockportal implemented and tested (test infrastructure).
- Go contracts (panic stubs) + failing tests for every package; Kotlin core contracts (TODO stubs)
  + failing tests; docscheck tests for the iOS guide and site; scripts/check-dist.sh.
- verify.sh: Go (gofmt, vet x3 OS, builds x3 OS, go test -race) + Kotlin via scripts/test-android.sh.
- TASKS.json (22 tasks) and tasks/<ID>/BRIEF.md for every EASY/MEDIUM task.

## Half-done
Nothing.

## Decisions
See DECISIONS.md.

## Next 3 steps
1. After approval: bs-task.py queue; T07 (tlsconf) myself, then second-opinion.sh task T07.
2. Review worker results as they arrive (T01-T04, T08, T10-T12, T21, T23, T30, T31 run first).
3. T14 agent/CLI/tray once T06, T08, T09 are merged.

## Known problems
- internal/deps pins modules in go.mod; remove imports there as real packages import them (T14).
- No Android SDK locally; APK build planned via Docker image in T24.
- Campus facts (fields, cert trust, SSIDs) unconfirmed by design: campus checklist in PLAN.md.

## Verify commands
bash scripts/verify.sh               # everything (needs Docker for Kotlin)
SKIP_TESTS=1 bash scripts/verify.sh  # what the gate runs before task tests
go test ./...                        # Go only
bash scripts/test-android.sh         # Kotlin core only
