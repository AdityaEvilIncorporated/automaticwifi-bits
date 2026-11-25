# Review checklist (supervisor)
1. Only the task's `files` changed; no test was weakened, skipped or deleted.
2. Behaviour matches the brief and the contract docstrings, including edge cases and errors.
3. No invented APIs, no dead code, no debug prints, no hard-coded secrets or URLs.
4. Inputs validated at the boundary; SQL parameterised; auth checks where the contract says.
5. New dependencies: none, unless the brief allowed one.
6. Names and structure follow the example file.
7. The gate passed (it did, or the task would not be in review), and the test actually exercises the change.
