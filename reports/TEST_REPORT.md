# Test report — AutoWiFi BITS v0.1.0 (beta)

Date: 2026-11-25/26. Commit: dev at the "Windows dialog: stop on error" commit and later doc commits.

## Summary
Everything that can run on this Mac passes. **Nothing has been tested against the real BITS
FortiGate portal, on a real Windows or Linux desktop, on an Android device/emulator, or on an
iPhone.** That is why v0.1.0 is a pre-release labelled "beta: not yet verified on the real BITS
portal", with the campus verification checklist on the download page.

## What ran (and passed)
| Check | Result | How |
|---|---|---|
| Go: gofmt, vet, builds for darwin/windows/linux | pass | `bash scripts/verify.sh` |
| Go tests, `-race`, 17 packages | pass | verify.sh (go1.26.11) |
| Kotlin core: 33 tests (Engine, Scenarios, SharedCases, PortalTls, UrlConnectionHttp) incl. the shared JSON cases and a live mock portal | pass | verify.sh → scripts/test-android.sh (Docker) |
| Android app: unit tests (CredentialCodec, Logic) + lint, signed release APK with the real key | pass | `scripts/build-apk.sh` (Docker, amd64) in the release dry run |
| Release dry run: 7 assets, SHA256SUMS, check-dist | pass | `build-apk.sh && build-release.sh v0.1.0 && check-dist.sh v0.1.0` |
| gitleaks (whole history) | no leaks | verify.sh |
| semgrep | clean | verify.sh |
| govulncheck | no vulnerabilities (28 stdlib ones fixed by `toolchain go1.26.11`) | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Download page: title, beta banner, checklist anchor, 7 links to releases/download/v0.1.0, no console errors, no overflow at 1280×800 and 390×844 | 9/9 pass | Playwright (`e2e/site-check.mjs`), screenshots in reports/screenshots/ |
| docscheck (page + iOS guide content) | pass | `go test ./internal/docscheck/` |

## PLAN.md flows
| Flow | Automated coverage | Not covered (manual / campus) |
|---|---|---|
| 1 Desktop setup | cmd/autowifi e2e: setup → keyring (mock), autostart file, forget; page screenshot | Gatekeeper/SmartScreen prompts; native dialogs on Windows/Linux; macOS tray only smoke-run |
| 2 Android setup | Kotlin core + app unit tests; APK builds and signs | Install, permissions, battery prompt, Wi-Fi callbacks on a device (no emulator here) |
| 3 Silent login + keepalive | Go agent/engine e2e against the mock portal (login, keepalive, re-login after expiry, logout); Kotlin ScenariosTest | Real portal; > 1 hour session |
| 4 Off campus / online | Go + Kotlin engine and classify tests | Real networks |
| 5 Errors and backoff | engine tests with fake clock; wrong password; hourly notification limit | Real wrong-password behaviour on the portal |
| 6 Capture | capture tests vs mock (trusted and untrusted CA, pin, control-character stripping) | Field names / TLS result on the real portal |
| 7 iOS Shortcut | docscheck on the guide | Building and running the Shortcut on an iPhone; SSID names |

## Reviews
- Security review (whole repo): 0 critical/high; 1 medium + 8 low, fixed or accepted with reasons (DECISIONS.md).
- Second opinion (Gemini), release: round 1 "block" (8 issues, all fixed with tests where testable);
  round 2 "concerns" (1 low, fixed).

## Not applicable
- `docker compose up --build` / production Playwright on a fresh database: there is no server or
  database in this product (PLAN.md non-goals). The "production build" is the release dry run above.

## Known gaps
- Windows PowerShell credential script not executed on Windows (pwsh parse check in Docker hung).
- Unsigned desktop binaries (no notarisation/Authenticode): users see OS warnings, documented on the page.
- No build-provenance attestation yet.
