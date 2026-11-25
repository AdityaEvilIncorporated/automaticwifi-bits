#!/bin/bash
# Full project check. Used by the gate, by Claude, and by you before merging to main.
# SKIP_TESTS=1 runs everything except the test suites (the gate uses this while other
# tasks' tests are still meant to fail; it runs the finished tasks' tests itself).
set -e
if [ -f package.json ]; then
  npm run lint --if-present
  npm run typecheck --if-present
  [ -z "$SKIP_TESTS" ] && npm test --if-present
  npm audit --audit-level=high
fi
if [ -f pyproject.toml ] || ls tests/test_*.py >/dev/null 2>&1 || ls ./*.py app/*.py >/dev/null 2>&1; then
  # Python project, with or without pyproject.toml yet (prep adds it): never skip silently.
  ruff check .
  [ -z "$SKIP_TESTS" ] && { ls tests/test_*.py >/dev/null 2>&1 || [ -f pyproject.toml ]; } && python3 -m pytest -q
fi
if [ -f go.mod ]; then
  # Go desktop client. gofmt, vet and build for all three OSes (darwin needs cgo for the tray).
  unformatted=$(gofmt -l $(git ls-files -co --exclude-standard '*.go'))
  [ -z "$unformatted" ] || { echo "gofmt needed: $unformatted"; exit 1; }
  go vet ./...
  GOOS=windows go vet ./...
  GOOS=linux go vet ./...
  go build ./...
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
  [ -z "$SKIP_TESTS" ] && go test -race -count=1 ./...
  if command -v govulncheck >/dev/null; then govulncheck ./...; fi
fi
if [ -d android/core ] && [ -z "$SKIP_TESTS" ] && [ -z "$SKIP_ANDROID" ]; then
  # Kotlin core (protocol, client, engine) in a JDK container against the Go mock portal.
  if docker info >/dev/null 2>&1; then bash scripts/test-android.sh; else echo "verify: docker not running, Kotlin tests NOT run" >&2; exit 1; fi
fi
if command -v gitleaks >/dev/null; then gitleaks detect --no-banner --redact; fi
if command -v semgrep >/dev/null; then semgrep --config auto --error --quiet --exclude node_modules --exclude .venv .; fi
if ls playwright.config.* > /dev/null 2>&1 && [ -z "$SKIP_TESTS" ] && [ -z "$SKIP_E2E" ]; then npx playwright test; fi
echo VERIFY OK
