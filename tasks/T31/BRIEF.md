# T31: Static download page

## Goal
Write the static GitHub Pages download page: links to every release asset, the unsigned-app bypass steps per OS, setup commands, and the campus verification checklist.

## Implement
Write site/index.html: one self-contained HTML5 file, inline CSS only, no scripts, no external fonts or images, readable on a phone (`<meta name="viewport" ...>`), light and dark (`prefers-color-scheme`). Title exactly `AutoWiFi BITS`.
Content:
- One-line pitch; "Free, open source, no servers: your ID and password never leave your device except to the campus portal."
- Download table: one row per asset in internal/docscheck Assets, each linking `https://github.com/<RELEASE_REPO>/releases/latest/download/<asset>`; plus a link to SHA256SUMS.txt and the repo.
- macOS: unzip, **right-click → Open** the first time (Gatekeeper; unsigned app), then `./autowifi setup` and `./autowifi autostart on`; Keychain may ask once per update: choose "Always Allow".
- Windows: unzip, SmartScreen: **More info → Run anyway**, run `autowifi.exe`, use the tray menu "Set ID and password…"; `autowifi setup` also works in a terminal.
- Linux: tar -xzf, `./autowifi setup`, `./autowifi autostart on`; tray needs AppIndicator support; without a Secret Service keyring a mode-600 file is used.
- Android: download the APK, allow **Install unknown apps** for your browser, open the app, enter ID and password, allow notifications, accept the battery optimisation prompt (recommended).
- iPhone: link to the guide on GitHub (`https://github.com/<RELEASE_REPO>/blob/main/ios/SHORTCUT_GUIDE.md`).
- Troubleshooting: `autowifi capture` prints the portal form fields (never values).
- "Campus verification checklist" section copied from PLAN.md (portal fields, certificate, SSIDs, wrong-password behaviour, 1-hour keepalive).
- Footer: not affiliated with BITS Pilani.

## Imitate
No example; keep it simple semantic HTML with a small CSS block.

## Acceptance
`go test ./internal/docscheck/ -count=1 -run TestDownloadPage` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
