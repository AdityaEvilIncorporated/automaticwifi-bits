# Known problems (the "problem repository")
Claude adds an entry when the same failure shows up twice (gate output, review rejections, worker
notes). compile-brief.py puts the latest entries into every worker packet, so no model rediscovers
a trap that is already solved. Keep each entry short; newest at the bottom.

<!-- Format:
## PRB-001: <symptom in a few words>
- Seen in: T03, T07
- Cause: <one line>
- Fix / rule: <one line a worker can follow>
-->

## PRB-001: gate "regression in finished task" for a test the worker never touched
- Seen in: T05 (twice), T09 (twice), 2026-11-25
- Cause: the task worktree was created from dev before another task merged; the gate then runs that
  newly finished task's tests on the stale worktree, where its implementation is still a stub
  (T10 autostart, T23 EngineTest). Not the worker's fault.
- Fix / rule: workers do nothing differently. Supervisor: when a gate failure names another task's
  acceptance command, requeue (status=todo attempts=0) instead of rejecting; the new attempt starts
  from the latest dev. Long-term fix belongs in the orchestrator (rebase worktree on dev before the gate).
