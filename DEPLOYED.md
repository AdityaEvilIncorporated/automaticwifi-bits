# DEPLOYED — AutoWiFi BITS v0.1.0 (pre-release)

Date: 2026-11-26. Tag `v0.1.0` → commit e513e31 on dev.

## URLs
- Release (pre-release, "v0.1.0 (beta: not yet verified on the real BITS portal)"):
  https://github.com/AdityaEvilIncorporated/automaticwifi-bits/releases/tag/v0.1.0
- Download page (GitHub Pages, deployed from dev):
  https://adityaevilincorporated.github.io/automaticwifi-bits/
  **Known problem:** the account's user Pages site has the custom domain `adityaevilincorporated.me`,
  so GitHub redirects this project page to https://www.adityaevilincorporated.me/automaticwifi-bits/.
  That domain now points at a Render app, which returns 404. The page deploy itself succeeded.
  Fix options are for the account owner (see "Open items"). Until then, the release notes carry the
  campus checklist and all download links.

## Providers and plans
GitHub only: public repo, Actions and Pages on the free plan. No card, no other accounts.

## What was created
- Android release signing key (RSA 4096, alias autowifi-bits). Cert SHA-256
  a4:3b:43:c0:ac:44:a6:fa:82:f0:a4:93:a8:9f:2c:b3:33:8a:4c:7c:a8:06:1c:c8:61:f9:fa:96:e2:8a:f8:23.
  Stored only in ~/.agent-secrets (provider android-automaticwifi-bits) and as 4 GitHub Actions
  secrets (ANDROID_KEYSTORE_BASE64, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD).
  **Back up the vault file offline**: without it, updates need an uninstall.
- GitHub Release v0.1.0 with 7 assets (macOS universal zip, Windows amd64/arm64 zips, Linux
  amd64/arm64 tar.gz, signed APK, SHA256SUMS.txt), built by .github/workflows/release.yml (run 45681396940).
- GitHub Pages deployment of site/ (workflow pages, run 45680827045).

## Smoke test (2026-11-26)
- All 7 download links on the page return 200; `shasum -a 256 -c SHA256SUMS.txt` OK for every file.
- macOS zip: `autowifi version` → v0.1.0, universal (x86_64 arm64).
- APK: `apksigner verify --print-certs` OK, signer is the release key above.
- Not done: installing on Android/Windows/Linux, and anything on the real campus portal (campus checklist).

## Rollback
- Unpublish: GitHub → Releases → v0.1.0 → Edit → "Save as draft" (or delete), then
  `git push origin :refs/tags/v0.1.0`.
- Page: revert the site commit on dev and push (pages workflow redeploys).

## Limits
- Pre-release builds are not shown by /releases/latest; the page links the tag in RELEASE_TAG.
- Desktop binaries are unsigned: Gatekeeper/SmartScreen warnings (instructions on the page).

## Open items
1. Pages custom-domain redirect (owner decision): either remove the custom domain from the
   user site (Settings → Pages of the `AdityaEvilIncorporated.github.io` repo), or serve
   `/automaticwifi-bits/` from that domain, or give this project its own (sub)domain.
2. Campus verification checklist (CAMPUS_CHECKLIST.md); then delete RELEASE_BETA and the banner and release v0.1.1 or v0.2.0.
3. No PR to main: main has never been pushed to GitHub (dev is the default branch), and the agent
   never pushes main. Push main yourself if you want the PR record.
