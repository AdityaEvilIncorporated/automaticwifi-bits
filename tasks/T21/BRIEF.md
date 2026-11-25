# T21: Kotlin PortalConfig and Classify (shared cases)

## Goal
Port PortalConfig and the pure classifiers to Kotlin so Android follows the same protocol. They must pass the same shared JSON cases as Go.

## Implement
Line-by-line port of internal/portal/classify.go (read it: it is merged and tested) and PROTOCOL.md sections 3-5, 7.
- `portalBase()`, `fromOverrides` (map snake_case keys; numbers may arrive as Int or Long: use `(v as Number).toInt()`), `Credentials.toString()`.
- `Classify.*`: Jsoup (`Jsoup.parse(body)`, `doc.select("form")`, `form.select("input[name]")`; attribute names are case-insensitive in Jsoup). `onPortal` with `java.net.URI` (catch exceptions → false; port -1 means 443).
- Searching: same needle approach as Go (plain and `\/`-escaped, case-insensitive host).

## Imitate
internal/portal/classify.go (example): copy its structure and helper names. Example file: `internal/portal/classify.go`.

## Acceptance
`bash scripts/test-android.sh --tests '*SharedCasesTest*'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
