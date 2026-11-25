# T32: Release build script and GitHub workflows

## Goal
Build every desktop release asset with one script and publish releases and the Pages site from GitHub Actions.

## Implement
- scripts/build-release.sh VERSION: `set -euo pipefail`; clean dist/; ldflags `-s -w -X main.version=$VERSION -X main.repo=$(cat RELEASE_REPO)`; `-trimpath`.
  - macOS: build darwin/amd64 and darwin/arm64 with CGO_ENABLED=1 (macOS host only; fail with a clear message elsewhere), `lipo -create` → `autowifi`, zip as dist/autowifi-macos.zip (file at zip root).
  - Windows amd64/arm64: CGO_ENABLED=0, extra ldflag `-H windowsgui`, `autowifi.exe` zipped as dist/autowifi-windows-<arch>.zip.
  - Linux amd64/arm64: CGO_ENABLED=0, `autowifi` in dist/autowifi-linux-<arch>.tar.gz.
  - If dist/autowifi-bits.apk exists (CI copies it in first), keep it. Write dist/SHA256SUMS.txt with `shasum -a 256` over every asset (run inside dist/ so names are bare).
- .github/workflows/release.yml: on push of tags `v*`: job `apk` (ubuntu-latest, setup-java 17, android-actions/setup-android, `bash scripts/build-apk.sh` → upload artifact autowifi-bits.apk; the script comes from T24, so the job may be marked `continue-on-error: false` and simply fail until it exists), job `desktop` (macos-latest, setup-go with go-version-file go.mod, download the apk artifact into dist/, `bash scripts/build-release.sh ${{ github.ref_name }}`, `bash scripts/check-dist.sh ${{ github.ref_name }}`, then `gh release create "$GITHUB_REF_NAME" dist/* --generate-notes` with `GH_TOKEN: ${{ github.token }}`). `permissions: contents: write`. Pin actions to major versions (actions/checkout@v4, actions/setup-go@v5, actions/setup-java@v4, android-actions/setup-android@v3, actions/upload-artifact@v4, actions/download-artifact@v4).
- .github/workflows/pages.yml: on push to main touching site/** and workflow_dispatch: actions/configure-pages@v5, actions/upload-pages-artifact@v3 with path site, actions/deploy-pages@v4; `permissions: pages: write, id-token: write, contents: read`.

## Imitate
scripts/test-android.sh for shell style.

## Acceptance
`bash scripts/build-release.sh v0.0.1-test && bash scripts/check-dist.sh v0.0.1-test` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
