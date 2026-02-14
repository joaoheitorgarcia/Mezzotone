package app

import (
	"strings"
	"testing"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/termtext"
	"github.com/charmbracelet/bubbles/viewport"
)

func TestUpdateMessageViewPortContent_TruncatesByLeftColumnWidth(t *testing.T) {
	messageViewContent := "this is a long status line that must be truncated"
	model := &MezzotoneModel{
		messageViewPort: viewport.New(8, 3),
		style: styleVariables{
			leftColumnWidth: 8,
		},
	}

	model.updateMessageViewPortContent(messageViewContent, false)
	view := model.messageViewPort.View()
	expectedFirstLine := termtext.TruncateLinesANSI(messageViewContent, model.style.leftColumnWidth)

	if !strings.Contains(view, expectedFirstLine) {
		t.Fatalf("expected viewport to contain truncated first line %q, got %q", expectedFirstLine, view)
	}
}
