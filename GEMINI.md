# GEMINI.md: rules for Antigravity agents in automaticwifi-bits

You are a worker in an automated build station. A human and Claude (the supervisor) review
everything you produce. Your job is the task in your prompt, nothing more.

## Hard rules
- Edit only the files your task lists. A guard hook denies other writes; when denied, stop and
  say what you needed instead of finding another way.
- Never commit, push, switch branches, rebase, install packages, or edit lock files, `.env*`,
  `.agents/`, `GEMINI.md`, `AGENTS.md`, `CLAUDE.md`, `.github/`, or CI files.
- Never read or print secrets. `.env` holds local development values only; do not copy them
  into code, tests, logs or your answer.
- Never weaken, skip or delete a test to make it pass. Never add `# noqa`, `@ts-ignore`,
  `eslint-disable` or similar to silence a real problem.

## How to work
1. Tests first: write or extend the task's tests so they fail for the right reason.
2. Implement the smallest change that makes them pass, in the project's existing style.
3. Run the task's test command and include its last lines in your final answer.
4. If the task is ambiguous or needs a design decision, stop and say so in one paragraph.

## Project facts
- Code map: `REPO_MAP.md` (generated). Architecture and decisions: `ARCHITECTURE.md`,
  `docs/`, `DECISIONS.md`. Known pitfalls: `PROBLEMS.md`.
- Full test suite: `bash scripts/verify.sh` if it exists, otherwise the command in your task.
- Local services (database, mail, S3, Redis) are already running for the test suite; the
  gate gives every task its own fresh database.
