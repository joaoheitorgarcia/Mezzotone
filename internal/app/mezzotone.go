package app

import (
	"fmt"
	"os"
	"strconv"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/services"
	"codeberg.org/JoaoGarcia/Mezzotone/internal/termtext"
	"codeberg.org/JoaoGarcia/Mezzotone/internal/ui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TODO REORDER Layout IF TERMINAL width < height

const firstScreen = ""

type MezzotoneModel struct {
	filePicker   filepicker.Model
	selectedFile string

	renderView      viewport.Model
	leftColumn      viewport.Model
	renderSettings  ui.SettingsPanel
	messageViewPort viewport.Model

	style styleVariables

	currentActiveMenu int
	helpVisible       bool
	helpPreviousMenu  int
	renderContent     string

	width  int
	height int

	err error
}

type styleVariables struct {
	windowMargin    int
	leftColumnWidth int
}

var renderSettingsItemsSize int
var messageViewMessage string

const (
	filePickerMenu = iota
	renderOptionsMenu
	renderViewText
)

func NewMezzotoneModel() MezzotoneModel {
	windowStyles := styleVariables{
		windowMargin: 2,
	}

	runeMode := []string{"ASCII", "UNICODE", "DOTS", "RECTANGLES", "BARS", "LOADING"}
	renderSettingsItems := []ui.SettingItem{
		{Label: "Text Size", Key: "textSize", Type: ui.TypeInt, Value: "10"},
		{Label: "Font Aspect", Key: "fontAspect", Type: ui.TypeFloat, Value: "2.3"},
		{Label: "Directional Render", Key: "directionalRender", Type: ui.TypeBool, Value: "FALSE"},
		{Label: "Edge Threshold", Key: "edgeThreshold", Type: ui.TypeFloat, Value: "0.6"},
		{Label: "Reverse Chars", Key: "reverseChars", Type: ui.TypeBool, Value: "TRUE"},
		{Label: "High Contrast", Key: "highContrast", Type: ui.TypeBool, Value: "TRUE"},
		{Label: "Rune Mode", Key: "runeMode", Type: ui.TypeEnum, Value: "ASCII", Enum: runeMode},
	}
	renderSettingsItemsSize = len(renderSettingsItems)
	renderSettingsModel := ui.NewSettingsPanel("Render Options", renderSettingsItems)
	renderSettingsModel.ClearActive()

	fp := filepicker.New()
	fp.AllowedTypes = []string{".png", ".jpg", ".jpeg", ".bmp", ".webp", ".tiff"}
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.KeyMap = filepicker.KeyMap{
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		GoToTop:  key.NewBinding(key.WithKeys("K", "pgup"), key.WithHelp("pgup", "page up")),
		GoToLast: key.NewBinding(key.WithKeys("J", "pgdown"), key.WithHelp("pgdown", "page down")),
		Back:     key.NewBinding(key.WithKeys("left", "backspace"), key.WithHelp("h", "back")),
		Open:     key.NewBinding(key.WithKeys("right", "enter"), key.WithHelp("l", "open")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}

	renderView := viewport.New(0, 0)
	leftColumn := viewport.New(0, 0)

	messageViewPort := viewport.New(0, 0)
	messageViewMessage = "Select image gif or video to convert:"
	messageViewPort.SetContent(messageViewMessage + lipgloss.NewStyle().Faint(true).Render("\n\nPress h to toggle Help. Press esc to Quit."))

	return MezzotoneModel{
		filePicker:        fp,
		renderView:        renderView,
		messageViewPort:   messageViewPort,
		style:             windowStyles,
		leftColumn:        leftColumn,
		renderSettings:    renderSettingsModel,
		currentActiveMenu: filePickerMenu,
		helpPreviousMenu:  filePickerMenu,
	}
}

func (m MezzotoneModel) Init() tea.Cmd {
	return m.filePicker.Init()
}

func (m MezzotoneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		m.renderView.Height = m.height - m.style.windowMargin
		m.renderView.Width = m.width / 7 * 5

		m.style.leftColumnWidth = m.width / 7 * 2

		m.renderSettings.SetWidth(m.style.leftColumnWidth)
		m.renderSettings.SetHeight(renderSettingsItemsSize)

		m.messageViewPort.Height = 3
		m.messageViewPort.Width = m.style.leftColumnWidth

		computedFilePickerHeight := m.renderView.Height -
			(renderSettingsItemsSize + 4) - //renderSettings header and end
			(m.messageViewPort.Height + 2) - //message render view
			(m.style.windowMargin + 3) //inputFile Title

		m.filePicker.SetHeight(computedFilePickerHeight)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "h":
			if m.currentActiveMenu == renderOptionsMenu && m.renderSettings.Editing {
				break
			}
			if m.helpVisible {
				m.helpVisible = false
				m.currentActiveMenu = m.helpPreviousMenu
				m.renderView.SetContent(m.renderContent)
				return m, nil
			}
			m.helpVisible = true
			m.helpPreviousMenu = m.currentActiveMenu
			m.currentActiveMenu = renderViewText
			m.renderView.GotoTop()
			m.renderView.SetContent(buildRenderHelpText())
			return m, nil
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.helpVisible {
				m.helpVisible = false
				m.currentActiveMenu = m.helpPreviousMenu
				m.renderView.SetContent(m.renderContent)
				return m, nil
			}
			if m.currentActiveMenu == filePickerMenu {
				//TODO ask for confimation
				return m, tea.Quit
			}
			if m.currentActiveMenu == renderOptionsMenu {
				if !m.renderSettings.Editing {
					m.currentActiveMenu--
					m.renderSettings.ClearActive()
				}
				return m, cmd
			}
			if m.currentActiveMenu == renderViewText {
				m.currentActiveMenu--
				return m, cmd
			}

		case "enter":
			if m.currentActiveMenu == renderOptionsMenu {
				if !m.renderSettings.Editing && m.renderSettings.Confirm {
					m.currentActiveMenu++

					normalizedOptions := normalizeRenderOptionsForService(m.renderSettings.Items)
					runeArray, err := services.ConvertImageToString(m.selectedFile, normalizedOptions)
					if err != nil {
						//TODO HAndle this
					}
					m.renderContent = services.ImageRuneArrayIntoString(runeArray)
					_ = services.Logger().Info(fmt.Sprintf("%s", m.renderContent))
					if !m.helpVisible {
						m.renderView.SetContent(m.renderContent)
					}
					return m, cmd
				}
			}
		case "left":
			if m.currentActiveMenu == renderViewText {
				m.renderView.ScrollLeft(1)
				return m, cmd
			}
		case "right":
			if m.currentActiveMenu == renderViewText {
				m.renderView.ScrollRight(1)
				return m, cmd
			}
		case "up":
			if m.currentActiveMenu == renderViewText {
				m.renderView.ScrollUp(1)
				return m, cmd
			}
		case "down":
			if m.currentActiveMenu == renderViewText {
				m.renderView.ScrollDown(1)
				return m, cmd
			}
		case "pgdown":
			if m.currentActiveMenu == renderOptionsMenu {
				m.renderSettings.SetActive(renderSettingsItemsSize)
				return m, cmd
			}
		case "pgup":
			if m.currentActiveMenu == renderOptionsMenu {
				m.renderSettings.SetActive(renderSettingsItemsSize)
				return m, cmd
			}
		}
	}

	if m.currentActiveMenu == filePickerMenu {
		m.filePicker, cmd = m.filePicker.Update(msg)
		cmds = append(cmds, cmd)
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			m.selectedFile = path
			_ = services.Logger().Info(fmt.Sprintf("Selected File: %s", m.selectedFile))

			m.renderSettings.SetActive(0)
			m.renderSettings.Confirm = false
			m.currentActiveMenu++
			return m, cmd
		}

		if didSelect, path := m.filePicker.DidSelectDisabledFile(msg); didSelect {
			//TODO maybe make a modal here with error ? or no modal but better error info
			m.renderView.SetContent("Selected file need to be an image.\nAllowed types: .png, .jpg, .jpeg, .bmp, .webp, .tiff")
			m.selectedFile = ""
			_ = services.Logger().Info(fmt.Sprintf("Tried Selecting File: %s", path))
			return m, cmd
		}
	}
	if m.currentActiveMenu == renderOptionsMenu {
		m.renderSettings, cmd = m.renderSettings.Update(msg)
		return m, cmd
	}
	if m.currentActiveMenu == renderViewText {
		m.renderView, cmd = m.renderView.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m MezzotoneModel) View() string {
	innerW := m.style.leftColumnWidth - 2

	messageViewportRenderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		Width(m.style.leftColumnWidth)
	messageViewportRender := messageViewportRenderStyle.Render(m.messageViewPort.View())

	filePickerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		Width(m.style.leftColumnWidth)
	fpView := termtext.TruncateLinesANSI(m.filePicker.View(), innerW)
	filePickerRender := filePickerStyle.Render(fpView)

	lefColumnRender := lipgloss.JoinVertical(lipgloss.Top, messageViewportRender, filePickerRender, m.renderSettings.View())

	renderViewStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder())
	renderViewRender := renderViewStyle.Render(m.renderView.View())

	return lipgloss.JoinHorizontal(lipgloss.Left, lefColumnRender, renderViewRender)
}

func normalizeRenderOptionsForService(settingsValues []ui.SettingItem) services.RenderOptions {
	var textSize int
	var fontAspect, edgeThreshold float64
	var directionalRender, reverseChars, highContrast bool
	var runeMode string

	for _, item := range settingsValues {
		switch item.Key {
		case "textSize":
			textSize, _ = strconv.Atoi(item.Value)

		case "fontAspect":
			fontAspect, _ = strconv.ParseFloat(item.Value, 2)

		case "edgeThreshold":
			edgeThreshold, _ = strconv.ParseFloat(item.Value, 2)

		case "directionalRender":
			directionalRender, _ = strconv.ParseBool(item.Value)

		case "reverseChars":
			reverseChars, _ = strconv.ParseBool(item.Value)

		case "highContrast":
			highContrast, _ = strconv.ParseBool(item.Value)

		case "runeMode":
			runeMode = item.Value
		}
	}
	options, err := services.NewRenderOptions(textSize, fontAspect, directionalRender, edgeThreshold, reverseChars, highContrast, runeMode)
	if err != nil {
		//TODO render Error and go back to renderOptionsMenu
	}
	return options
}
