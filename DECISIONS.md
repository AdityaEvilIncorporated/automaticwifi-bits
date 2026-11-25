# Decisions

- 2026-11-25 Stack: Go desktop binary + Kotlin Android core/app + hand-built iOS Shortcut, as in
  intake/ANSWERS.md. No Rust/UniFFI.
- 2026-11-25 No database or server: the CLAUDE.md DB/seed machinery is unused; scripts/seed.sh is a
  no-op. Settings that would be env vars in a web app are config.json keys (PROTOCOL.md section 1),
  so a portal change needs no code change. .env.example keys are template leftovers and unused.
- 2026-11-25 Host rule: the login POST always goes to {portal base}{login_path}, never to the form
  action or any other host (protects credentials from look-alike portals).
- 2026-11-25 TLS: OS trust store first; optional SPKI SHA-256 pin bound to portal_host. The capture
  command reads the certificate with a handshake-only connection (no HTTP) to print the pin.
- 2026-11-25 Network change on desktop = interface-address poll every 2 s with a 3 s debounce (no
  per-OS APIs, no SSIDs). Was 15 s; changed in T14: the debounce only works if polling is faster.
- 2026-11-25 Kotlin core is plain JVM, tested in Docker gradle:8.14-jdk17 against the Go mock binary.
- 2026-11-25 Release assets have unversioned names so the page can link releases/latest/download/<name>.
- 2026-11-25 Release repo guess: RELEASE_REPO = AdityaEvilIncorporated/automaticwifi-bits (from the git
  user name); confirmed or changed at deploy time, one file to edit.
- 2026-11-25 Review: a probe redirect whose fgtauth query is not purely [0-9A-Za-z] is `malformed`
  (FortiGate magics are hex; anything else means the portal changed). Added to PROTOCOL + probe.json.
- 2026-11-25 Windows notifications are shown under PowerShell's AppUserModelID instead of registering
  our own (registration needs a Start-menu shortcut; not worth it for quiet error notices).
- 2026-11-25 T07 tlsconf: InsecureSkipVerify + VerifyConnection (fail-closed, runs on resumed sessions)
  is the only way crypto/tls allows "trust store, else host-bound pin". Host binding: SNI must equal
  portal_host; an empty SNI is accepted only when portal_host is an IP literal (Go sends no SNI for
  IPs; only the mock uses an IP). Pin compared in constant time. Semgrep rule silenced on 2 lines.
- 2026-11-25 T07 second opinion NOT run: second-opinion.sh only diffs dev...task/<ID> and T07 was
  committed on dev (17d553a); the scratch-clone workaround was denied. Run it on a task branch next
  time; `second-opinion.sh release` before deploy covers this code.
- 2026-11-25 L1 "concerns" on T05 (needle case) and T09 (notify on error) were false positives:
  PortalBase() lowercases the host; the contract requires counting a failed Notify as sent.
- 2026-11-25 T14 desktop agent: one agent per user (flock / Local\ named mutex); other autowifi
  processes talk to it with kick files (kick-creds, kick-manual, kick-logout) in the config dir,
  polled every 2 s, so two processes never write state.json at once. First run without
  credentials asks once (terminal or native dialog) and turns on start at login.
- 2026-11-25 T14 second opinion (Gemini) ran twice on task/T14 (CLI/tray/prompt; the agent loop in
  3e26b53 was already on dev). Round 1 "block": fixed all 11 (incl. -NonInteractive breaking
  Get-Credential, headless Linux tray panic). Round 2 "concerns": fixed 6; accepted 2 low ones:
  `status` briefly takes the agent lock to test it (harmless), and checkUpdates has no test
  (hard-coded GitHub URL; covered by update package tests).
- 2026-11-25 Credential dialogs: macOS osascript with prompt text as argv; Windows PowerShell
  Get-Credential (UTF-8 over a stdout pipe only); Linux zenity/kdialog only with a display.
  Untested on real Windows/Linux desktops: on the release checklist.
- 2026-11-25 T24 Android: plain Activity + programmatic views (no AppCompat); WorkManager only
  dependency. Wi-Fi join = registerNetworkCallback(PendingIntent) → receiver (goAsync) → unique work
  per trigger; 15-min periodic work; delayed keepalive/retry work. Every request uses the Wi-Fi
  Network object (not a VPN, must have INTERNET). No-Wi-Fi checks still run through the Engine.
  Config = standard defaults; only the pin is editable (Advanced), since only the campus TLS check
  may need it on Android. Credentials: AES-256-GCM, non-exportable AndroidKeyStore key,
  randomized encryption; all backup and device-transfer domains excluded.
