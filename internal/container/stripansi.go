package container

import (
	"regexp"
	"strings"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripANSI(str string) string {
	// Every sequence starts with ESC or CSI, and almost no line has either, so the
	// regex only runs on the lines that can match.
	if !strings.ContainsAny(str, "\u001B\u009B") {
		return str
	}
	return re.ReplaceAllString(str, "")
}
