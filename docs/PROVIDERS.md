# Free providers for automatic deploys (VERIFY: free tiers change often; checked 13 Nov 2026)
Pick the first that fits. All are free without a card at the time of writing. If a sign-up now asks
for a card, stop (need-human.sh card) and suggest the next provider in the same row to the human.

| Need | First choice | Next | Notes |
| --- | --- | --- | --- |
| Web app / API (Docker or buildpack) | Render (free web service) | Railway free plan | Render free services sleep after ~15 idle minutes; first request takes about a minute. |
| Postgres | Neon (permanent free tier) | Supabase | Render's free Postgres expires after 30 days: do not use it for anything kept. |
| Static site / frontend | Cloudflare Pages | GitHub Pages, Vercel (non-commercial only) | |
| Transactional email | Resend (free tier) | Brevo | Needs a domain for real recipients; without one, send only to the agent mailbox and say so in DEPLOYED.md. |
| File storage | Supabase Storage | Cloudflare R2 (may need a card) | |
| Redis | Upstash (free) | | |
| Payments | Stripe in TEST mode only | | Live mode, KYC and bank details are always the human's. |
| Domain | none (provider subdomain) | | Buying a domain costs money: human only. |
