package app_test

import (
	"strings"
	"testing"

	"github.com/joaoheitorgarcia/Mezzotone/internal/app"

	tea "charm.land/bubbletea/v2"
)

func key(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func keyText(text string) tea.KeyPressMsg {
	var code rune
	if len(text) > 0 {
		code = []rune(text)[0]
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

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
	model, ok := updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	view := model.View().Content
	if strings.TrimSpace(view) == "" {
		t.Fatalf("expected non-empty view after resize")
	}
}

func TestMezzotoneModelEscFromFilePickerRequiresDoubleEscToQuit(t *testing.T) {
	m := app.NewMezzotoneModel()

	updated, cmd := m.Update(key(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected first esc from file picker to not quit")
	}

	updatedModel, ok := updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	updated, cmd = updatedModel.Update(key(tea.KeyEsc))
	if cmd == nil {
		t.Fatalf("expected quit command on second esc from file picker")
	}

	if msg := cmd(); msg == nil {
		t.Fatalf("expected quit command to return a message")
	}
}

func TestMezzotoneModelEscQuitIsCanceledByOtherKey(t *testing.T) {
	m := app.NewMezzotoneModel()

	updated, cmd := m.Update(key(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected first esc from file picker to not quit")
	}

	updatedModel, ok := updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	updated, cmd = updatedModel.Update(keyText("j"))
	if cmd != nil {
		t.Fatalf("expected non-esc key to not quit")
	}

	updatedModel, ok = updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	updated, cmd = updatedModel.Update(key(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected esc after reset to not quit")
	}

	updatedModel, ok = updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	_, cmd = updatedModel.Update(key(tea.KeyEsc))
	if cmd == nil {
		t.Fatalf("expected second esc after reset to quit")
	}
}

func TestMezzotoneModelHelpToggleRendersAndHidesHelp(t *testing.T) {
	m := app.NewMezzotoneModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, ok := updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	updated, _ = model.Update(keyText("h"))
	model, ok = updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	helpView := model.View().Content
	if !strings.Contains(helpView, "CONTROLS") {
		t.Fatalf("expected help overlay to render after pressing h")
	}

	updated, _ = model.Update(keyText("h"))
	model, ok = updated.(*app.MezzotoneModel)
	if !ok {
		t.Fatalf("expected updated model type *app.MezzotoneModel")
	}

	viewWithoutHelp := model.View().Content
	if strings.Contains(viewWithoutHelp, "CONTROLS") {
		t.Fatalf("expected help overlay to hide after pressing h again")
	}
}
