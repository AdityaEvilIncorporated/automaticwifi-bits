# Prep report for automaticwifi-bits
Generated 2026-11-25 19:21 by prep-project.sh. Read this instead of the chat.

## Spec summary
Goal: Automatically and silently logs BITS Pilani students into the FortiGate campus Wi-Fi captive portal in the background when joining the network or when sessions expire.
Users: BITS Pilani students authenticating with their own campus ID and password

Features:
- [must] Captive portal detection and FortiGate login: Check http://neverssl.com (fallback http://connectivitycheck.gstatic.com/generate_204); if redirected or body contains https://fw.bits-pilani.ac.in:8090/fgtauth?<magic>, POST credentials to https://fw.bits-pilani.ac.in:8090/ using configurable form fields (defaults: username, password, magic, 4Tredir).
- [must] Already-logged-in detection and session keepalive: Do nothing when the check URL loads normally without fgtauth; after login, poll the returned keepalive URL (https://fw.bits-pilani.ac.in:8090/keepalive?<token>) every 5 minutes and trigger a fresh login if keepalive fails.
- [must] Campus network detection and host-pinned TLS: Treat DNS resolution failure for fw.bits-pilani.ac.in as off-campus and do nothing; verify HTTPS on port 8090 against the standard trust store first and pin fw.bits-pilani.ac.in only if an OS store fails, never disabling TLS verification globally.
- [must] Retry backoff and quiet failure notifications: Retry failed checks or logins at 5 s, 15 s, 60 s, then every 5 minutes; send at most one quiet notification per hour on rejected credentials (pausing retries until updated), 3 consecutive non-credential failures, or unexpected portal form changes, and never notify on success.
- [must] OS secure credential storage: Store credentials only in macOS Keychain, Windows Credential Manager, Android Keystore, or Linux Secret Service API via go-keyring (falling back to ~/.config/autowifi-bits/credentials with mode 600 and a one-time warning if Secret Service is unavailable), never in code or logs.
- [must] macOS background agent: Cross-compiled Go single binary for macOS 12+ running silently at login with an optional menu-bar icon and no windows.
- [must] Windows background tray app: Cross-compiled Go single binary for Windows 10+ running at login with a system tray icon.
- [must] Android background app: Kotlin app for Android 8.0 (API 26)+ triggered on Wi-Fi connect via ConnectivityManager.NetworkCallback plus a 15-minute WorkManager periodic check, requesting battery-optimization exemption once and distributed as an APK via GitHub Releases.
- [must] iOS Apple Shortcuts automation and guide: Apple Shortcut triggered when joining specified campus Wi-Fi SSIDs, storing credentials locally in Shortcut Text actions with instructions noting iCloud sync and recommending iCloud Advanced Data Protection.
- [must] Shared protocol spec, JSON test suite, and local mock portal: One written protocol specification and shared JSON test cases run by both Go and Kotlin implementations against a local mock FortiGate portal that issues per-visit magic tokens and exposes keepalive and logout endpoints.
- [must] Campus portal capture diagnostic command: Diagnostic command that inspects an active captive portal page and prints the form action URL and input field names (never values) to confirm campus portal configuration.
- [must] GitHub Releases and static download page: GitHub Pages static site linking to GitHub Releases binaries/APK and documenting unsigned-app bypass steps (macOS Gatekeeper, Windows SmartScreen, Android APK sideloading) and a pre-release campus verification checklist.
- [should] Linux desktop support: Cross-compiled Go single binary for Ubuntu 22.04+ with tray icon, start-at-login support, and Secret Service or mode-600 credential storage.
- [should] Manual log-in-now trigger: Tray menu or app button allowing the student to trigger an immediate captive portal check and login.
- [should] Auto-update check: Periodic check against GitHub Releases to inform the user when a newer version is available.
- [later] Code signing and notarization: Sign and notarize desktop binaries for macOS Gatekeeper and Windows SmartScreen in a future release.

Flows:
- First-time setup: Student downloads the binary, APK, or Apple Shortcut from GitHub Pages/Releases, completes OS unsigned-app prompts if needed, saves campus ID and password into the OS secure store (or iOS Shortcut Text actions), and enables login startup or grants Android battery-optimization exemption.
- Silent background login: On network change, Wi-Fi join, or periodic timer, the client requests the check URL; if redirected to fgtauth, it extracts the magic token, submits credentials over HTTPS to fw.bits-pilani.ac.in:8090, and polls the returned keepalive URL every 5 minutes.
- Off-campus or active session: If fw.bits-pilani.ac.in fails DNS resolution or the check URL returns normally without fgtauth, the client exits the check quietly with no notification.
- Error handling and backoff: On network/portal errors the client retries at 5 s, 15 s, 60 s, then every 5 minutes; on wrong credentials or changed form it emits at most one quiet notification per hour and pauses retries on rejected credentials until updated.
- On-campus capture verification: Student runs the capture diagnostic command while the captive portal is displayed to inspect the form action URL and field names before finalizing defaults.

Data:
- Credentials: username, password
- PortalConfig: check_url, fallback_check_url, portal_host, portal_port, username_field, password_field, magic_field, redirect_field, pinned_cert_sha256
- SessionState: magic_token, keepalive_url, logout_url, consecutive_failures, last_notification_at, credentials_rejected

Integrations: BITS Pilani FortiGate captive portal (https://fw.bits-pilani.ac.in:8090 fgtauth, login POST, keepalive, logout), Captive portal probe endpoints (http://neverssl.com, http://connectivitycheck.gstatic.com/generate_204), OS secure credential stores (macOS Keychain, Windows Credential Manager, Linux Secret Service API, Android Keystore), Android ConnectivityManager.NetworkCallback and WorkManager, Apple Shortcuts Wi-Fi join automation, GitHub Releases and GitHub Pages
Non-goals: Backend servers, central databases, user accounts, or hosted web applications; Rust + UniFFI shared native library; Disabling TLS certificate verification globally; Storing user credentials in source code, logs, Git repository, or the static download page; Native background iOS app; SSID-based triggering on macOS, Windows, Linux, or Android; Code signing or notarization for the initial release

## Stack (latest versions checked now)
Language: Go (desktop: macOS, Windows, Linux) and Kotlin (Android)  Framework: Cross-compiled Go single binary with tray/autostart wrappers; Android SDK (API 26+) with WorkManager; Apple Shortcuts  Database: None (OS secure credential stores and local mode-600 Linux fallback file only)
- go-keyring: not found in npm or PyPI, check the name
- AndroidX: not found in npm or PyPI, check the name
- WorkManager: not found in npm or PyPI, check the name
