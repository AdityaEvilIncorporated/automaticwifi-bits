# T13: Capture diagnostic

## Goal
Implement the `capture` diagnostic: report the portal's form action, field names and types, and TLS facts, never values, so the student can confirm the campus configuration.

## Implement
Follow the numbered steps in the `Capture` docstring in internal/portal/capture.go and the exact line formats in `FormatCapture`.
- Inspect address: `net.JoinHostPort(cfg.PortalHost, strconv.Itoa(cfg.PortalPort))`.
- CertSubject/Issuer: `chain[0].Subject.CommonName` / `chain[0].Issuer.CommonName`.
- Field.Type: the input's (lower-cased) type; an input with no type attribute is "text" in the report.
- The probe result's Magic must not be stored in the report.

## Imitate
Reuse ProbeOnce, ParseForms, OnPortal from the same package.

## Acceptance
`go test ./internal/portal/ -count=1 -run 'TestCapture'` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
