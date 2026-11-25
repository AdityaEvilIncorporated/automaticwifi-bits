# Schedule (5-hour windows; 25% of each kept for review)

| Window | Supervisor (HARD + review) | Workers in parallel (EASY/MEDIUM) |
|---|---|---|
| 1 (done) | Planning: protocol, shared cases, mock portal, contracts, failing tests, briefs | none |
| 2 | T07 tlsconf (1 h), then review queue | T01 T02 T03 T04 T08 T10 T11 T12 T21 T23 T30 T31 (all independent) |
| 3 | Review; T14 agent + CLI + tray + e2e (3 h) | T05, T10W, T22 (after T21), then T06 and T09 as their deps merge |
| 4 | Review; T24 Android app + PortalTls + APK (3 h) | T13 (after T06), T32 (after T14) |
| 5 | Definition of done: security-reviewer, e2e-runner, second-opinion release, TEST_REPORT, SHIPPING, deploy (GitHub repo, Release, Pages) | fixes from review |

Critical path: T01+T04 → T05 → T06 → T13/T14 → T32 → release. T07 is first in window 2 because T06
and T13 need it.
