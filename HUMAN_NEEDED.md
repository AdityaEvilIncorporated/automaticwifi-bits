BLOCKED: two GitHub repository settings need the repo owner (v0.1.0 is built and tested, waiting only on these).

WHY: The `gh` token on this Mac cannot write Actions secrets (HTTP 403 "Resource not accessible by
personal access token"), and the deploy browser is not signed in to GitHub. The Android signing key
is already generated and stored in ~/.agent-secrets (provider `android-automaticwifi-bits`); it is
not in the repo and must not be pasted anywhere else.

STEPS:
1. Let the `gh` token write secrets for this repo. If it is a fine-grained token: open
   https://github.com/settings/personal-access-tokens, edit the token `gh` uses on this Mac,
   Repository access must include `automaticwifi-bits`, then Permissions → Repository →
   **Secrets: Read and write** → Update. (Classic token: it needs the `repo` scope.)
   Alternatively, run it yourself in this session after `gh auth login` with an account token:
   `! bs-secret.sh run android-automaticwifi-bits -- bash scripts/set-gh-secrets.sh`
2. Open https://github.com/AdityaEvilIncorporated/automaticwifi-bits/settings/pages and under
   "Build and deployment" set **Source: GitHub Actions**. (Nothing else; no custom domain.)

VERIFY: `bs-secret.sh run android-automaticwifi-bits -- bash scripts/set-gh-secrets.sh` lists the four
ANDROID_* secrets; then the agent pushes tag v0.1.0 and checks the release and pages workflow runs.

REPLY: done
