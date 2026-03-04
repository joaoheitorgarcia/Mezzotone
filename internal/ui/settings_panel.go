package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joaoheitorgarcia/Mezzotone/internal/termtext"

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

type RenderSettingsStyles struct {
	BoxStyle        lipgloss.Style
	TitleStyle      lipgloss.Style
	LabelStyle      lipgloss.Style
	ValueStyle      lipgloss.Style
	SelectedStyle   lipgloss.Style
	ConfirmBtnStyle lipgloss.Style
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

	Styles RenderSettingsStyles
}

func NewSettingsPanel(title string, items []SettingItem, styles RenderSettingsStyles) SettingsPanel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64

	return SettingsPanel{
		Title:  title,
		Items:  items,
		input:  ti,
		Styles: styles,
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
				if it, ok := m.currentItem(); ok {
					it.Value = m.beforeEdit
				}
				return *m, nil

			case "enter":
				raw := strings.TrimSpace(m.input.Value())
				it, ok := m.currentItem()
				if !ok {
					m.errMsg = ""
					m.Editing = false
					m.input.Blur()
					m.input.SetValue("")
					return *m, nil
				}

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
			it, ok := m.currentItem()
			if !ok {
				return *m, nil
			}

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
	innerW := max(1, m.width-2-4 /*border + padding left+right*/)
	gapW := 2

	// Keep value readable, but always size columns from the real available width.
	valueW := min(10, max(1, innerW/3))
	labelW := max(1, innerW-gapW-valueW)

	lines := []string{m.Styles.TitleStyle.Render(termtext.TruncateLinesANSI(strings.ToUpper(m.Title), labelW)), ""}

	for i, it := range m.Items {
		val := it.Value
		if m.Editing && i == m.cursor {
			m.input.Width = valueW
			val = m.input.View()
		}

		leftText := termtext.TruncateLinesANSI(it.Label, labelW)
		rightText := termtext.TruncateLinesANSI(val, valueW)

		left := m.Styles.LabelStyle.MaxWidth(labelW).Width(labelW).Render(leftText)
		right := m.Styles.ValueStyle.Width(valueW).Render(rightText)

		row := left + strings.Repeat(" ", gapW) + right
		if i == m.cursor {
			left = lipgloss.NewStyle().MaxWidth(labelW).Width(labelW).Render(leftText)
			right = lipgloss.NewStyle().Width(valueW).Render(rightText)
			row = left + strings.Repeat(" ", gapW) + right
			row = m.Styles.SelectedStyle.Render(row)
		}
		lines = append(lines, row)
	}

	confirmText := termtext.TruncateLinesANSI("CONFIRM", labelW)
	confirmButton := m.Styles.ConfirmBtnStyle.Width(labelW + valueW).Render(confirmText)
	if m.cursor == len(m.Items) {
		confirmButton = lipgloss.NewStyle().Width(labelW + valueW).Render(confirmText)
		confirmButton = m.Styles.SelectedStyle.Render(confirmButton)
	}
	lines = append(lines, "\n"+confirmButton)
	return m.Styles.BoxStyle.Render(strings.Join(lines, "\n"))
}

func (m *SettingsPanel) toggleBool() {
	it, ok := m.currentItem()
	if !ok {
		return
	}
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
	it, ok := m.currentItem()
	if !ok {
		return
	}
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

func (m *SettingsPanel) ErrorMessage() string {
	return m.errMsg
}

func (m *SettingsPanel) currentItem() (*SettingItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.Items) {
		return nil, false
	}
	return &m.Items[m.cursor], true
}
