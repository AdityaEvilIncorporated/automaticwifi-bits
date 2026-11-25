package notify

// NewSystem returns the OS notifier for this platform (system_darwin.go, system_linux.go,
// system_windows.go). Each implementation runs one external command with the title and body
// passed as separate arguments or environment variables, never interpolated into a shell or
// script string:
//   - darwin:  /usr/bin/osascript -e 'on run argv' -e 'display notification (item 2 of argv)
//     with title (item 1 of argv)' -e 'end run' <title> <body>
//   - linux:   notify-send --app-name=AutoWiFi-BITS --urgency=low <title> <body>
//     (if notify-send is not on PATH: return nil, do nothing)
//   - windows: powershell.exe -NoProfile -NonInteractive -WindowStyle Hidden -Command <script>
//     with the script reading $env:AWB_TITLE and $env:AWB_BODY and showing a WinRT toast
//     (ToastNotificationManager, template ToastText02, shown under PowerShell's registered AppUserModelID because
//     Windows drops toasts from unregistered ids); the command is
//     started with SysProcAttr.HideWindow.
//
// Commands time out after 10 s (exec.CommandContext). Errors are returned, not logged.
func NewSystem() Notifier {
	return newSystem()
}
