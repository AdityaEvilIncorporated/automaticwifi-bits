# T23: Kotlin Engine, Backoff, notifyAllowed

## Goal
Port the engine decision logic to Kotlin so Android behaves exactly like desktop: keepalive, fresh login, backoff, pause on rejection, hourly notifications.

## Implement
Follow the rules in the doc comment of `Run` in internal/engine/engine.go (Go is the reference; read the merged implementation if it exists) with these Kotlin names: `Backoff.delayMs`, `notifyAllowed`, `Engine.run`, `Engine.logout`, `ports.notify("AutoWiFi BITS", reason.body)`, `ports.save(state)` at the end of every run/logout. Step.outcome is the Outcome wire name, or "keepalive_ok" / "paused".

## Imitate
internal/engine/engine.go (example). Example file: `internal/engine/engine.go`.

## Acceptance
`bash scripts/test-android.sh --tests '*EngineTest*'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
