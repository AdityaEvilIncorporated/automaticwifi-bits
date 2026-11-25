#!/bin/bash
# Checks the output of scripts/build-release.sh (acceptance test for T32).
# usage: scripts/check-dist.sh <version>   (desktop assets only; the APK is built in CI)
set -euo pipefail
cd "$(dirname "$0")/.."
v=${1:?version}
fail() { echo "check-dist: $*" >&2; exit 1; }
for f in autowifi-macos.zip autowifi-windows-amd64.zip autowifi-windows-arm64.zip \
         autowifi-linux-amd64.tar.gz autowifi-linux-arm64.tar.gz SHA256SUMS.txt; do
  [ -s "dist/$f" ] || fail "missing dist/$f"
done
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
unzip -q dist/autowifi-macos.zip -d "$tmp/mac"
[ -x "$tmp/mac/autowifi" ] || fail "macos zip must contain an executable 'autowifi'"
lipo -archs "$tmp/mac/autowifi" | grep -q x86_64 || fail "macos binary not universal (x86_64)"
lipo -archs "$tmp/mac/autowifi" | grep -q arm64 || fail "macos binary not universal (arm64)"
"$tmp/mac/autowifi" version | grep -qx "$v" || fail "'autowifi version' must print exactly $v"
unzip -q dist/autowifi-windows-amd64.zip -d "$tmp/win"
file "$tmp/win/autowifi.exe" | grep -q 'PE32+.*x86-64' || fail "windows amd64 exe"
file "$tmp/win/autowifi.exe" | grep -q 'GUI' || fail "windows exe must be built with -H windowsgui"
tar -xzf dist/autowifi-linux-amd64.tar.gz -C "$tmp" autowifi
file "$tmp/autowifi" | grep -q 'ELF 64-bit.*x86-64' || fail "linux amd64 binary"
(cd dist && shasum -a 256 -c SHA256SUMS.txt >/dev/null) || fail "SHA256SUMS.txt does not verify"
for f in dist/*.zip dist/*.tar.gz; do
  if unzip -l "$f" 2>/dev/null | grep -qE '\.env|credentials' || tar -tzf "$f" 2>/dev/null | grep -qE '\.env|credentials'; then fail "$f contains secrets-like files"; fi
done
echo "check-dist OK"
