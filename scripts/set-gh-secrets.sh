#!/bin/bash
# Copies the Android release signing key from the vault into GitHub Actions secrets.
# usage: bs-secret.sh run android-automaticwifi-bits -- bash scripts/set-gh-secrets.sh
# Values go to gh on stdin only (never argv, never printed). Needs a gh token that may write
# Actions secrets on the repo in RELEASE_REPO.
set -euo pipefail
cd "$(dirname "$0")/.."
repo=$(tr -d '[:space:]' < RELEASE_REPO)
for k in ANDROID_KEYSTORE_BASE64 ANDROID_KEYSTORE_PASSWORD ANDROID_KEY_ALIAS ANDROID_KEY_PASSWORD; do
  [ -n "${!k:-}" ] || { echo "set-gh-secrets: $k is not set (run through bs-secret.sh run)" >&2; exit 1; }
  printf '%s' "${!k}" | gh secret set "$k" -R "$repo"
done
gh secret list -R "$repo"
