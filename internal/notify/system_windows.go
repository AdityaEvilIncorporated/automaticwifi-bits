//go:build windows

package notify

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// winToastScript shows the toast under PowerShell's AppUserModelID: Windows 10/11 silently
// drop toasts from an unregistered app id such as "AutoWiFi BITS" (no Start-menu shortcut).
const winToastScript = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$nodes = $xml.GetElementsByTagName('text')
$nodes.Item(0).AppendChild($xml.CreateTextNode($env:AWB_TITLE)) | Out-Null
$nodes.Item(1).AppendChild($xml.CreateTextNode($env:AWB_BODY)) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show($toast)
`

type windowsNotifier struct{}

// newSystem: see NewSystem in system.go for the exact command on windows.
func newSystem() Notifier {
	return windowsNotifier{}
}

func (windowsNotifier) Notify(title, body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-Command", winToastScript,
	)
	cmd.Env = append(os.Environ(), "AWB_TITLE="+title, "AWB_BODY="+body)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
