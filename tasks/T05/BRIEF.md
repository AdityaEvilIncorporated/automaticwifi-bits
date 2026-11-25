# T05: Protocol classifiers (probe, login page, login reply, keepalive, host rule)

## Goal
Implement the pure FortiGate protocol classifiers. They decide if we are captive, extract the one-time magic, parse the login form and classify the login reply. They must pass the shared JSON cases that the Kotlin port also runs.

## Implement
Read protocol/PROTOCOL.md sections 2-5 and 7 carefully; the JSON cases are normative.
- `OnPortal(cfg, rawURL)`: url.Parse; scheme == "https"; strings.EqualFold(u.Hostname(), cfg.PortalHost); port == cfg.PortalPort, where an empty port counts as 443. Any parse error → false.
- `ClassifyProbe`: section 3. For the body search, build the needle `https://<host>:<port>/fgtauth?` and also the escaped form `https:\/\/<host>:<port>\/fgtauth?`; search case-insensitively for the host (lower-case both the body copy you search and the needle, but take the magic characters from the original body at the same offset). Magic = the run of `[0-9A-Za-z]` right after the needle. For a 3xx Location, use OnPortal plus path == "/fgtauth"; magic = u.RawQuery (must be non-empty and all `[0-9A-Za-z]`, else malformed). MagicURL = cfg.PortalBase()+"/fgtauth?"+magic.
- `ParseLoginPage`: section 4 using ParseForms (T04). Return `ErrFormChanged` itself.
- `ClassifyLoginReply`: section 5. Search Location and body (plain and `\/`-escaped) for `<base>/keepalive?<token>` and `<base>/logout?<token>`; build the returned URLs from cfg.PortalBase() and the token (never copy the matched text). A keepalive on another host is simply not found.
- `KeepaliveAlive`: section 7.
Edge cases are all in the JSON files: look-alike hosts, wrong port, http, uppercase host, escaped slashes, magic ending at `&`.

## Imitate
Keep helpers unexported in classify.go (e.g. findPortalToken(cfg, text, path) (token string, found bool)).

## Acceptance
`go test ./internal/portal/ -count=1 -run 'TestClassifyProbe|TestParseLoginPage|TestClassifyLoginReply|TestKeepaliveSharedCases|TestOnPortal'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
