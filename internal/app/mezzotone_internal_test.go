package app

import (
	"strings"
	"testing"

	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
	"github.com/joaoheitorgarcia/Mezzotone/internal/termtext"
	"github.com/joaoheitorgarcia/Mezzotone/internal/ui"

	"charm.land/bubbles/v2/viewport"
)

func TestUpdateMessageViewPortContent_TruncatesByLeftColumnWidth(t *testing.T) {
	messageViewContent := "this is a long status line that must be truncated"
	model := &MezzotoneModel{
		messageViewPort: viewport.New(viewport.WithWidth(8), viewport.WithHeight(3)),
		style: styleVariables{
			leftColumnWidth: 8,
		},
	}

	model.updateMessageViewPortContent(messageViewContent, false)
	view := model.messageViewPort.View()
	expectedFirstLine := termtext.TruncateLinesANSI(messageViewContent, model.style.leftColumnWidth-2)

	if !strings.Contains(view, expectedFirstLine) {
		t.Fatalf("expected viewport to contain truncated first line %q, got %q", expectedFirstLine, view)
	}
}

func TestNormalizeRenderOptionsForService_MapsRenderColor(t *testing.T) {
	settings := []ui.SettingItem{
		{Key: "renderMode", Value: global.CHARACTER_RAMP},
		{Key: "textSize", Value: "8"},
		{Key: "fontAspect", Value: "2.0"},
		{Key: "directionalRender", Value: "FALSE"},
		{Key: "edgeThreshold", Value: "0.6"},
		{Key: "reverseChars", Value: "TRUE"},
		{Key: "highContrast", Value: "TRUE"},
		{Key: "renderColor", Value: "TRUE"},
		{Key: "runeMode", Value: global.ASCII},
	}

	opts, err := normalizeRenderOptionsForService(settings)
	if err != nil {
		t.Fatalf("normalizeRenderOptionsForService returned error: %v", err)
	}

	if !opts.RenderColor {
		t.Fatalf("expected RenderColor to be true")
	}
}

func TestNormalizeRenderOptionsForService_ValidatesDensity(t *testing.T) {
	settings := []ui.SettingItem{
		{Key: "renderMode", Value: global.BRAILLE},
		{Key: "textSize", Value: "8"},
		{Key: "fontAspect", Value: "2.0"},
		{Key: "density", Value: "1.5"},
		{Key: "dithering", Value: "TRUE"},
	}

	_, err := normalizeRenderOptionsForService(settings)
	if err == nil {
		t.Fatalf("expected density validation error")
	}
	if !strings.Contains(err.Error(), "density") {
		t.Fatalf("expected density error, got %v", err)
	}
}

func TestNormalizeRenderOptionsForService_ValidatesEdgeThreshold(t *testing.T) {
	settings := []ui.SettingItem{
		{Key: "renderMode", Value: global.BRAILLE},
		{Key: "textSize", Value: "8"},
		{Key: "fontAspect", Value: "2.0"},
		{Key: "edgeThreshold", Value: "1.5"},
		{Key: "dithering", Value: "TRUE"},
	}

	_, err := normalizeRenderOptionsForService(settings)
	if err == nil {
		t.Fatalf("expected edge threshold validation error")
	}
	if !strings.Contains(err.Error(), "edge threshold") {
		t.Fatalf("expected edgeThreshold error, got %v", err)
	}
}
