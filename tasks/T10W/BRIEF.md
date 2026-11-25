# T10W: Autostart: Windows Run key

## Goal
Windows start-at-login via the HKCU Run key.

## Implement
- `install`: `filepath.IsAbs(exe)` else error; `registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Run", registry.SET_VALUE)` (Go string literal, backslashes escaped); `SetStringValue("AutoWiFiBITS", RunValue(exe))`.
- `uninstall`: DeleteValue; `registry.ErrNotExist` → nil.
- `installed`: QUERY_VALUE, GetStringValue; ErrNotExist → false, nil.
- `home` is unused on Windows.

## Imitate
golang.org/x/sys/windows/registry (in go.mod).

## Acceptance
`GOOS=windows go vet ./internal/autostart/ && GOOS=windows go build ./internal/autostart/` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
