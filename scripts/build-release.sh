#!/bin/bash
# Builds every desktop release asset into dist/ (names in internal/docscheck Assets).
# usage: scripts/build-release.sh <version>      e.g. v1.2.0
# Runs on macOS only (the macOS tray needs cgo; Linux and Windows are cross-compiled without it).
# If dist/autowifi-bits.apk already exists (CI copies it in first) it is kept and checksummed.
set -euo pipefail
cd "$(dirname "$0")/.."
v=${1:?usage: build-release.sh <version>}
[ "$(uname -s)" = Darwin ] || { echo "build-release: run on macOS (the macOS build needs cgo)" >&2; exit 1; }
repo=$(tr -d '[:space:]' < RELEASE_REPO)
ld="-s -w -X main.version=$v -X main.repo=$repo"

# keep a pre-built APK, clear everything else
apk=""
if [ -f dist/autowifi-bits.apk ]; then apk=$(mktemp); cp dist/autowifi-bits.apk "$apk"; fi
mkdir -p dist
find dist -mindepth 1 -delete
[ -n "$apk" ] && mv "$apk" dist/autowifi-bits.apk

work=$(mktemp -d)
trap 'find "$work" -mindepth 1 -delete; rmdir "$work"' EXIT

# macOS universal binary
for arch in amd64 arm64; do
  CGO_ENABLED=1 GOOS=darwin GOARCH=$arch go build -trimpath -ldflags "$ld" -o "$work/autowifi-$arch" ./cmd/autowifi
done
mkdir -p "$work/mac"
lipo -create -output "$work/mac/autowifi" "$work/autowifi-amd64" "$work/autowifi-arm64"
(cd "$work/mac" && zip -q -X ../autowifi-macos.zip autowifi)
mv "$work/autowifi-macos.zip" dist/

# Windows (GUI subsystem: no console window; CLI commands attach to the parent console)
for arch in amd64 arm64; do
  mkdir -p "$work/win-$arch"
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch go build -trimpath -ldflags "$ld -H windowsgui" -o "$work/win-$arch/autowifi.exe" ./cmd/autowifi
  (cd "$work/win-$arch" && zip -q -X "../autowifi-windows-$arch.zip" autowifi.exe)
  mv "$work/autowifi-windows-$arch.zip" dist/
done

# Linux
for arch in amd64 arm64; do
  mkdir -p "$work/linux-$arch"
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags "$ld" -o "$work/linux-$arch/autowifi" ./cmd/autowifi
  tar -C "$work/linux-$arch" -czf "dist/autowifi-linux-$arch.tar.gz" autowifi
done

(cd dist && shasum -a 256 -- * | grep -v ' SHA256SUMS.txt$' > SHA256SUMS.txt)
echo "build-release: $(ls dist | wc -l | tr -d ' ') files in dist/ for $v"
