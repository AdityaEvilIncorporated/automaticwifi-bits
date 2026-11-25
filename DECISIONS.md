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
- 2026-11-25 Network change on desktop = 15 s interface-address poll (no per-OS APIs, no SSIDs).
- 2026-11-25 Kotlin core is plain JVM, tested in Docker gradle:8.14-jdk17 against the Go mock binary.
- 2026-11-25 Release assets have unversioned names so the page can link releases/latest/download/<name>.
- 2026-11-25 Release repo guess: RELEASE_REPO = AdityaEvilIncorporated/automaticwifi-bits (from the git
  user name); confirmed or changed at deploy time, one file to edit.
- 2026-11-25 Review: a probe redirect whose fgtauth query is not purely [0-9A-Za-z] is `malformed`
  (FortiGate magics are hex; anything else means the portal changed). Added to PROTOCOL + probe.json.
- 2026-11-25 Windows notifications are shown under PowerShell's AppUserModelID instead of registering
  our own (registration needs a Start-menu shortcut; not worth it for quiet error notices).
