package app

import (
	"strings"
	"testing"

	"Mezzotone/internal/termtext"
	"Mezzotone/internal/ui"

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

func TestNormalizeRenderOptionsForService_MapsRenderColor(t *testing.T) {
	settings := []ui.SettingItem{
		{Key: "textSize", Value: "8"},
		{Key: "fontAspect", Value: "2.0"},
		{Key: "directionalRender", Value: "FALSE"},
		{Key: "edgeThreshold", Value: "0.6"},
		{Key: "reverseChars", Value: "TRUE"},
		{Key: "highContrast", Value: "TRUE"},
		{Key: "renderColor", Value: "TRUE"},
		{Key: "runeMode", Value: "ASCII"},
	}

	opts, err := normalizeRenderOptionsForService(settings)
	if err != nil {
		t.Fatalf("normalizeRenderOptionsForService returned error: %v", err)
	}

	if !opts.RenderColor {
		t.Fatalf("expected RenderColor to be true")
	}
}
