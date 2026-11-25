# T02: SessionState persistence

## Goal
Persist the small session state (keepalive URL, failure counter, rejected flag, last notification time) so it survives restarts. A corrupt file must never stop the agent.

## Implement
- `LoggedIn`, `ClearSession`, `Load`, `Save` per internal/state/state.go docstrings.
- `Load`: missing file or invalid JSON → zero value, nil error; other read errors returned.
- `Save`: MkdirAll 0700, write `.tmp` 0600, rename.

## Imitate
Same atomic-write pattern as T01 (config.Save).

## Acceptance
`go test ./internal/state/ -count=1` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
