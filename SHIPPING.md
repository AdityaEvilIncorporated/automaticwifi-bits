# SHIPPING.md — manual release of AutoWiFi BITS

## Architecture in 5 lines
1. Desktop: one Go binary `autowifi` (macOS universal, Windows/Linux amd64+arm64), tray + background agent.
2. Android: Kotlin app (`android/core` shared logic, `android/app` UI + WorkManager), sideloaded APK.
3. iOS: a hand-built Shortcut (`ios/SHORTCUT_GUIDE.md`); no app.
4. Everything talks only to the campus FortiGate portal; no servers, no database, no accounts.
5. Distribution: GitHub Release (built by `.github/workflows/release.yml`) + GitHub Pages (`site/`).

## Accounts needed
| Account | Sign-up | Free tier | Card | Token for scripts |
|---|---|---|---|---|
| GitHub (repo owner) | https://github.com/signup | Public repo: Actions + Pages free | No | `gh` token with **Secrets: read/write** on the repo (only for `scripts/set-gh-secrets.sh`) |

No other provider. There is no `scripts/provision.sh`: nothing is provisioned.

## Release steps
1. [CHECK] On dev: `bash scripts/verify.sh` prints `VERIFY OK`.
2. [HUMAN] One time: Settings → Pages → Build and deployment → Source: **GitHub Actions**.
3. [HUMAN] One time: Android signing secrets. The key is in `~/.agent-secrets/providers/android-automaticwifi-bits.env`:
   `bs-secret.sh run android-automaticwifi-bits -- bash scripts/set-gh-secrets.sh`
   (sets ANDROID_KEYSTORE_BASE64, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD).
   Back up that vault file offline: losing the key means users must uninstall to update.
4. [HUMAN] Set the version: write it into `RELEASE_TAG` (e.g. `v0.1.1`), change the links and the banner
   in `site/index.html` to match, run `go test ./internal/docscheck/`, commit, `git push origin dev`.
   While `RELEASE_BETA` exists the release is a pre-release with that label in its title;
   `git rm RELEASE_BETA` (and remove the banner) once the campus checklist passes.
5. [HUMAN] `git tag v0.1.1 && git push origin v0.1.1` (the tag must be on dev and equal RELEASE_TAG).
6. [CHECK] Actions → "release" run is green; the Release has 8 assets (7 files + source).
7. [CHECK] Actions → "pages" run is green after the dev push.

Manual fallback with no CI (macOS only): `bash scripts/build-apk.sh` (needs ANDROID_KEYSTORE_FILE etc.),
then `bash scripts/build-release.sh v0.1.1 && bash scripts/check-dist.sh v0.1.1`, then upload `dist/*`
with `gh release create v0.1.1 dist/* --prerelease --title "v0.1.1 (beta: …)"`.

## Environment variables (build time only; no runtime env)
| Name | Purpose | Where from |
|---|---|---|
| ANDROID_KEYSTORE_FILE | path to the release .jks (local builds) | decode ANDROID_KEYSTORE_BASE64 from the vault into a temp dir |
| ANDROID_KEYSTORE_BASE64 | the .jks, base64 (CI secret) | vault |
| ANDROID_KEYSTORE_PASSWORD / ANDROID_KEY_PASSWORD | keystore / key password | vault |
| ANDROID_KEY_ALIAS | key alias | vault |
| APP_VERSION_NAME / APP_VERSION_CODE | APK version | derived from the tag by release.yml |

## Database, domain, payments, email
None. Pages URL: https://adityaevilincorporated.github.io/automaticwifi-bits/ (no custom domain).

## Post-release smoke test
1. Open the Pages URL: the beta banner shows and every download link returns the file (not 404).
2. `shasum -a 256 -c SHA256SUMS.txt` in a folder with the downloads: all OK.
3. macOS: unzip, `./autowifi version` prints the tag. Android: install the APK, the app opens.
4. On campus: the checklist on the page (owner: the human).

## Rollback
- Bad release: edit it to a draft or delete it on GitHub (Releases → … → Delete), delete the tag
  (`git push origin :refs/tags/vX.Y.Z`), point RELEASE_TAG and the page back at the previous tag, push dev.
- Bad page: revert the site commit on dev and push; Pages redeploys.

## Monitoring and alerts
None by design (no telemetry). Watch GitHub Issues; Actions emails the owner when a workflow fails.

## Cost
0. Public repo: Actions minutes and Pages are free. No billing to configure.

## Pre-launch security checklist
- [x] No secrets in the repo (gitleaks clean); signing key only in the vault and Actions secrets.
- [x] Actions pinned to commit SHAs; the signing step runs only for well-formed tags on dev.
- [x] govulncheck: no known vulnerabilities (toolchain go1.26.11).
- [x] Security review: no critical/high findings (DECISIONS.md, 2026-11-25).
- [ ] Campus verification checklist (on the page) before removing the beta label.
