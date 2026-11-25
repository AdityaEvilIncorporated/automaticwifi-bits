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
if command -v gitleaks >/dev/null; then gitleaks detect --no-banner --redact; fi
if command -v semgrep >/dev/null; then semgrep --config auto --error --quiet --exclude node_modules --exclude .venv .; fi
if ls playwright.config.* > /dev/null 2>&1 && [ -z "$SKIP_TESTS" ] && [ -z "$SKIP_E2E" ]; then npx playwright test; fi
echo VERIFY OK
