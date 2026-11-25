package autostart

import (
	"strings"
	"testing"
)

func TestLaunchAgentPlist(t *testing.T) {
	p := LaunchAgentPlist("/Applications/Auto WiFi/autowifi&<x>")
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		"<key>Label</key>", "<string>" + Label + "</string>",
		"<key>ProgramArguments</key>",
		"<string>/Applications/Auto WiFi/autowifi&amp;&lt;x&gt;</string>",
		"<string>run</string>",
		"<key>RunAtLoad</key>", "<true/>",
		"<key>KeepAlive</key>", "<key>SuccessfulExit</key>", "<false/>",
		"<key>ProcessType</key>", "<string>Background</string>",
		"<key>LimitLoadToSessionType</key>", "<string>Aqua</string>",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("plist missing %q:\n%s", want, p)
		}
	}
	if strings.Contains(p, "StandardOutPath") || strings.Contains(p, "&<x>") {
		t.Fatal("plist has log paths or unescaped exe")
	}
	if strings.Index(p, "/Applications") > strings.Index(p, "<string>run</string>") {
		t.Fatal("exe must come before run")
	}
}

func TestDesktopEntry(t *testing.T) {
	got := DesktopEntry(`/home/u/my "apps"/$bin/auto\wifi`)
	want := "[Desktop Entry]\nType=Application\nName=AutoWiFi BITS\nComment=Logs in to BITS campus Wi-Fi automatically\n" +
		`Exec="/home/u/my \"apps\"/\$bin/auto\\wifi" run` + "\n" +
		"Terminal=false\nNoDisplay=true\nX-GNOME-Autostart-enabled=true\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRunValue(t *testing.T) {
	if got := RunValue(`C:\Users\a b\autowifi.exe`); got != `"C:\Users\a b\autowifi.exe" run` {
		t.Fatalf("got %s", got)
	}
}
