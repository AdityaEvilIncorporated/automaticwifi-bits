# T08: Credential stores: keyring, file fallback, memory, NewDefault

## Goal
Implement credential storage: the OS keyring via go-keyring, a mode-600 file fallback for Linux without Secret Service, and an in-memory store. Passwords must never leak through formatting.

## Implement
Docstrings in internal/creds/*.go are the spec.
- `String`/`GoString`: exact formats. Because both are value-receiver methods, `%v`, `%+v`, `%#v`, `%s` and pointers all go through them.
- Shared `Set` validation: `u := strings.TrimSpace(c.Username)`; empty u or empty password → ErrEmpty. Store `{u, password}`.
- Keyring: `keyring.Get/Set/Delete(Service, Account)`; `keyring.ErrNotFound` → ErrNotFound (wrap with %w); Delete of a missing item → nil. Bad JSON → ErrNotFound.
- File: atomic write as in config.Save; Get checks `st.Mode().Perm()&0o077 != 0` → `os.Chmod(path, 0o600)` first.
- Memory: a struct with a sync.Mutex and a *Credentials.
- `KeyringProbe`: Set/Get/Delete under (Service, "probe") with a random value; return the first error.
- `NewDefault`: as documented; the marker file is `fallbackPath+".warned"`.

## Imitate
Standard library + github.com/zalando/go-keyring (already in go.mod).

## Acceptance
`go test ./internal/creds/ -count=1` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
