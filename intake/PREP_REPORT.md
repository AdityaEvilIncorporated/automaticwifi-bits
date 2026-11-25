# Prep report for automaticwifi-bits
Generated 2026-11-25 19:20 by prep-project.sh. Read this instead of the chat.

## Spec summary
Goal: Automatically and silently logs BITS Pilani students into the campus FortiGate Wi-Fi captive portal in the background when joining campus Wi-Fi or when sessions expire.
Users: BITS Pilani students using their own campus ID and password

Features:
- [must] Captive portal detection and FortiGate login: Check http://neverssl.com (fallback http://connectivitycheck.gstatic.com/generate_204); if redirected or served a 200 page containing https://fw.bits-pilani.ac.in:8090/fgtauth?<magic>, POST configurable form fields (default username, password, magic, 4Tredir). Do nothing if already logged in or if fw.bits-pilani.ac.in does not resolve.
- [must] Keepalive and session refresh: Extract the keepalive URL (https://fw.bits-pilani.ac.in:8090/keepalive?<token>) after successful login and call it every 5 minutes; trigger a fresh login if keepalive fails.
- [must] Retry backoff and quiet failure notifications: Retry failed checks or logins at 5s, 15s, 60s, then every 5 minutes while the portal is showing. Send at most one quiet notification per hour (never on success) when credentials are rejected (stop retrying until updated), after 3 consecutive non-credential login failures, or if the portal form changes.
- [must] Secure OS credential storage: Store campus ID and password only in macOS Keychain, Windows Credential Manager, Android Keystore, or Linux Secret Service (via go-keyring, with ~/.config/autowifi-bits/credentials mode 600 fallback and one-time warning). Never store credentials in code, logs, repo, or download page.
- [must] macOS background agent: Single Go binary for macOS 12+ running in the background with no windows, optional menu-bar icon, and start-at-login support.
- [must] Windows background app: Single Go binary for Windows 10+ with system tray icon and start-at-login support.
- [must] Android background app: Kotlin app for Android 8.0+ (API 26+) triggering checks on Wi-Fi connect via ConnectivityManager.NetworkCallback plus a 15-minute WorkManager periodic check, with a one-time battery-optimization exemption request.
- [must] iOS Apple Shortcuts automation and guide: Apple Shortcuts automation triggered when joining BITS Wi-Fi SSIDs with on-device Text actions for credentials and setup instructions advising iCloud Advanced Data Protection.
- [must] TLS verification and host-only pinning: Verify the *.bits-pilani.ac.in certificate on port 8090 against the standard OS trust store first, falling back to pinning fw.bits-pilani.ac.in only if verification fails on an OS; never disable TLS verification globally.
- [must] Shared protocol spec, JSON test cases, and local mock portal: One written protocol spec and shared JSON test cases executed by both Go and Kotlin implementations against a local mock FortiGate portal issuing per-visit magic tokens, keepalive, and logout URLs.
- [must] Capture diagnostic command: Diagnostic command that prints the captive portal form action URL and input field names (never values) when the portal is showing, with configurable field names.
- [must] GitHub Releases and static download page: Distribute binaries and APK via GitHub Releases and a static GitHub Pages download site documenting Gatekeeper, SmartScreen, and Android sideload steps.
- [should] Linux desktop support: Go single binary for Ubuntu 22.04+ with tray icon, start at login, and Secret Service or mode-600 file credential storage.
- [should] Manual login trigger: Provide a Log in now button or menu action to trigger an immediate portal check and login.
- [should] Auto-update check: Periodically check GitHub Releases for newer versions.
- [later] Code signing and notarization: Sign Windows binaries and sign/notarize macOS binaries in a future release.

Flows:
- Onboarding: Student downloads the installer/APK or Apple Shortcut from the static GitHub Pages site, follows OS trust prompts (Gatekeeper/SmartScreen/Android install permission), and saves their campus ID and password into the OS secure store (or Shortcut Text actions on iOS).
- Automatic background login: Device connects to campus Wi-Fi or an existing session expires; the app checks neverssl.com (or gstatic generate_204), detects redirect to fw.bits-pilani.ac.in:8090/fgtauth?<magic>, submits the login POST, and stores the returned keepalive URL silently without notifying.
- Already logged in or off-campus: Check returns normal 200/204 without fgtauth, or fw.bits-pilani.ac.in fails DNS resolution; the app does nothing.
- Session keepalive: Every 5 minutes after login, the app requests the keepalive URL; if keepalive fails, it triggers a fresh captive portal check and login.
- Failure and backoff: On wrong credentials, the app sends one quiet notification and pauses retries until the user updates credentials; on network or portal errors, it retries at 5s, 15s, 60s, and 5m intervals, notifying quietly once per hour after 3 consecutive failures or unexpected portal form changes.
- Campus diagnostic capture: Student runs the capture command while the portal is visible to print the form action URL and field names for confirmation.

Data:
- CampusCredentials: username, password
- PortalConfig: check_urls, portal_host, login_url, username_field, password_field, magic_field, redirect_field, keepalive_interval_seconds, pinned_cert_fingerprint, known_ssids
- SessionState: magic_token, keepalive_url, logout_url, consecutive_failures, backoff_step, last_notification_at, credentials_invalid

Integrations: BITS Pilani FortiGate Captive Portal (https://fw.bits-pilani.ac.in:8090), Connectivity check endpoints (http://neverssl.com, http://connectivitycheck.gstatic.com/generate_204), macOS Keychain, Windows Credential Manager, Linux Secret Service API, Android Keystore, Android ConnectivityManager.NetworkCallback and WorkManager, Apple Shortcuts (iOS Wi-Fi automation), GitHub Releases and GitHub Pages
Non-goals: Any backend server, database, user accounts, or hosted web app; Disabling TLS certificate verification globally; Storing credentials in source code, logs, repository, or download page; Building a background login app for iOS; Depending on Wi-Fi SSID detection on desktop or Android; Using Rust with UniFFI; Code signing and notarization in the first release

## Stack (latest versions checked now)
Language: Go (macOS 12+, Windows 10+, Ubuntu 22.04+), Kotlin (Android 8.0+ / API 26+), Apple Shortcuts (iOS)  Framework: Go single cross-compiled desktop binary with system tray and auto-start; Native Android SDK  Database: None (OS secure credential stores only; mode-600 file fallback on Linux)
- go-keyring: not found in npm or PyPI, check the name
- WorkManager: not found in npm or PyPI, check the name
- ConnectivityManager.NetworkCallback: not found in npm or PyPI, check the name

## Toolchain on this Mac
- node: v26.13.1
- pnpm: 12.18.0
- python3: Python 3.13.7
- uv: uv 0.13.7 (Homebrew 2026-11-03 aarch64-apple-darwin)
- docker: Docker version 29.2.1, build a5c7197
- gitleaks: 8.30.1
- semgrep: 1.185.0

## Local services, databases and accounts (all automatic, no human step)
- .env ready (0 keys added), services: db,mail,storage
- services up: db mail storage 
- migrate: no migrations yet
- seed: no seed script yet
- db: running
- mail: running
- storage: running
- database: app
- database: app_test
- database: postgres
- Connection details and generated passwords: .env (local only, never committed)
- What each service stands in for, and what production will need: intake/ACCOUNTS.md

## Open questions still unanswered
1. What are the exact BITS Pilani Wi-Fi SSID names from iPhone Wi-Fi settings needed for the Apple Shortcuts automation trigger?
2. What are the exact FortiGate portal form action URL, input field names, and keepalive/logout URL formats captured on campus Wi-Fi?
3. Does the fw.bits-pilani.ac.in:8090 TLS certificate validate against standard OS trust stores on macOS, Windows, Ubuntu, and Android, or is a pinned host certificate needed?
4. What exact 4Tredir value does the campus portal expect, and what exact HTTP status or response body text distinguishes rejected credentials from transient portal errors?
