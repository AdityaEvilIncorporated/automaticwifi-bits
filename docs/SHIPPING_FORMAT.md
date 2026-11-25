# SHIPPING.md format (for a human who deploys by hand)
- Architecture in 5 lines.
- Accounts needed (from intake/ACCOUNTS.md): signup URL, free-tier limits to check, card needed or not,
  and the exact API token to create for scripts/provision.sh (narrowest scope).
- One command after the accounts exist: `bash scripts/provision.sh` (idempotent; what it creates, how to undo it).
- Numbered deploy steps with exact commands and dashboard clicks, one action per step, each marked [HUMAN] or [CHECK].
- Environment variable table: name, purpose, where to get it (no values).
- Database: creation, migrations, backups and restore test.
- Domain and DNS records.
- Payments or email: test mode first, then go-live steps.
- Post-deploy smoke test (exact URLs and expected results).
- Rollback for each step.
- Monitoring and alerts.
- Cost estimate and billing alerts.
- Pre-launch security checklist.
