verdict: looks_good (by gemini-4-hydrogen)
Implementation in internal/autostart/autostart_windows.go matches the brief: install validates absolute exe path and sets the HKCU Run key value using RunValue(exe), uninstall deletes the value while treating missing key or value as nil, and installed queries the value while treating missing key or value as false, nil.

Checklist:
- only_allowed_files: yes
- tests_untouched: yes
- matches_brief: yes
- no_secrets_or_debug_code: yes
- error_handling_ok: yes
