# Architecture

Two clients sharing one written protocol and one set of JSON test cases. No server, no
database. Source of truth for behaviour: `protocol/PROTOCOL.md` + `protocol/testcases/`.

```
protocol/            PROTOCOL.md, testcases/*.json        (shared, normative)
cmd/autowifi/        desktop CLI + agent entry point       (T14, HARD)
cmd/mockportal/      mock FortiGate binary (for Kotlin)    (done, test infra)
internal/
  config/            PortalConfig, defaults, load/save     (T01)
  state/             SessionState persistence              (T02)
  backoff/           retry schedule                        (T03)
  notify/            reasons, texts, hourly limit (T03); OS notifiers (T11)
  portal/            form.go (T04) classify.go (T05) client.go (T06) capture.go (T13)
  tlsconf/           OS trust store + host-only SPKI pin   (T07, HARD)
  creds/             go-keyring store, file fallback, memory store (T08)
  engine/            decision logic per trigger            (T09)
  agent/             goroutine loop, network watcher       (T14, HARD)
  autostart/         LaunchAgent / XDG autostart / Run key (T10, T10W)
  update/            GitHub latest-release check           (T12)
  mockportal/        in-process mock portal                (done, test infra)
  docscheck/         content tests for the guide and page  (tests only)
  deps/              pins modules in go.mod until real imports exist (delete later)
android/
  core/              Kotlin/JVM: PortalConfig, Classify (T21), UrlConnectionHttp,
                     PortalClient (T22), Engine (T23), PortalTls (T24)
  app/               Android app: receiver, worker, settings UI, Keystore store (T24, HARD)
ios/SHORTCUT_GUIDE.md                                      (T30)
site/index.html      GitHub Pages download page            (T31)
scripts/             verify.sh, test-android.sh, build-release.sh (T32), check-dist.sh
.github/workflows/   release.yml, pages.yml                (T32)
```

## Desktop data flow
```
trigger (start | 15 s address poll change | Step.Next timer | tray "Log in now" | setup)
  → agent (one goroutine, coalesces triggers)
  → engine.Run(trigger)
      paused?  keepalive due?  → portal.Client.CheckOnce(creds.Get)
          OnCampus (DNS) → ProbeOnce (no redirects) → ClassifyProbe
          → creds → GET fgtauth → ParseLoginPage → POST {portal base}{login_path}
          → ClassifyLoginReply
      → state.SessionState updated, notify (≤ 1/h), Step{Outcome, Next}
  → state.Save, tray status text, timer re-armed (Next ± 20 s jitter when Next = 5 min)
```

## Interfaces (the contracts are the real files; this is the map)
- `portal.Client{Cfg, Probe, Portal *http.Client, Resolver, Roots}`: `OnCampus`, `ProbeOnce`,
  `Login`, `CheckOnce(get) Result`, `Keepalive`, `Logout`, `Capture`.
- `engine.Engine{Portal, Creds, Notifier, State, Now, Save, Log}.Run(ctx, Trigger) Step`.
- `creds.Store`: `Get/Set/Delete`; `NewKeyring`, `NewFile`, `NewMemory`, `NewDefault(probe, path, warn)`.
- `tlsconf.PortalTLSConfig(host, pin, roots)`, `VerifySystem`, `Inspect`, `SPKISHA256`.
- Kotlin mirrors these names: `Classify`, `PortalClient : PortalApi`, `Engine(portal, ports, state)`.

## Data model (files, not a database)
| What | Where | Mode |
|---|---|---|
| Credentials | OS store: service `autowifi-bits`, account `campus-login`, JSON `{username,password}`; Linux fallback `~/.config/autowifi-bits/credentials` | OS / 0600 |
| PortalConfig | `<UserConfigDir>/autowifi-bits/config.json` (absent = defaults) | 0600 |
| SessionState | `<UserConfigDir>/autowifi-bits/state.json` | 0600 |
| Log | `<UserConfigDir>/autowifi-bits/autowifi.log`, 1 MiB, rotated once; outcomes only | 0600 |
| Android | Keystore AES-256-GCM key `autowifi-creds` encrypts credentials in SharedPreferences; config + state in SharedPreferences | app-private |

## Security rules (enforced by tests)
1. Credentials go only to `https://{portal_host}:{portal_port}{login_path}` (host rule).
2. Portal TLS: OS trust store, else host-bound SPKI pin, else refuse. Probe client is separate.
3. Credentials never in logs, errors, notifications, config files or the repo.
4. No telemetry; the only third-party requests are the probe URLs and the daily GitHub
   release check.

## Toolchain
- Go 1.26 (local), cgo only for the macOS tray. Release: macOS host (universal binary via
  `lipo`), Linux and Windows cross-compiled with `CGO_ENABLED=0`.
- Kotlin 2.2.20, JDK 17 in Docker `gradle:8.14-jdk17` (`scripts/test-android.sh`); the APK is
  built with the Android SDK image in CI (and locally in T24 with
  `ghcr.io/cirruslabs/android-sdk`), AGP pinned in T24.

## Dependencies (each one has a reason)
Go:
- `github.com/zalando/go-keyring`: one API over macOS Keychain, Windows Credential Manager and
  Linux Secret Service; required by the spec. Pure Go (uses `/usr/bin/security` on macOS).
- `fyne.io/systray`: tray/menu-bar icon on all three OSes; pure Go on Windows/Linux.
- `golang.org/x/net/html`: tolerant HTML parsing of the login page (regex would break on real
  markup).
- `golang.org/x/term`: no-echo password prompt for `autowifi setup` in a terminal.
- `golang.org/x/sys/windows/registry`: HKCU Run key for start-at-login on Windows;
  `golang.org/x/sys/windows` also for AttachConsole and the single-instance mutex.
Kotlin (core):
- `org.jsoup:jsoup`: same reason as x/net/html.
- tests only: JUnit 5, `org.json` (reads the shared JSON cases).
Kotlin (app, T24): AndroidX WorkManager (spec), AndroidX core/appcompat for the settings
screen. Nothing else.

## Decisions
- No SSID logic anywhere except the iOS automation (spec non-goal).
- Network change on desktop = polling interface addresses: no per-OS APIs, cheap, works for
  Wi-Fi joins, cable changes and wake from sleep.
- The login POST never uses the form's `action` (host rule); `capture` shows it so a change
  can be made in config deliberately.
- Kotlin core is plain JVM so the shared cases run without an Android SDK.
- iOS Shortcuts are hand-built from the guide: unsigned `.shortcut` files cannot be imported on
  iOS 15+, and a signed one would have to be shared through iCloud links we do not control.
