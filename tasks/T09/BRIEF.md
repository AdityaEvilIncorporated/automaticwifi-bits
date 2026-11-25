# T09: Engine decision logic

## Goal
Implement the engine: the per-trigger decision logic of PROTOCOL.md sections 7-8 (keepalive, fresh login, backoff, pause on rejection, hourly notifications). It is pure logic over injected interfaces, fully pinned by engine_test.go.

## Implement
Follow the numbered rules in the `Run` docstring in internal/engine/engine.go exactly and in that order; then `Logout`.
- Use `backoff.Delay`, `backoff.Steady`, `notify.Allow`, `notify.Message`.
- Helper `func (e *Engine) notify(r notify.Reason) notify.Reason` returning r when sent, "" otherwise.
- Log lines: `e.logf("check %s: %s", trigger, outcome)`; for PortalError include `r.Err` (it never contains secrets). A nil Log or Save must be safe.

## Imitate
Plain methods on Engine; no goroutines, no time.Sleep.

## Acceptance
`go test ./internal/engine/ -count=1` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
