#!/bin/bash
# Runs the Kotlin core tests (android/core) in a JDK container, with the Go mock portal.
# Needs Docker. The Gradle cache lives in the docker volume autowifi-gradle.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .build
arch=$(docker info --format '{{.Architecture}}' 2>/dev/null || uname -m)
case "$arch" in aarch64|arm64) goarch=arm64 ;; *) goarch=amd64 ;; esac
GOOS=linux GOARCH=$goarch CGO_ENABLED=0 go build -o .build/mockportal-linux ./cmd/mockportal
docker run --rm -v "$PWD":/work -w /work/android -v autowifi-gradle:/home/gradle/.gradle \
  -e MOCKPORTAL_BIN=/work/.build/mockportal-linux gradle:8.14-jdk17 \
  gradle --no-daemon -q :core:test "$@"
