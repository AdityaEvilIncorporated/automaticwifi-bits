# T03: Backoff schedule, notification texts and hourly limit

## Goal
Implement the retry delays and the fixed notification texts with the one-per-hour limit (PROTOCOL.md section 8).

## Implement
- `backoff.Delay(failures)`: 1→5s, 2→15s, 3→60s, ≥4→Steady, ≤0→Steady.
- `notify.Message(r)`: exact strings from the docstring (tests compare them byte for byte).
- `notify.Allow(last, now)`: true if last is zero, or now.Sub(last) >= MinInterval, or last.After(now).

## Imitate
Standard library only; a switch statement is enough.

## Acceptance
`go test ./internal/backoff/ ./internal/notify/ -count=1 -run 'TestDelay|TestMessage|TestAllow'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
