# AutoWiFi BITS: plan

## Goal
Log BITS Pilani students into the FortiGate campus Wi-Fi captive portal
(`https://fw.bits-pilani.ac.in:8090`) automatically and silently: when they join a campus
network, and again whenever the session expires. No windows, no prompts after setup, no
notifications on success.

## Users
BITS Pilani students using their own campus ID and password, on macOS 12+, Windows 10+,
Ubuntu 22.04+ (should), Android 8.0+ (API 26), and iOS through Apple Shortcuts.

## Deliverables
| Platform | What | How it triggers |
|---|---|---|
| macOS / Windows / Linux | `autowifi` Go single binary, tray/menu-bar icon, login item | start, network change (interface address poll every 15 s), every 5 min, "Log in now" |
| Android | `autowifi-bits.apk` (Kotlin) | Wi-Fi connect (`registerNetworkCallback` with PendingIntent), WorkManager every 15 min, "Log in now" button |
| iOS | `ios/SHORTCUT_GUIDE.md`: build-it-yourself Shortcut + "When joining Wi-Fi" personal automation | joining a listed campus SSID |
| All | `protocol/PROTOCOL.md` + `protocol/testcases/*.json` + mock portal | n/a |
| Web | GitHub Pages download page (`site/index.html`) + GitHub Releases assets | n/a |

## User flows (each one is tested end to end)
1. **First-time setup (desktop).** Download the asset from the page, pass the unsigned-app
   prompt, run `autowifi setup` (or "Set ID and password…" in the tray, a native dialog),
   credentials go to the OS store, `autowifi autostart on` registers the login item.
   *Test:* `cmd/autowifi` e2e test with a mock keyring + temp HOME: setup → stored, autostart
   file written; site check (docscheck) + Playwright screenshot of the page.
2. **First-time setup (Android).** Install APK, enter ID/password, grant notification
   permission, accept or refuse the battery-optimisation prompt (works either way).
   *Test:* Kotlin core tests + manual APK checklist on an emulator (reports/).
3. **Silent background login.** Network change/timer → probe → fgtauth → login →
   keepalive every 5 min → fresh login when keepalive fails.
   *Test:* Go e2e: `autowifi run` against the mock portal (logs in, keeps alive, re-logs after
   `/__mock/expire`); Kotlin `ScenariosTest`; shared JSON cases in both languages.
4. **Off campus / already online.** Portal host does not resolve, or the probe loads normally:
   nothing happens, no notification.
   *Test:* engine tests (Go + Kotlin), client tests with a failing resolver, scenario "already
   logged in".
5. **Errors and backoff.** Portal errors retry at 5 s, 15 s, 60 s, then 5 min; one quiet
   notification per hour on rejected credentials (and retries pause), 3rd consecutive error,
   or changed form.
   *Test:* engine tests (fake clock), e2e with `/__mock/fail` and a wrong password.
6. **On-campus capture.** `autowifi capture` prints form action, field names/types, TLS
   verification result and the SPKI pin, never values.
   *Test:* capture tests against the mock (trusted and untrusted CA).
7. **iOS Shortcut.** Follow the guide to build the Shortcut and automation.
   *Test:* docscheck test for required content; a manual run on an iPhone is on the campus
   checklist (cannot be automated here).

## Campus verification checklist (release step; does not block the build)
Owned by the human, on campus, before announcing a release:
- [ ] `autowifi capture` while the portal shows: field names match config defaults, action is `/`
- [ ] "TLS verified by OS trust store: yes" on macOS, Windows, Linux; Android logs the same
- [ ] wrong password → one notification, no retry storm; right password → silent login
- [ ] session survives > 1 hour with keepalive
- [ ] SSID names written into ios/SHORTCUT_GUIDE.md; Shortcut tested on an iPhone
- [ ] if any field differs: change config defaults (both languages + shared cases), re-release

## Non-goals
Backend servers, databases, accounts or hosted apps; Rust/UniFFI; disabling TLS verification;
credentials in code, logs, the repo or the page; a native iOS app; SSID-based triggering on
desktop/Android; code signing/notarisation (later); telemetry of any kind.

## Pre-mortem (what could go wrong, and what the plan does about it)
- **Credential theft by a look-alike portal** (rogue AP redirecting to its own `fgtauth`):
  credentials are only ever POSTed to `https://{portal_host}:{portal_port}` (host rule, shared
  cases "look-alike host", "wrong port", "http"), and TLS is always verified. DNS spoofing of
  the real host name fails TLS. The pin is a SPKI hash of the real key, host-bound.
- **Account lockout from retrying a wrong password**: `rejected` pauses all automatic retries
  until the credentials change; one notification per hour max.
- **Portal differs from the standard FortiGate form**: everything is configurable; `capture`
  shows what to change; `form_changed` notifies instead of failing silently.
- **Android sends the probe over mobile data** (captive Wi-Fi is not "validated"): every request
  goes through `network.openConnection` of the Wi-Fi `Network` from the callback.
- **Android blocks cleartext HTTP** (API 28+): `network_security_config` allows cleartext only
  for `neverssl.com` and `connectivitycheck.gstatic.com`.
- **Android background limits** (Doze, 12+ restrictions): PendingIntent network callback (no
  running service), expedited one-time work, 15-min periodic work, keepalive chained as delayed
  one-time work; battery exemption requested once; POST_NOTIFICATIONS asked on Android 13+.
- **macOS Keychain prompt on every update** (unsigned binary, new code hash): documented on the
  page ("Always Allow"); signing is the later fix.
- **Windows GUI binary has no console**: `setup` uses a native dialog; CLI commands attach to
  the parent console when started from a terminal.
- **Linux without tray support or Secret Service**: the agent works without a tray; credentials
  fall back to a mode-600 file with a one-time warning.
- **Two agents running** (login item + manual start): single-instance lock in the config dir.
- **Secrets in logs**: `Credentials` redacts itself; logs carry outcomes only; tests check
  errors never contain the password.
- **Clock jumps / sleep**: hourly limit treats a "future" last-notification as allowed; timers
  re-armed after each run; a wake from sleep changes addresses → network trigger.
- **Load**: one GET per client per 5 min to neverssl.com plus jitter (±20 s) so thousands of
  clients do not hit the portal in lockstep; update check once a day (GitHub unauthenticated
  limit 60/h per IP; campus NAT shares IPs, so failures are silent and retried next day).
- **Data loss**: there is no user data besides credentials (OS store) and small config/state
  files; corrupt state is ignored, corrupt config falls back with an error message.

## Release and deploy
`.agent/DEPLOY_MODE` is `auto`. "Deploy" for this project = a public GitHub repository
(`RELEASE_REPO`), a GitHub Release built by `.github/workflows/release.yml` from a
`v*` tag (desktop binaries, APK, SHA256SUMS), and GitHub Pages serving `site/`. No servers,
no paid plans, no secrets beyond the GitHub token Actions already has. The local `origin` stays
the working remote; the GitHub remote is added at deploy time.
