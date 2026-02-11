package app_test

import (
	"strings"
	"testing"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMezzotoneModelInitReturnsCmd(t *testing.T) {
	m := app.NewMezzotoneModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command to be non-nil")
	}
}

func TestMezzotoneModelWindowResizeRendersView(t *testing.T) {
	m := app.NewMezzotoneModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, ok := updated.(app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type app.MezzotoneModel")
	}

	view := model.View()
	if strings.TrimSpace(view) == "" {
		t.Fatalf("expected non-empty view after resize")
	}
}

func TestMezzotoneModelEscFromFilePickerReturnsQuitCmd(t *testing.T) {
	m := app.NewMezzotoneModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected quit command on esc from file picker")
	}

	_ = updated
	if msg := cmd(); msg == nil {
		t.Fatalf("expected quit command to return a message")
	}
}

func TestMezzotoneModelHelpToggleRendersAndHidesHelp(t *testing.T) {
	m := app.NewMezzotoneModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, ok := updated.(app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type app.MezzotoneModel")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model, ok = updated.(app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type app.MezzotoneModel")
	}

	helpView := model.View()
	if !strings.Contains(helpView, "MEZZOTONE HELP") {
		t.Fatalf("expected help overlay to render after pressing h")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model, ok = updated.(app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type app.MezzotoneModel")
	}

	viewWithoutHelp := model.View()
	if strings.Contains(viewWithoutHelp, "MEZZOTONE HELP") {
		t.Fatalf("expected help overlay to hide after pressing h again")
	}
}
