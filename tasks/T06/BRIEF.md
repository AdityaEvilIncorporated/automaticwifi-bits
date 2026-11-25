# T06: Portal network client (probe, login, check, keepalive, logout)

## Goal
Implement the network side of one check: campus DNS test, probe with fallback, login (GET fgtauth page, POST form), keepalive and logout. It must pass the shared scenarios against the in-process mock portal.

## Implement
Read PROTOCOL.md section 2 and the docstrings in internal/portal/client.go.
- `NewClient`: two `*http.Client`s with `CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }`, transports cloned from `http.DefaultTransport.(*http.Transport).Clone()` with `Proxy = nil`; the portal transport gets `TLSClientConfig` from `tlsconf.PortalTLSConfig(cfg.PortalHost, cfg.PinnedCertSHA256, nil)`.
- `OnCampus`: `net.ParseIP(host) != nil` → true without lookup; else LookupHost with a 5 s timeout.
- Reading bodies: `io.ReadAll(io.LimitReader(resp.Body, MaxBody))`, always close the body.
- `Login`: build the POST body with `url.Values`-style escaping BUT in protocol order (username, password, magic, redirect): url.Values.Encode sorts keys, so build the string yourself with `url.QueryEscape(k)+"="+url.QueryEscape(v)` joined by "&". The test checks the exact body.
- Errors: wrap with a fixed description and the host only, e.g. `fmt.Errorf("login POST to %s: %w", cfg.PortalHost, err)`. *url.Error contains the URL, which is fine (no credentials in URLs), but never include the body.
- `CheckOnce`: OnCampus → ProbeOnce (error → PortalError) → Online/Foreign return → Malformed → FormChanged → get() (errors.Is ErrNotFound → NoCredentials; other error → PortalError) → Login.

## Imitate
The tests in client_test.go show how a Client is built for the mock (testClient).

## Acceptance
`go test ./internal/portal/ -count=1 -run 'TestScenarios|TestCheckOnce|TestOnCampus|TestCredentialsReadOnly|TestCredsGet|TestProbe|TestLogin|TestKeepaliveAndLogout|TestMagic'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
