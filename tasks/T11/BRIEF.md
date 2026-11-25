# T11: OS notifiers (osascript, notify-send, WinRT toast)

## Goal
Show quiet desktop notifications on each OS with one external command, never interpolating text into a script.

## Implement
Implement `newSystem()` in the three files exactly as the NewSystem docstring in internal/notify/system.go says. Each returns a small struct implementing Notifier.
- Use `exec.CommandContext` with a 10 s `context.WithTimeout`; return `cmd.Run()`'s error.
- linux: `exec.LookPath("notify-send")` failing → Notify returns nil.
- windows: set `cmd.Env = append(os.Environ(), "AWB_TITLE="+title, "AWB_BODY="+body)` and `cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}`. The PowerShell script is a constant: load `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]`, get template ToastText02, set text nodes 0/1 from $env:AWB_TITLE/$env:AWB_BODY via `.AppendChild($xml.CreateTextNode(...))`, `CreateToastNotifier("AutoWiFi BITS").Show(...)`.

## Imitate
Standard library only.

## Acceptance
`go test ./internal/notify/ -count=1 -run TestNewSystemNotNil && GOOS=linux go vet ./internal/notify/ && GOOS=windows go vet ./internal/notify/` must pass. Then `SKIP_TESTS=1 bash scripts/verify.sh` (gofmt, vet, cross-builds, gitleaks, semgrep) must pass.

## Data
No database in this project. Tests use temp dirs (`t.TempDir()`), the in-process mock portal (internal/mockportal) or fakes.

## Do not
- change any exported signature, type, constant or docstring in the contract files
- edit any test, protocol/testcases/*.json, go.mod or go.sum, or add dependencies
- touch files not listed in TASKS.json `files`
- log, print or put into errors any password, username, magic or token
- leave `panic("not implemented")`/`TODO(...)` in the functions you own
