## Campus verification checklist

Run on campus before relying on a beta build:

- [ ] `autowifi capture` while the portal shows: portal fields match config defaults, action is `/`.
- [ ] Certificate: "TLS verified by OS trust store: yes" on macOS, Windows and Linux; Android logs the same.
- [ ] Wrong password → one notification, no retry storm; right password → silent login.
- [ ] Session survives > 1 hour with keepalive.
- [ ] SSIDs: campus SSID names written into `ios/SHORTCUT_GUIDE.md`; Shortcut tested on an iPhone.
- [ ] If any field differs: update config defaults (both languages + shared cases) and re-release.
