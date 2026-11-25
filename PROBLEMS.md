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
