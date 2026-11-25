# T22: Kotlin UrlConnectionHttp and PortalClient

## Goal
Implement Kotlin HTTP over HttpURLConnection and the PortalClient check flow, a port of internal/portal/client.go, tested against the Go mock portal binary.

## Implement
- `UrlConnectionHttp`: per the class KDoc. Cast to HttpURLConnection; for HttpsURLConnection set sslSocketFactory when non-null. Read with `(if (status >= 400) errorStream else inputStream)` (either may be null) limited to MAX_BODY bytes. Always `disconnect()` in finally. POST: `doOutput = true`, header `Content-Type: application/x-www-form-urlencoded; charset=UTF-8`.
- `PortalClient`: same steps and order as Go `CheckOnce`/`Login` (fields posted in protocol order). `onCampus`: IP literal check with a regex for dotted IPv4, else `resolver.resolves(host)`. `error` = a fixed description + exception class name, never the request body.
- Tests run with `bash scripts/test-android.sh --tests '*ScenariosTest*'` (Docker; the script builds the mock).

## Imitate
internal/portal/client.go (example). Example file: `internal/portal/client.go`.

## Acceptance
`bash scripts/test-android.sh --tests '*UrlConnectionHttpTest*' --tests '*ScenariosTest*'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
