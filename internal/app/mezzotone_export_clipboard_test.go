package app

import (
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"golang.design/x/clipboard"
)

func TestMezzotoneModelExportTxtSavesRenderedContentToHome(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	fixedUUID := uuid.MustParse("41c92b29-4eb7-4f33-bf3c-8a3d29efe330")
	previousNewUUID := newUUID
	newUUID = func() uuid.UUID { return fixedUUID }
	t.Cleanup(func() { newUUID = previousNewUUID })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = "rendered-output"
	m.style.leftColumnWidth = 120
	m.renderedImgOutput = renderedImgOutput{
		renderedRunes: [][]rune{
			[]rune("rendered-output"),
		},
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	exportPath := filepath.Join(tmpHome, "Mezzotone_"+fixedUUID.String()+".txt")
	t.Cleanup(func() {
		if err := os.Remove(exportPath); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to remove exported file %q: %v", exportPath, err)
		}
	})
	got, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("expected exported file at %q, got read error: %v", exportPath, err)
	}
	if string(got) != m.renderContent {
		t.Fatalf("expected exported file content %q, got %q", m.renderContent, string(got))
	}
}

func TestMezzotoneModelExportPngCreatesValidPNG(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	fixedUUID := uuid.MustParse("1f3870be-274f-46c4-9c95-2f95f71f0111")
	previousNewUUID := newUUID
	newUUID = func() uuid.UUID { return fixedUUID }
	t.Cleanup(func() { newUUID = previousNewUUID })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = "rendered-output"
	m.style.leftColumnWidth = 120
	m.renderedImgOutput = renderedImgOutput{
		renderedRunes: [][]rune{
			[]rune("rendered-output"),
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Fatalf("expected png export command")
	}
	if !strings.Contains(m.messageViewPort.View(), "Exporting image to") {
		t.Fatalf("expected exporting image message before command completion, got %q", m.messageViewPort.View())
	}
	_, _ = m.Update(cmd())

	exportPath := filepath.Join(tmpHome, "Mezzotone_"+fixedUUID.String()+".png")
	f, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("expected png export file at %q, got error: %v", exportPath, err)
	}
	defer f.Close()

	if _, err := png.DecodeConfig(f); err != nil {
		t.Fatalf("expected valid png file at %q, decode failed: %v", exportPath, err)
	}
}

