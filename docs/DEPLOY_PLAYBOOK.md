# Deploy playbook (Claude runs this; the human only answers HUMAN STEP pings)

Start when the Definition of done in CLAUDE.md is complete and `.agent/DEPLOY_MODE` says `auto`
(if it says `ask`: notify-human.sh 'READY to deploy __PROJECT__: reply deploy __PROJECT__' and wait).

## 0. Rules
- Free tiers that need no card only (docs/PROVIDERS.md). Never pick a paid plan, never start a trial
  that converts to paid, never enter or store card details. A card or paid-plan page -> need-human.sh card.
- One account per provider for the whole station: check `bs-secret.sh list` before signing up.
  Reuse existing accounts and create a new project/service inside them.
- Sign-up email: `bs-secret.sh email <provider>` (agent+<provider>@... in the agent mailbox).
- Passwords: `bs-secret.sh new-password <provider>`, then type the SECRET NAME it prints into the
  password field of the deploybrowser; the browser fills in the real value.
- Never bypass, automate or outsource a CAPTCHA. Never use an SMS-receiving service. Never accept
  terms beyond the standard sign-up terms of a free plan; if a page asks for anything unusual
  (identity documents, business details, a phone call): need-human.sh other.
- Every command that needs provider credentials runs as `bs-secret.sh run <providers> -- <cmd>`.
- need-human.sh waits at most 9 minutes per call. If it prints `still-waiting` (exit 5), run
  `need-human.sh wait` again; keep doing that until it prints done, a code, skip (exit 3) or
  timeout (exit 4). On skip/timeout: choose the next provider in docs/PROVIDERS.md or stop and
  write the state into DEPLOYED.md.

## 1. Accounts (deploybrowser MCP = the visible Chrome on the Mac)
For each provider in the plan that `bs-secret.sh list` does not show yet:
1. Open its sign-up page. Prefer "sign up with email" (not "with Google/GitHub").
2. Email = `bs-secret.sh email <provider>`; password = the secret name from `bs-secret.sh new-password <provider>`.
3. CAPTCHA (any kind, including "I'm not a robot" boxes that turn into puzzles): run
   `need-human.sh captcha "<provider> sign-up"` and wait for it to return, then take a fresh snapshot.
4. Email verification: `mail-otp.py --to <that email> --from <provider domain> --since <epoch before step 2>`
   prints the code; add `--link` when the email holds a link, then open the link in the deploybrowser.
5. Phone verification: type the human's number only if PHONE_NUMBER is set in ~/.agent/station.env
   (`grep ^PHONE_NUMBER ~/.agent/station.env`), click send, then `need-human.sh phone-otp "<provider>"`
   prints the SMS code; type it. If PHONE_NUMBER is not set, use need-human.sh other.
6. Turn on 2FA if offered with an authenticator app: choose "enter key manually", pipe the key into
   `bs-secret.sh totp-add <provider>`, confirm with `bs-secret.sh totp <provider>`. Save recovery codes:
   `printf '%s' "<codes>" | bs-secret.sh put <provider> RECOVERY_CODES`.
7. Create the narrowest API token the deploy needs (one project/service, no billing or team admin
   scope, expiry if offered) and store it: `printf '%s' "<token>" | bs-secret.sh put <provider> <PROVIDER>_API_TOKEN`.
   Close the token page right after.

## 2. Provision and deploy
0. `second-opinion.sh release` (skipped by itself unless allow-cloud). Fix every "block" issue first.
1. Write scripts/provision.sh (idempotent, provider CLIs or REST APIs, reads tokens from env):
   production database, app service, env vars (generated secrets: never reuse .env values), buckets,
   email domain or sender, OAuth redirect URLs, Stripe webhook in TEST mode.
2. `T=release-$(date +%Y%m%d-%H%M); git tag "$T" && git push origin "$T"` (deploy exactly this tested commit of dev).
3. `bs-secret.sh run <providers> -- bash scripts/provision.sh`
4. Production migrations run inside provision.sh with the stack's migration tool and the production
   DATABASE_URL (scripts/db.sh is local only and refuses other hosts).
5. Create the production admin with a new password: `bs-secret.sh new-password __PROJECT__-admin`.

## 3. Verify and report
1. Smoke test the live URL with Playwright: home page, sign-up (real flow, mail through the provider:
   read it with mail-otp.py using `bs-secret.sh email __PROJECT__-smoke`), login, one main flow,
   logout. No load tests. Delete the smoke-test user afterwards.
2. Write DEPLOYED.md: URLs, providers and plans (all free), what was created, how to roll back,
   limits (sleeping services, 30-day databases, email quotas). Commit on dev.
3. `gh pr create --base main --head dev --title "Release <tag>" --body-file DEPLOYED.md`
   (main stays the human's record of what is live; merging it is one tap and changes nothing live).
4. notify-human.sh 'LIVE: __PROJECT__ at <url>. Free tiers only. Details in DEPLOYED.md and the PR.'

## Rollback (Telegram: rollback __PROJECT__)
Redeploy the previous release tag through the provider (provision.sh with RELEASE=<tag>), or the
provider's "rollback to previous deploy" button in the deploybrowser. Then smoke test and notify.
