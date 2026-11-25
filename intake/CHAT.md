
Build "AutoWiFi BITS": logs a student into the BITS Pilani campus Wi-Fi captive portal automatically, silently, in the background, whenever the session expires (every few hours) or the device joins the campus Wi-Fi.
Users: BITS students; each uses their own campus ID and password.
Must have: (1) macOS background agent (menu-bar icon optional, no windows) that detects the captive portal and submits the login form; (2) Windows background app (tray icon, starts at login) doing the same; (3) Android app with background check; (4) iPhone: an Apple Shortcuts automation ("when joining the BITS Wi-Fi → log in") plus setup instructions, since iOS forbids background login apps; (5) credentials stored only in the OS secure store (Keychain / Credential Manager / Android Keystore), never in code or logs; (6) detect "already logged in" and do nothing; retry with backoff; quiet notification only on failure (wrong password, portal changed).
Core logic once, in one shared library, tested against a local mock of the portal; platform apps are thin wrappers.
Nice to have: Linux; a "log in now" button; auto-update check.
Not needed: any server, database, accounts or web hosting. Distribution = GitHub Releases + a simple static download page (GitHub Pages). Do not deploy a web app.
Stack: you choose, simple and popular (e.g. Python core + small native wrappers, or Go/Rust single binary for desktop + Kotlin for Android).
Open question for me: the exact portal URL and form fields (I can capture them on campus Wi-Fi).