func TestMezzotoneModelExportGifCreatesValidGIF(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	fixedUUID := uuid.MustParse("7d8f4f65-17de-4e98-9f4a-a2ec6e55b019")
	previousNewUUID := newUUID
	newUUID = func() uuid.UUID { return fixedUUID }
	t.Cleanup(func() { newUUID = previousNewUUID })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = "rendered-output"
	m.style.leftColumnWidth = 120
	m.renderedGifOutput = renderedGifOutput{
		renderedRunes: [][][]rune{
			{[]rune("rendered-output")},
		},
		renderedColor: [][][]color.NRGBA{
			{{{R: 255, G: 255, B: 255, A: 255}}},
		},
		delayTimes: []time.Duration{
			50 * time.Millisecond,
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd == nil {
		t.Fatalf("expected gif export command")
	}
	if !strings.Contains(m.messageViewPort.View(), "Exporting gif to") {
		t.Fatalf("expected exporting gif message before command completion, got %q", m.messageViewPort.View())
	}
	_, _ = m.Update(cmd())

	exportPath := filepath.Join(tmpHome, "Mezzotone_"+fixedUUID.String()+".gif")
	f, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("expected gif export file at %q, got error: %v", exportPath, err)
	}
	defer f.Close()

	if _, err := gif.DecodeConfig(f); err != nil {
		t.Fatalf("expected valid gif file at %q, decode failed: %v", exportPath, err)
	}
}

func TestMezzotoneModelExportGifFromAnimationExportsMultipleFrames(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	fixedUUID := uuid.MustParse("49cff2ec-5f00-4092-bf75-8e28f6d5a4fd")
	previousNewUUID := newUUID
	newUUID = func() uuid.UUID { return fixedUUID }
	t.Cleanup(func() { newUUID = previousNewUUID })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = "frame-zero"
	m.style.leftColumnWidth = 120
	m.renderedGifOutput = renderedGifOutput{
		renderedRunes: [][][]rune{
			{[]rune("frame-one")},
			{[]rune("frame-two")},
		},
		renderedColor: [][][]color.NRGBA{
			{{{R: 255, G: 255, B: 255, A: 255}}},
			{{{R: 255, G: 255, B: 255, A: 255}}},
		},
		delayTimes: []time.Duration{
			40 * time.Millisecond,
			80 * time.Millisecond,
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd == nil {
		t.Fatalf("expected gif export command")
	}
	_, _ = m.Update(cmd())

	exportPath := filepath.Join(tmpHome, "Mezzotone_"+fixedUUID.String()+".gif")
	f, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("expected gif export file at %q, got error: %v", exportPath, err)
	}
	defer f.Close()

	decoded, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("expected valid animated gif file at %q, decode failed: %v", exportPath, err)
	}
	if len(decoded.Image) != 2 {
		t.Fatalf("expected exported animated gif to have 2 frames, got %d", len(decoded.Image))
	}
}

func TestMezzotoneModelCopyToClipboardWhenUnavailableShowsError(t *testing.T) {
	previousClipboardOK := clipboardOK
	t.Cleanup(func() { clipboardOK = previousClipboardOK })
	previousClipboardCommands := clipboardCommands
	t.Cleanup(func() { clipboardCommands = previousClipboardCommands })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = "rendered-output"
	m.style.leftColumnWidth = 120
	clipboardOK = false
	clipboardCommands = nil

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !strings.Contains(strings.ToLower(m.messageViewPort.View()), "clipboard not available (init failed)") {
		t.Fatalf("expected clipboard unavailable message, got %q", m.messageViewPort.View())
	}
}

func TestMezzotoneModelCopyToClipboardWithEmptyRenderShowsError(t *testing.T) {
	previousClipboardOK := clipboardOK
	t.Cleanup(func() { clipboardOK = previousClipboardOK })
	previousClipboardCommands := clipboardCommands
	t.Cleanup(func() { clipboardCommands = previousClipboardCommands })

	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderContent = ""
	m.style.leftColumnWidth = 120
	clipboardOK = false
	clipboardCommands = nil

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !strings.Contains(m.messageViewPort.View(), "nothing to copy") {
		t.Fatalf("expected empty render content message, got %q", m.messageViewPort.View())
	}
}

func TestCopyTextToClipboardKeepsContentAsIs(t *testing.T) {
	previousClipboardOK := clipboardOK
	t.Cleanup(func() { clipboardOK = previousClipboardOK })
	previousClipboardWrite := clipboardWrite
	t.Cleanup(func() { clipboardWrite = previousClipboardWrite })
	previousClipboardCommands := clipboardCommands
	t.Cleanup(func() { clipboardCommands = previousClipboardCommands })

	clipboardOK = true
	clipboardCommands = nil

	var gotFormat clipboard.Format
	var gotData []byte
	clipboardWrite = func(format clipboard.Format, data []byte) <-chan struct{} {
		gotFormat = format
		gotData = append([]byte(nil), data...)
		done := make(chan struct{}, 1)
		done <- struct{}{}
		close(done)
		return done
	}

	colored := "\x1b[38;2;255;0;0mAB\x1b[0m\nC"
	if err := copyTextToClipboard(colored); err != nil {
		t.Fatalf("expected copy to succeed, got error: %v", err)
	}
	if gotFormat != clipboard.FmtText {
		t.Fatalf("expected clipboard text format, got %v", gotFormat)
	}
	if string(gotData) != colored {
		t.Fatalf("expected copied content %q, got %q", colored, string(gotData))
	}
}
