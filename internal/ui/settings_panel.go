package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingType int

const (
	TypeInt SettingType = iota
	TypeFloat
	TypeBool
	TypeEnum
)

type SettingItem struct {
	Key   string
	Type  SettingType
	Label string
	Value string
	Enum  []string
}

type SettingsPanel struct {
	Title string
	Items []SettingItem

	cursor     int
	Editing    bool
	beforeEdit string
	errMsg     string
	Confirm    bool

	input         textinput.Model
	width, height int
}

func NewSettingsPanel(title string, items []SettingItem) SettingsPanel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64

	return SettingsPanel{
		Title: title,
		Items: items,
		input: ti,
	}
}

func (m *SettingsPanel) Init() tea.Cmd {
	return nil
}
func (m *SettingsPanel) Update(msg tea.Msg) (SettingsPanel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.Editing {
			switch msg.String() {
			case "esc":
				m.errMsg = ""
				m.Editing = false
				m.input.Blur()
				m.input.SetValue("")
				m.Items[m.cursor].Value = m.beforeEdit
				return *m, nil

			case "enter":
				raw := strings.TrimSpace(m.input.Value())
				it := &m.Items[m.cursor]

				if err := validateAndSet(it, raw); err != nil {
					m.errMsg = err.Error()
					return *m, nil
				}

				m.errMsg = ""
				m.Editing = false
				m.input.Blur()
				m.input.SetValue("")
				return *m, nil

			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return *m, cmd
			}
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.Confirm = false
				m.cursor--
			}
			if m.cursor == len(m.Items) {
				m.Confirm = true
			}
			m.errMsg = ""
			return *m, nil

		case "down", "j":
			if m.cursor < len(m.Items) {
				m.Confirm = false
				m.cursor++
			}
			if m.cursor == len(m.Items) {
				m.Confirm = true
			}
			m.errMsg = ""
			return *m, nil

		case "left", "h":
			m.errMsg = ""
			m.stepEnum(-1)
			return *m, nil

		case "right", "l":
			m.errMsg = ""
			m.stepEnum(+1)
			return *m, nil

		case " ", "space":
			m.errMsg = ""
			m.toggleBool()
			return *m, nil

		case "enter":
			m.errMsg = ""
			if m.cursor == len(m.Items) {
				return *m, nil
			}
			it := &m.Items[m.cursor]

			if it.Type == TypeBool {
				m.toggleBool()
				return *m, nil
			}
			if it.Type == TypeEnum {
				m.stepEnum(+1)
				return *m, nil
			}

			m.Editing = true
			m.beforeEdit = it.Value
			m.input.SetValue(it.Value)
			m.input.CursorEnd()
			m.input.Focus()
			return *m, nil
		}
	}

	return *m, nil
}

func (m *SettingsPanel) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(1, 2).
		Width(m.width)

	title := lipgloss.NewStyle().
		Bold(true).
		Render(strings.ToUpper(m.Title))

	labelStyle := lipgloss.NewStyle()
	valueStyle := lipgloss.NewStyle()

	selected := lipgloss.NewStyle().Reverse(true)
	//errStyle := lipgloss.NewStyle().Faint(true)

	innerW := max(20, m.width-2-4 /*border + padding left+right*/)

	labelW := max(18, innerW/2)
	valueW := min(10, innerW-labelW-2)

	lines := []string{title, ""}

	for i, it := range m.Items {
		val := it.Value
		if m.Editing && i == m.cursor {
			m.input.Width = valueW
			val = m.input.View()
		}

		left := labelStyle.Width(labelW).Render(it.Label)
		right := valueStyle.Width(valueW).Render(val)

		row := left + "  " + right
		if i == m.cursor {
			row = selected.Render(row)
		}
		lines = append(lines, row)
	}

	//TODO hint and erro line bellow viewport
	//if m.errMsg != "" {
	//	lines = append(lines, "")
	//	lines = append(lines, errStyle.Render("⚠ "+m.errMsg))
	//} else {
	//	lines = append(lines, "")
	//	lines = append(lines, errStyle.Render(hintLine(m)))
	//}
	confirmButton := labelStyle.Width(labelW + valueW).Render("CONFIRM")
	if m.cursor == len(m.Items) {
		confirmButton = selected.Render(confirmButton)
	}
	lines = append(lines, "\n"+confirmButton)
	return box.Render(strings.Join(lines, "\n"))
}

func (m *SettingsPanel) toggleBool() {
	it := &m.Items[m.cursor]
	if it.Type != TypeBool {
		return
	}
	switch strings.ToLower(strings.TrimSpace(it.Value)) {
	case "true":
		it.Value = "FALSE"
	default:
		it.Value = "TRUE"
	}
}

func (m *SettingsPanel) stepEnum(dir int) {
	it := &m.Items[m.cursor]
	if it.Type != TypeEnum || len(it.Enum) == 0 {
		return
	}
	cur := indexOf(it.Enum, it.Value)
	if cur < 0 {
		cur = 0
	}
	next := (cur + dir) % len(it.Enum)
	if next < 0 {
		next += len(it.Enum)
	}
	it.Value = it.Enum[next]
}

func validateAndSet(it *SettingItem, raw string) error {
	switch it.Type {
	case TypeInt:
		if _, err := strconv.Atoi(raw); err != nil {
			return fmt.Errorf("must be an integer")
		}
		it.Value = raw
		return nil

	case TypeFloat:
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return fmt.Errorf("must be a number")
		}
		it.Value = raw
		return nil

	case TypeBool:
		switch strings.ToLower(raw) {
		case "true", "false":
			it.Value = normalizeBool(raw)
			return nil
		default:
			return fmt.Errorf("must be TRUE/FALSE")
		}

	case TypeEnum:
		for _, opt := range it.Enum {
			if strings.EqualFold(opt, raw) {
				it.Value = opt
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(it.Enum, ", "))

	default:
		it.Value = raw
		return nil
	}
}

func normalizeBool(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return "true"
	default:
		return "false"
	}
}

func indexOf(xs []string, v string) int {
	for i := range xs {
		if xs[i] == v {
			return i
		}
	}
	return -1
}

func (m *SettingsPanel) SetWidth(w int) {
	m.width = w
}

func (m *SettingsPanel) SetHeight(h int) {
	m.height = h
}

func (m *SettingsPanel) ClearActive() {
	m.cursor = -1
}

func (m *SettingsPanel) SetActive(i int) {
	m.cursor = i
}

//func hintLine(m SettingsPanel) string {
//	if m.Editing {
//		return "Enter: save   Esc: cancel"
//	}
//	return "↑/↓: select   Enter: edit/toggle   Space: toggle bool   ←/→: enum"
//}
