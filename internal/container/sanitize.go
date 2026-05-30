package container

import "strings"

// stripControlBytes removes C0 control characters except tab (\x09), newline
// (\x0a) and carriage return (\x0d). A NUL byte terminates the clipboard string
// on Windows, dropping everything after it.
func stripControlBytes(r rune) rune {
	if (r >= 0x00 && r <= 0x08) || r == 0x0b || r == 0x0c || (r >= 0x0e && r <= 0x1f) {
		return -1
	}
	return r
}

// SanitizeForPlainText strips ANSI escape sequences and control bytes so log
// text is safe to copy to the clipboard or open in a text editor.
func SanitizeForPlainText(str string) string {
	return strings.Map(stripControlBytes, StripANSI(str))
}
