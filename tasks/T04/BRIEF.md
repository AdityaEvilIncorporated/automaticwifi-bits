# T04: HTML form parser

## Goal
Parse every HTML form on a page into its action, method and inputs, tolerant of real-world broken markup. The login page parser and the capture diagnostic both build on it.

## Implement
- `ParseForms(body)`: `html.Parse(bytes.NewReader(body))` from golang.org/x/net/html (already in go.mod), walk the node tree; on an `atom.Form` element start a Form (action, lower-cased method); every `atom.Input` below it (any depth) appends `Input{Name, lower-cased Type, Value}` (html.Parse already decodes entities). Inputs outside a form are skipped. nil body → nil; never return an error.
- Note: html.Parse moves inputs that appear after an unclosed form; the test only needs the documented cases.
- `Form.Has(name)`, `Form.Value(name)`: exact, case-sensitive name match, first input wins.

## Imitate
Standard x/net/html tree walk (recursive func over n.FirstChild / n.NextSibling).

## Acceptance
`go test ./internal/portal/ -count=1 -run 'TestParseForms'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
