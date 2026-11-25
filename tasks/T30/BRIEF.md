# T30: iOS Shortcut guide

## Goal
Write the iOS guide: how a student builds an Apple Shortcut that logs in to the BITS portal and a personal automation that runs it when joining campus Wi-Fi. Credentials live only in the Shortcut's Text actions on the device.

## Implement
Write ios/SHORTCUT_GUIDE.md (Markdown, plain English, numbered steps a first-year student can follow). Sections:
1. What this does and its limits (runs on Wi-Fi join only; no background keepalive on iOS; Shortcuts may show a banner).
2. Privacy: ID and password are stored in two **Text action**s inside the Shortcut; if Shortcuts iCloud sync is on they sync through iCloud; recommend iCloud **Advanced Data Protection**; never share the Shortcut once filled in; use placeholders YOUR_ID / YOUR_PASSWORD in the guide.
3. Build the Shortcut, action by action: Text (YOUR_ID), Text (YOUR_PASSWORD), **Get Contents of URL** `http://neverssl.com` → **Match Text** with the regex `fgtauth\?([0-9A-Za-z]+)` (also explain the magic is `[0-9A-Za-z]+`) → If no match: stop (already online / not campus) → Get Contents of URL `https://fw.bits-pilani.ac.in:8090/fgtauth?<magic>` (loads the form) → Get Contents of URL `https://fw.bits-pilani.ac.in:8090/`, Method **POST**, Request Body **Form**, fields `username`, `password`, `magic`, `4Tredir` (value `http://neverssl.com/`) → optional: Show Notification only on failure ("authentication failed" in the result).
4. The **Personal Automation**: Automation tab → New → **Wi-Fi** → choose the campus network names (SSIDs; leave a TODO table "SSID names: to be confirmed on campus") → **Run Immediately**, notify off.
5. Troubleshooting: Shortcuts follows redirects, so the Match Text runs on the final page; if field names differ, run `autowifi capture` on a laptop on campus and change the field names; no keepalive on iOS: if the session expires, toggle Wi-Fi or run the Shortcut.
The test (internal/docscheck TestShortcutGuide) lists the words that must appear.

## Imitate
Tone and structure of protocol/PROTOCOL.md; keep it under ~200 lines.

## Acceptance
`go test ./internal/docscheck/ -count=1 -run TestShortcutGuide` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
