# Build-station rules (interactive Antigravity use in this repo)
- Same rules as GEMINI.md. When you use /swarm or subagents here by hand, give every worker
  disjoint files, ask for a verifier that runs `bash scripts/verify.sh`, and prefer Gemini 4
  Hydrogen workers to save quota.
- Never merge to `dev` or `main` yourself: leave changes on a `feat/*` branch for Claude's review.
