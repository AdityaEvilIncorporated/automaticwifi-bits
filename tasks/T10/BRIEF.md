# T10: Autostart: file generators, macOS LaunchAgent, Linux XDG

## Goal
Register the agent to start at login for the current user: pure generators for the file contents, plus install/uninstall on macOS (LaunchAgent) and Linux (XDG autostart).

## Implement
Docstrings in internal/autostart/*.go are the spec.
- `LaunchAgentPlist`: standard Apple plist with `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`; escape exe with `html.EscapeString`-equivalent for & < > " '. Tab indentation.
- `DesktopEntry`, `RunValue`: exact strings (see tests).
- darwin: `launchctl` package var defaults to `exec.Command("/usr/bin/launchctl", args...).Run()`; install: `filepath.IsAbs(exe)` else error; MkdirAll(LaunchAgents, 0o755); write the plist 0o644; `launchctl("bootout", target)` errors ignored, then `launchctl("bootstrap", "gui/"+uid, path)` where uid = `strconv.Itoa(os.Getuid())` (ignore an error whose message contains "already"; otherwise return it). Uninstall: bootout (ignore error), remove file (missing ok). Installed: file exists.
- linux: dir = $XDG_CONFIG_HOME/autostart if set, else <home>/.config/autostart; MkdirAll 0o755; file 0o644.
- Idempotent: installing twice rewrites the same content without error.

## Imitate
Standard library only.

## Acceptance
`go test ./internal/autostart/ -count=1 && GOOS=linux go test -c -o /dev/null ./internal/autostart/` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