- 2026-11-25 APK build: Docker ghcr.io/cirruslabs/android-sdk:35 as linux/amd64 (its aapt2 is x86-64
  only); natively when ANDROID_HOME is set (CI). Release signing only from env vars; the keystore
  is created at deploy time and kept in bs-secret + GitHub Actions secrets, never in the repo.
  Gradle wrapper 8.14.3 pinned by distributionSha256Sum.
- 2026-11-25 T24 second opinion: round 1 "block" (12, all fixed), round 2 "concerns" (7, all fixed).
- 2026-11-25 "Log out" ends the current session only; automatic login resumes at the next scheduled
  check (desktop timer / Android periodic work), same on both platforms. A "pause" feature would be
  a new requirement.
- 2026-11-25 GitHub Actions are pinned to full commit SHAs (tag in a comment): the release job
  holds the Android signing key, so a repointed tag would be a supply-chain hole. Update pins
  deliberately (git ls-remote the tag, review the diff).
- 2026-11-25 Process slip: T32 (64a08cf) was committed and pushed while verify.sh was failing
  (semgrep, unpinned actions) because the commit was chained with ';' after the check. Fixed in the
  next commit; from now on the commit is chained with '&&' after verify.
- 2026-11-25 v0.1.0 ships as a GitHub pre-release labelled "beta: not yet verified on the real BITS
  portal" (RELEASE_BETA). The page links /releases/download/<RELEASE_TAG>/ because
  /releases/latest/ ignores pre-releases. release.yml refuses a tag that is malformed, not equal to
  RELEASE_TAG, or not on dev. Delete RELEASE_BETA after the campus checklist passes.
- 2026-11-25 Pages deploys on dev pushes: dev is the GitHub default branch; main is never pushed by
  the agent (the human's record).
- 2026-11-25 Security review (whole repo): 0 critical/high. Fixed: tag validation/injection and
  reviewed-commit check before the signing key (M1, L2); atomic writes through O_EXCL temp files
  (L4); capture output stripped of control characters (L6); update link must be a github.com
  …/releases/ page (L7); Android pin+hostname difference documented in PROTOCOL.md (L8); tracked
  .bak files removed (L9). Accepted: no release environment with required reviewers (single-owner
  repo; the dev check covers it), no build-provenance attestation yet (L3), kick files in a shared
  --config dir (L5: the default dir is per-user 0700; the temp-file fix closes the data leak).
- 2026-11-25 govulncheck found 28 reachable stdlib vulns in go1.26.0: go.mod pins
  `toolchain go1.26.11`; govulncheck now reports none.
- 2026-11-25 Android release key: RSA 4096, PKCS12, alias autowifi-bits, 10000 days, made with
  keytool in Docker; stored only in ~/.agent-secrets (provider android-automaticwifi-bits) and, once
  allowed, GitHub Actions secrets. Cert SHA-256 A4:3B:43:C0:AC:44:A6:FA:82:F0:A4:93:A8:9F:2C:B3:33:
  8A:4C:7C:A8:06:1C:C8:61:F9:FA:96:E2:8A:F8:23. Losing it means users must uninstall to update.
- 2026-11-25 Release second opinion round 1 "block", 8 issues, all fixed: Windows dialog now uses
  $Host.UI.PromptForCredential (Get-Credential -Message needs -UserName on PowerShell 5.1; a failed
  dialog is now an error, not "cancelled"); iOS guide link → blob/dev (docscheck forbids
  blob/main); macOS autostart never boots out (it killed the launchd-started tray) and bootstraps
  only when not loaded; Kotlin skips token-less keepalive/logout mentions like Go (new shared
  case); versionCode uses base-10 and refuses 0 / minor,patch ≥ 100; Log out re-arms the agent
  timer (a stale keepalive timer logged straight back in); Ctrl-C handling decided from the parsed
  subcommand (--config before it no longer traps SIGINT); guide step number. The PowerShell script
  could not be parse-checked (pwsh in Docker hung); real Windows stays on the manual checklist.
