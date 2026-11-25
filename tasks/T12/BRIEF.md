# T12: GitHub latest-release check

## Goal
Check GitHub Releases for a newer stable version (informational only).

## Implement
Per internal/update/update.go docstrings.
- `Latest`: `http.NewRequestWithContext`, headers, `io.LimitReader(resp.Body, 1<<20)`, json decode; status != 200 → `fmt.Errorf("github releases: HTTP %d", code)`.
- `Newer`: strip a leading "v"; split off "-…" suffix (pre-release); latest with a suffix → false; parse exactly three dot-separated non-negative integers (`strconv.Atoi`), else false; compare lexicographically by number.

## Imitate
Standard library only.

## Acceptance
`go test ./internal/update/ -count=1` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
