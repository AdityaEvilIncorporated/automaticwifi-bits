# T01: PortalConfig defaults, validation, load/save

## Goal
Implement the portal configuration package: defaults, validation, JSON load/save. Every other package reads its settings from here, so the defaults must match PROTOCOL.md section 1 exactly.

## Implement
- `Default()`, `Validate()`, `PortalBase()`, `Load`, `Save`, `Dir`, `DefaultPath`: exactly as the docstrings in internal/config/config.go say.
- `Validate`: return `fmt.Errorf("%w: <json_key> ...", ErrInvalid)`; check the keys in the docstring order and return the first problem. Use `net/url` for the URLs (scheme http/https, non-empty Host).
- `Load`: start from `Default()`, `json.Unmarshal` the file into it (that overlays only present keys), then `Validate`. `os.ErrNotExist` → `Default(), nil`. JSON syntax/type errors → wrap ErrInvalid.
- `Save`: `Validate` first; `os.MkdirAll(dir, 0o700)`; write `path+".tmp"` with `os.WriteFile(..., 0o600)`, then `os.Rename`. `json.MarshalIndent(cfg, "", "  ")`.

## Imitate
Standard library only.

## Acceptance
`go test ./internal/config/ -count=1` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
