package tests

import (
	"testing"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func newRenderSettingsPanelForTests() ui.SettingsPanel {
	return ui.NewSettingsPanel("Render Options", []ui.SettingItem{
		{Label: "Text Size", Key: "textSize", Type: ui.TypeInt, Value: "10"},
		{Label: "Font Aspect", Key: "fontAspect", Type: ui.TypeFloat, Value: "2.3"},
		{Label: "Directional Render", Key: "directionalRender", Type: ui.TypeBool, Value: "FALSE"},
		{Label: "Rune Mode", Key: "runeMode", Type: ui.TypeEnum, Value: "ASCII", Enum: []string{"ASCII", "UNICODE", "DOTS"}},
	})
}

func TestSettingsPanelConfirmFlagTracksConfirmRow(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(0)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Confirm {
		t.Fatalf("confirm should be false on setting rows")
	}

	for range len(m.Items) - 1 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if !m.Confirm {
		t.Fatalf("confirm should be true on confirm row")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Confirm {
		t.Fatalf("confirm should be false after leaving confirm row")
	}
}

func TestSettingsPanelEnterStartsEditingForInt(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(0)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Editing {
		t.Fatalf("expected editing=true after enter on int field")
	}
}

func TestSettingsPanelEnterTogglesBool(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(2)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Items[2].Value != "TRUE" {
		t.Fatalf("expected bool to toggle to TRUE, got %q", m.Items[2].Value)
	}
	if m.Editing {
		t.Fatalf("bool field should not enter text editing mode")
	}
}

func TestSettingsPanelEnumCyclesWithArrowsAndEnter(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(3)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Items[3].Value != "UNICODE" {
		t.Fatalf("expected enum step right to UNICODE, got %q", m.Items[3].Value)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Items[3].Value != "DOTS" {
		t.Fatalf("expected enter to step enum to DOTS, got %q", m.Items[3].Value)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.Items[3].Value != "UNICODE" {
		t.Fatalf("expected enum step left to UNICODE, got %q", m.Items[3].Value)
	}
}

func TestSettingsPanelIntEditSaveAndCancel(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(0)

	// Start editing int field, replace value, and save.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("12")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.Items[0].Value != "12" {
		t.Fatalf("expected saved int value 12, got %q", m.Items[0].Value)
	}
	if m.Editing {
		t.Fatalf("expected editing=false after saving")
	}

	// Enter edit again, modify, cancel with esc, value must remain unchanged.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.Items[0].Value != "12" {
		t.Fatalf("expected cancel to restore previous value, got %q", m.Items[0].Value)
	}
}

func TestSettingsPanelInvalidFloatKeepsEditingAndValue(t *testing.T) {
	m := newRenderSettingsPanelForTests()
	m.SetActive(1)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.Editing {
		t.Fatalf("expected editing to remain true after invalid float input")
	}
	if m.Items[1].Value != "2.3" {
		t.Fatalf("invalid input should not change stored value, got %q", m.Items[1].Value)
	}
}
