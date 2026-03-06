package termtext

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func TruncateLinesANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	lines := strings.Split(s, "\n")
	for i := range lines {
		if ansi.StringWidth(lines[i]) > maxWidth {
			lines[i] = ansi.Truncate(lines[i], maxWidth, "…")
		}
	}
	return strings.Join(lines, "\n")
}
