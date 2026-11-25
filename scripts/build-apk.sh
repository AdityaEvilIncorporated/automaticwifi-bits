#!/bin/bash
# Builds dist/autowifi-bits.apk (and runs the app unit tests + lint).
# - With ANDROID_HOME set (CI: android-actions/setup-android) it runs Gradle directly.
# - Otherwise it uses Docker image ghcr.io/cirruslabs/android-sdk:35 as linux/amd64 (the SDK's
#   aapt2 is x86-64 only; Docker Desktop runs it under Rosetta on Apple silicon).
# Signing: if ANDROID_KEYSTORE_FILE (+ _PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD) are set,
# a signed release APK is built; otherwise a debug-signed APK (fine for testing, but updates
# between differently-signed builds need an uninstall: release builds must use the CI key).
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
variant=Debug
ks_mount=()
if [ -n "${ANDROID_KEYSTORE_FILE:-}" ]; then
  variant=Release
  [ -f "$ANDROID_KEYSTORE_FILE" ] || { echo "build-apk: ANDROID_KEYSTORE_FILE not found" >&2; exit 1; }
  ANDROID_KEYSTORE_FILE=$(cd "$(dirname "$ANDROID_KEYSTORE_FILE")" && pwd)/$(basename "$ANDROID_KEYSTORE_FILE")
  export ANDROID_KEYSTORE_FILE
fi
tasks=":app:test${variant}UnitTest :app:lint${variant} :app:assemble${variant}"
if [ -n "${ANDROID_HOME:-}" ]; then
  (cd android && ./gradlew --no-daemon -q $tasks)
else
  ks_env=()
  if [ -n "${ANDROID_KEYSTORE_FILE:-}" ]; then
    # the keystore lives outside the repo: mount it read-only at a fixed path
    ks_mount=(-v "$ANDROID_KEYSTORE_FILE:/keystore/release.jks:ro")
    ks_env=(-e ANDROID_KEYSTORE_FILE=/keystore/release.jks -e ANDROID_KEYSTORE_PASSWORD -e ANDROID_KEY_ALIAS -e ANDROID_KEY_PASSWORD)
  fi
  docker run --rm --platform linux/amd64 -v "$PWD":/work -w /work/android \
    -v autowifi-gradle-amd64:/root/.gradle ${ks_mount[@]+"${ks_mount[@]}"} \
    -e APP_VERSION_NAME -e APP_VERSION_CODE ${ks_env[@]+"${ks_env[@]}"} \
    ghcr.io/cirruslabs/android-sdk:35 ./gradlew --no-daemon -q $tasks
fi
lower=$(echo "$variant" | tr 'A-Z' 'a-z')
cp "android/app/build/outputs/apk/$lower/app-$lower.apk" dist/autowifi-bits.apk
echo "built dist/autowifi-bits.apk ($lower)"
