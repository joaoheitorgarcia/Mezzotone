package app

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"Mezzotone/internal/export"
	"Mezzotone/internal/services"
	"Mezzotone/internal/termtext"
	"Mezzotone/internal/ui"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"golang.design/x/clipboard"
)

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
	isQuitting        bool
	renderContent     string

	renderedImgOutput renderedImgOutput
	renderedGifOutput renderedGifOutput

	gifAnimation ui.AnimationRenderer

	width  int
	height int

	err error
}

type gifExportDoneMsg struct {
	outPath string
	err     error
}

type pngExportDoneMsg struct {
	outPath string
	err     error
}

type renderedImgOutput struct {
	renderedRunes [][]rune
	renderedColor [][]color.NRGBA
}

type renderedGifOutput struct {
	renderedRunes [][][]rune
	renderedColor [][][]color.NRGBA
	delayTimes    []time.Duration
}

type styleVariables struct {
	windowMargin           int
	leftColumnWidth        int
	isRenderViewFullscreen bool

	styleColors styleColors

	renderViewStyle     lipgloss.Style
	filePickerStyle     filePickerStyle
	renderSettingsStyle renderSettingsStyle
	messageViewStyle    messageViewStyle
}

type filePickerStyle struct {
	renderStyle             lipgloss.Style
	filePickerActiveStyle   filepicker.Styles
	filePickerInactiveStyle filepicker.Styles
}

type renderSettingsStyle struct {
	renderStyle                lipgloss.Style
	settingsPanelActiveStyle   ui.RenderSettingsStyles
	settingsPanelInactiveStyle ui.RenderSettingsStyles
}

type messageViewStyle struct {
	renderStyle  lipgloss.Style
	messageStyle lipgloss.Style
	errorStyle   lipgloss.Style
	helpStyle    lipgloss.Style
}

type styleColors struct {
	white    lipgloss.Color
	primary  lipgloss.Color
	selected lipgloss.Color
	gray     lipgloss.Color
	black    lipgloss.Color
	error    lipgloss.Color
}

var renderSettingsItemsSize int

var clipboardOK bool
var clipboardWrite = clipboard.Write
var clipboardCommands = [][]string{
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
}

var newUUID = uuid.New

const (
	filePickerMenu = iota
	renderOptionsMenu
	renderView
)

func NewMezzotoneModel() *MezzotoneModel {
	modelStyleColors := styleColors{
		white:    lipgloss.Color("255"),
		primary:  lipgloss.Color("99"),
		selected: lipgloss.Color("213"),
		gray:     lipgloss.Color("247"),
		black:    lipgloss.Color("232"),
		error:    lipgloss.Color("9"),
	}

	renderViewStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder())

	messageViewStyles := messageViewStyle{
		renderStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()),
		messageStyle: lipgloss.NewStyle().Foreground(modelStyleColors.selected),
		errorStyle: lipgloss.NewStyle().
			Foreground(modelStyleColors.error),
		helpStyle: lipgloss.NewStyle().
			Faint(true),
	}

	noFilesFoundString := "Oops. No Files Found."
	filePickerStyles := filePickerStyle{
		renderStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()),
		filePickerActiveStyle: filepicker.Styles{
			DisabledCursor:   lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Cursor:           lipgloss.NewStyle().Foreground(modelStyleColors.selected),
			Symlink:          lipgloss.NewStyle().Foreground(modelStyleColors.primary),
			Directory:        lipgloss.NewStyle().Foreground(modelStyleColors.primary),
			File:             lipgloss.NewStyle().Foreground(modelStyleColors.white),
			DisabledFile:     lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			DisabledSelected: lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Permission:       lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Selected:         lipgloss.NewStyle().Foreground(modelStyleColors.selected).Bold(true).Reverse(true),
			FileSize:         lipgloss.NewStyle().Foreground(modelStyleColors.gray).Width(7).Align(lipgloss.Right),
			EmptyDirectory:   lipgloss.NewStyle().Foreground(modelStyleColors.gray).PaddingLeft(2).SetString(noFilesFoundString),
		},
		filePickerInactiveStyle: filepicker.Styles{
			DisabledCursor:   lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Cursor:           lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Symlink:          lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Directory:        lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			File:             lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			DisabledFile:     lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			DisabledSelected: lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Permission:       lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			Selected:         lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			FileSize:         lipgloss.NewStyle().Foreground(modelStyleColors.gray).Width(7).Align(lipgloss.Right),
			EmptyDirectory:   lipgloss.NewStyle().Foreground(modelStyleColors.gray).PaddingLeft(2).SetString(noFilesFoundString),
		},
	}

	renderSettingsStyles := renderSettingsStyle{
		renderStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			Padding(1, 2),
		settingsPanelActiveStyle: ui.RenderSettingsStyles{
			LabelStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.primary),
			ValueStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.white),
			SelectedStyle:   lipgloss.NewStyle().Background(modelStyleColors.selected).Foreground(modelStyleColors.black).Bold(true),
			TitleStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.selected).Bold(true),
			ConfirmBtnStyle: lipgloss.NewStyle().Foreground(modelStyleColors.selected).Bold(true),
		},
		settingsPanelInactiveStyle: ui.RenderSettingsStyles{
			LabelStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			ValueStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			SelectedStyle:   lipgloss.NewStyle().Foreground(modelStyleColors.gray).Reverse(true),
			TitleStyle:      lipgloss.NewStyle().Foreground(modelStyleColors.gray),
			ConfirmBtnStyle: lipgloss.NewStyle().Foreground(modelStyleColors.gray),
		},
	}

	windowStyles := styleVariables{
		windowMargin:           2,
		leftColumnWidth:        10,
		isRenderViewFullscreen: false,

		styleColors: modelStyleColors,

		renderViewStyle:     renderViewStyle,
		messageViewStyle:    messageViewStyles,
		filePickerStyle:     filePickerStyles,
		renderSettingsStyle: renderSettingsStyles,
	}

	runeMode := []string{"ASCII", "UNICODE", "DOTS", "RECTANGLES", "BARS"}
	renderSettingsItems := []ui.SettingItem{
		{Label: "Text Size", Key: "textSize", Type: ui.TypeInt, Value: "10"},
		{Label: "Font Aspect", Key: "fontAspect", Type: ui.TypeFloat, Value: "2.3"},
		{Label: "Directional Render", Key: "directionalRender", Type: ui.TypeBool, Value: "FALSE"},
		{Label: "Edge Threshold", Key: "edgeThreshold", Type: ui.TypeFloat, Value: "0.6"},
		{Label: "Reverse Chars", Key: "reverseChars", Type: ui.TypeBool, Value: "TRUE"},
		{Label: "High Contrast", Key: "highContrast", Type: ui.TypeBool, Value: "TRUE"},
		{Label: "Render Color", Key: "renderColor", Type: ui.TypeBool, Value: "FALSE"},
		{Label: "Rune Mode", Key: "runeMode", Type: ui.TypeEnum, Value: "ASCII", Enum: runeMode},
	}
	renderSettingsItemsSize = len(renderSettingsItems)
	renderSettingsModel := ui.NewSettingsPanel("Render Options", renderSettingsItems, windowStyles.renderSettingsStyle.settingsPanelInactiveStyle)
	renderSettingsModel.ClearActive()

	fp := filepicker.New()
	fp.AllowedTypes = []string{".png", ".jpg", ".jpeg", ".bmp", ".webp", ".tiff", ".gif"}
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
	fp.Styles = windowStyles.filePickerStyle.filePickerActiveStyle

	renderViewPort := viewport.New(0, 0)
	leftColumn := viewport.New(0, 0)

	messageViewPort := viewport.New(0, 3)

	model := &MezzotoneModel{
		filePicker:        fp,
		renderView:        renderViewPort,
		messageViewPort:   messageViewPort,
		style:             windowStyles,
		leftColumn:        leftColumn,
		renderSettings:    renderSettingsModel,
		currentActiveMenu: filePickerMenu,
		helpPreviousMenu:  filePickerMenu,
		isQuitting:        false,
	}
	model.updateMessageViewPortContent("Select image or gif to convert:", false)

	if err := clipboard.Init(); err == nil {
		clipboardOK = true
	}

	return model
}

func (m *MezzotoneModel) Init() tea.Cmd {
	return m.filePicker.Init()
}

func (m *MezzotoneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case gifExportDoneMsg:
		if msg.err != nil {
			m.updateMessageViewPortContent("⚠ "+msg.err.Error(), true)
			return m, nil
		}
		m.updateMessageViewPortContent("Successfully exported to "+msg.outPath+" !", false)
		return m, nil

	case pngExportDoneMsg:
		if msg.err != nil {
			m.updateMessageViewPortContent("⚠ "+msg.err.Error(), true)
			return m, nil
		}
		m.updateMessageViewPortContent("Successfully exported to "+msg.outPath+" !", false)
		return m, nil

	case ui.TickMsg:
		if !m.gifAnimation.IsAnimationPlaying() {
			return m, nil
		}
		var c tea.Cmd
		m.gifAnimation, c = m.gifAnimation.Update(msg)
		if !m.helpVisible {
			m.renderContent = m.gifAnimation.View()
			m.renderView.SetContent(m.renderContent)
		}
		return m, c

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		m.style.leftColumnWidth = m.width / 7 * 2

		m.renderSettings.SetWidth(m.style.leftColumnWidth)
		m.renderSettings.SetHeight(renderSettingsItemsSize)

		m.messageViewPort.Width = m.style.leftColumnWidth

		m.renderView.Height = m.height - m.style.windowMargin

		computedFilePickerHeight := m.renderView.Height -
			(renderSettingsItemsSize + 4) - //renderSettings header and end
			(m.messageViewPort.Height + 2) - //message render view
			(m.style.windowMargin + 3) //inputFile Title

		m.filePicker.SetHeight(computedFilePickerHeight)

		m.toggleRenderViewFullscreen()
		m.updateMessageViewPortContent("Select image or gif to convert:", false)

		return m, nil

	case tea.KeyMsg:
		if m.style.isRenderViewFullscreen && msg.String() != "f" && msg.String() != "ctrl+c" {
			return m, nil
		}
		if m.currentActiveMenu == filePickerMenu && m.isQuitting && msg.Type != tea.KeyEsc {
			m.isQuitting = false
			m.updateMessageViewPortContent("Select image or gif to convert:", false)
		}
		switch msg.String() {
		case "c":
			if m.currentActiveMenu == renderView {
				if err := copyTextToClipboard(m.renderContent); err != nil {
					m.updateMessageViewPortContent("⚠ "+err.Error(), true)
					return m, nil
				}
				m.updateMessageViewPortContent("Successfully sent to clipboard !", false)
				return m, nil
			}
		case "t":
			if m.currentActiveMenu == renderView {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					m.updateMessageViewPortContent("⚠ "+err.Error(), true)
					return m, nil
				}
				generatedUuid := newUUID()
				outPath := filepath.Join(homeDir, "Mezzotone_"+generatedUuid.String()+".txt")

				err = export.ASCIItToTxT(outPath, m.renderContent)
				if err != nil {
					m.updateMessageViewPortContent("⚠ "+err.Error(), true)
					return m, nil
				}

				m.updateMessageViewPortContent("Successfully exported to "+outPath+" !", false)
				return m, nil
			}
		case "i":
			if m.currentActiveMenu == renderView {
				homeDir, _ := os.UserHomeDir()
				generatedUuid := newUUID()
				outPath := filepath.Join(homeDir, "Mezzotone_"+generatedUuid.String()+".png")

				fontAspect := 1.0
				for i := range m.renderSettings.Items {
					if m.renderSettings.Items[i].Key == "fontAspect" {
						fontAspect, _ = strconv.ParseFloat(m.renderSettings.Items[i].Value, 2)
					}
				}

				// Font Aspect is height/width (2.3). Export wants width/height.
				targetAspect := 1.0 / fontAspect

				exportOptions := export.ASCIIExportOptions{
					FontSize:     14,
					DPI:          300,
					BG:           color.Black,
					FG:           color.White,
					TargetAspect: targetAspect,
					RenderColor:  m.getRenderColor(),
				}

				m.updateMessageViewPortContent("Exporting image to "+outPath+" ...", false)

				var render renderedImgOutput
				if m.renderedImgOutput.renderedRunes == nil {
					i := m.gifAnimation.GetcurrentFrameIndex()
					render = renderedImgOutput{
						renderedRunes: m.renderedGifOutput.renderedRunes[i],
						renderedColor: m.renderedGifOutput.renderedColor[i],
					}
				} else {
					render = m.renderedImgOutput
				}
				return m, exportAsciiToPngCmd(outPath, render, exportOptions)
			}
		case "g":
			if m.currentActiveMenu == renderView {
				homeDir, _ := os.UserHomeDir()
				generatedUuid := newUUID()
				outPath := filepath.Join(homeDir, "Mezzotone_"+generatedUuid.String()+".gif")

				fontAspect := 1.0
				for i := range m.renderSettings.Items {
					if m.renderSettings.Items[i].Key == "fontAspect" {
						fontAspect, _ = strconv.ParseFloat(m.renderSettings.Items[i].Value, 2)
					}
				}

				// Font Aspect is height/width (2.3). Export wants width/height.
				targetAspect := 1.0 / fontAspect

				exportOptions := export.ASCIIExportOptions{
					FontSize:     14,
					DPI:          300,
					BG:           color.Black,
					FG:           color.White,
					TargetAspect: targetAspect,
					RenderColor:  m.getRenderColor(),
				}

				gifFrames := make([]export.ASCIIGIFFrame, 0, len(m.renderedGifOutput.renderedRunes))
				for i := range m.renderedGifOutput.renderedRunes {
					gifFrames = append(gifFrames, export.ASCIIGIFFrame{
						FrameRunes:  m.renderedGifOutput.renderedRunes[i],
						Duration:    m.renderedGifOutput.delayTimes[i],
						FrameColors: m.renderedGifOutput.renderedColor[i],
					})
				}

				m.updateMessageViewPortContent("Exporting gif to "+outPath+" ...", false)
				return m, exportAsciiToGifCmd(outPath, gifFrames, exportOptions)
			}
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
			m.currentActiveMenu = renderView
			m.renderView.GotoTop()
			m.renderView.SetContent(buildRenderHelpText(m.style))
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
				if m.isQuitting {
					return m, tea.Quit
				}
				m.isQuitting = true
				m.updateMessageViewPortContent("Press esc again to quit", false)
				return m, nil
			}
			if m.currentActiveMenu == renderOptionsMenu {
				if !m.renderSettings.Editing {
					m.decrementCurrentActiveMenu()
					m.renderSettings.ClearActive()
				}
				return m, cmd
			}
			if m.currentActiveMenu == renderView {
				m.decrementCurrentActiveMenu()
				return m, cmd
			}
		case "enter":
			if m.currentActiveMenu == renderOptionsMenu {
				if !m.renderSettings.Editing && m.renderSettings.Confirm {
					m.incrementCurrentActiveMenu()

					normalizedOptions, err := normalizeRenderOptionsForService(m.renderSettings.Items)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
					}

					f, err := os.Open(m.selectedFile)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
					}
					defer func() { _ = f.Close() }()

					_ = services.Logger().Info(fmt.Sprintf("Successfully Loaded: %s", m.selectedFile))

					if IsGIF(m.selectedFile) {
						frameArray, delays, err := SplitAnimatedGIF(f)
						if err != nil {
							m.updateMessageViewPortContent("⚠ "+err.Error(), true)
							return m, cmd
						}
						var gifRuneArrays [][][]rune
						var gifColorArrays [][][]color.NRGBA
						for i, frame := range frameArray {
							runeArray, colorArray, err := services.ConvertImageToString(frame, normalizedOptions)
							if err != nil {
								m.updateMessageViewPortContent("⚠ "+err.Error(), true)
								return m, cmd
							}
							gifRuneArrays = append(gifRuneArrays, runeArray)
							gifColorArrays = append(gifColorArrays, colorArray)

							m.renderedGifOutput.renderedRunes = append(m.renderedGifOutput.renderedRunes, runeArray)
							m.renderedGifOutput.renderedColor = append(m.renderedGifOutput.renderedColor, colorArray)
							m.renderedGifOutput.delayTimes = append(m.renderedGifOutput.delayTimes, time.Duration(delays[i])*10*time.Millisecond)
						}

						var animationFrames []ui.AnimationFrame
						for i, frameRuneArray := range gifRuneArrays {
							frameASCII := services.ImageRuneArrayIntoString(frameRuneArray, gifColorArrays[i], normalizedOptions.RenderColor)
							animationFrames = append(
								animationFrames,
								ui.AnimationFrame{
									Frame:    frameASCII,
									Duration: time.Duration(delays[i]) * 10 * time.Millisecond,
								},
							)
						}
						_ = services.Logger().Info(fmt.Sprintf("%s", m.renderContent))

						var escapeKeys []string
						escapeKeys = append(escapeKeys, "esc")
						gifAnimation := ui.NewAnimationRenderer(animationFrames, escapeKeys)
						m.gifAnimation = gifAnimation

						m.renderedImgOutput.renderedRunes = nil
						m.renderedImgOutput.renderedColor = nil

						return m, m.gifAnimation.StartAnimation
					}

					// else is Image
					inputImg, format, err := image.Decode(f)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
					}
					_ = services.Logger().Info(fmt.Sprintf("format: %s", format))

					runeArray, colorArray, err := services.ConvertImageToString(inputImg, normalizedOptions)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
					}

					m.renderedImgOutput.renderedRunes = runeArray
					m.renderedImgOutput.renderedColor = colorArray

					m.gifAnimation.StopAnimation()

					m.renderContent = services.ImageRuneArrayIntoString(runeArray, colorArray, normalizedOptions.RenderColor)
					_ = services.Logger().Info(fmt.Sprintf("%s", m.renderContent))

					if !m.helpVisible {
						m.renderView.SetContent(m.renderContent)
					}
					return m, cmd
				}
			}
		case "left":
			if m.currentActiveMenu == renderView {
				m.renderView.ScrollLeft(1)
				return m, cmd
			}
		case "right":
			if m.currentActiveMenu == renderView {
				m.renderView.ScrollRight(1)
				return m, cmd
			}
		case "up":
			if m.currentActiveMenu == renderView {
				m.renderView.ScrollUp(1)
				return m, cmd
			}
		case "down":
			if m.currentActiveMenu == renderView {
				m.renderView.ScrollDown(1)
				return m, cmd
			}
		case "pgdown":
			if m.currentActiveMenu == renderOptionsMenu {
				m.renderSettings.SetActive(renderSettingsItemsSize)
				m.renderSettings.Confirm = true
				return m, cmd
			}
			if m.currentActiveMenu == renderView {
				m.renderView.PageDown()
				return m, cmd
			}
		case "pgup":
			if m.currentActiveMenu == renderOptionsMenu {
				m.renderSettings.SetActive(0)
				m.renderSettings.Confirm = false
				return m, cmd
			}
			if m.currentActiveMenu == renderView {
				m.renderView.PageUp()
				return m, cmd
			}
		case "shift+up":
			if m.currentActiveMenu == renderView {
				m.renderView.PageUp()
				return m, cmd
			}
		case "shift+down":
			if m.currentActiveMenu == renderView {
				m.renderView.PageDown()
				return m, cmd
			}
		case "shift+left":
			if m.currentActiveMenu == renderView {
				m.renderView.SetXOffset(0)
				return m, cmd
			}
		case "shift+right":
			if m.currentActiveMenu == renderView {
				m.renderView.SetXOffset(1 << 30)
				return m, cmd
			}
		case "f":
			if m.currentActiveMenu == renderView {
				m.style.isRenderViewFullscreen = !m.style.isRenderViewFullscreen
				m.toggleRenderViewFullscreen()
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
			m.incrementCurrentActiveMenu()
			return m, cmd
		}

		if didSelect, path := m.filePicker.DidSelectDisabledFile(msg); didSelect {
			m.updateMessageViewPortContent("⚠ Selected file not allowed", true)
			m.selectedFile = ""
			_ = services.Logger().Info(fmt.Sprintf("Tried Selecting File: %s", path))
			return m, cmd
		}
	}
	if m.currentActiveMenu == renderOptionsMenu {
		m.renderSettings, cmd = m.renderSettings.Update(msg)
		if errMsg := m.renderSettings.ErrorMessage(); errMsg != "" {
			m.updateMessageViewPortContent("⚠ "+errMsg, true)
		} else {
			m.updateMessageViewPortContent("Edit render options and confirm:", false)
		}
		return m, cmd
	}
	if m.currentActiveMenu == renderView {
		m.renderView, cmd = m.renderView.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m *MezzotoneModel) View() string {
	switch m.currentActiveMenu {
	case renderView:
		m.filePicker.Styles = m.style.filePickerStyle.filePickerInactiveStyle
		m.renderSettings.Styles = m.style.renderSettingsStyle.settingsPanelInactiveStyle
	case renderOptionsMenu:
		m.filePicker.Styles = m.style.filePickerStyle.filePickerInactiveStyle
		m.renderSettings.Styles = m.style.renderSettingsStyle.settingsPanelActiveStyle
	case filePickerMenu:
		m.filePicker.Styles = m.style.filePickerStyle.filePickerActiveStyle
		m.renderSettings.Styles = m.style.renderSettingsStyle.settingsPanelInactiveStyle
	}

	if m.style.isRenderViewFullscreen {
		return m.style.renderViewStyle.Render(m.renderView.View())
	}

	innerW := m.style.leftColumnWidth - 2
	messageViewportRender := m.style.messageViewStyle.renderStyle.Width(m.style.leftColumnWidth).Render(m.messageViewPort.View())

	fpView := termtext.TruncateLinesANSI(m.filePicker.View(), innerW)
	filePickerRender := m.style.filePickerStyle.renderStyle.Width(m.style.leftColumnWidth).Render(fpView)

	renderSettingsRender := m.style.renderSettingsStyle.renderStyle.Width(m.style.leftColumnWidth).Render(m.renderSettings.View())

	lefColumnRender := lipgloss.JoinVertical(lipgloss.Top, messageViewportRender, filePickerRender, renderSettingsRender)

	renderViewRender := m.style.renderViewStyle.Render(m.renderView.View())

	return lipgloss.JoinHorizontal(lipgloss.Left, lefColumnRender, renderViewRender)
}

func (m *MezzotoneModel) safeFilePickerView() (out string) {
	defer func() {
		if recover() != nil {
			out = "⚠ File picker view failed. Change directory or restart."
		}
	}()
	return m.filePicker.View()
}

func normalizeRenderOptionsForService(settingsValues []ui.SettingItem) (services.RenderOptions, error) {
	var textSize int
	var fontAspect, edgeThreshold float64
	var directionalRender, reverseChars, highContrast, renderColor bool
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
		case "renderColor":
			renderColor, _ = strconv.ParseBool(item.Value)
		case "runeMode":
			runeMode = item.Value
		}
	}
	options, err := services.NewRenderOptions(textSize, fontAspect, directionalRender, edgeThreshold, reverseChars, highContrast, renderColor, runeMode)
	if err != nil {
		return services.RenderOptions{}, err
	}
	return options, nil
}

func (m *MezzotoneModel) getRenderColor() bool {
	for _, item := range m.renderSettings.Items {
		if item.Key == "renderColor" {
			value, _ := strconv.ParseBool(item.Value)
			return value
		}
	}
	return false
}

func (m *MezzotoneModel) incrementCurrentActiveMenu() {
	m.currentActiveMenu++
	m.updateMessageTextOnMenuChange()
}

func (m *MezzotoneModel) decrementCurrentActiveMenu() {
	m.currentActiveMenu--
	m.updateMessageTextOnMenuChange()
}

func (m *MezzotoneModel) updateMessageTextOnMenuChange() {
	switch m.currentActiveMenu {
	case filePickerMenu:
		m.updateMessageViewPortContent("Select image or gif to convert:", false)
		break
	case renderOptionsMenu:
		m.updateMessageViewPortContent("Edit render options and confirm:", false)
		break
	case renderView:
		m.updateMessageViewPortContent("Press f for fullscreen, see export options with h", false)
		break
	}
}

func (m *MezzotoneModel) updateMessageViewPortContent(messageViewContent string, isError bool) {
	if isError {
		messageViewContent = m.style.messageViewStyle.errorStyle.Render(messageViewContent)
	} else {
		messageViewContent = m.style.messageViewStyle.messageStyle.Render(messageViewContent)
	}

	m.messageViewPort.SetContent(
		termtext.TruncateLinesANSI(
			lipgloss.JoinVertical(lipgloss.Top, messageViewContent, m.style.messageViewStyle.helpStyle.Render("\nPress h to toggle Help. Press esc to Quit.")),
			m.style.leftColumnWidth,
		),
	)
}

func IsGIF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	if err != nil {
		return false
	}

	return format == "gif"
}

// SplitAnimatedGIF decodes an animated GIF and returns frames plus per-frame delayTimes.
// GIF frames are often partial/offset “patches”, so playback is simulated by drawing each frame onto a
// full-size RGBA canvas and then clone the canvas after each draw so frames don’t share the same pixel buffer.
func SplitAnimatedGIF(r io.Reader) (frames []image.Image, delays []int, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic while decoding gif: %v", rec)
		}
	}()

	g, err := gif.DecodeAll(r)
	if err != nil {
		return nil, nil, err
	}
	if len(g.Image) == 0 {
		return nil, nil, fmt.Errorf("gif has no frames")
	}

	w, h := g.Config.Width, g.Config.Height
	canvasBounds := image.Rect(0, 0, w, h)
	canvas := image.NewRGBA(canvasBounds)

	bg := color.RGBA{}
	if len(g.Image[0].Palette) > 0 && int(g.BackgroundIndex) < len(g.Image[0].Palette) {
		r0, g0, b0, a0 := g.Image[0].Palette[g.BackgroundIndex].RGBA()
		bg = color.RGBA{R: uint8(r0 >> 8), G: uint8(g0 >> 8), B: uint8(b0 >> 8), A: uint8(a0 >> 8)}
	}
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	delays = make([]int, 0, len(g.Image))

	var prevCanvas *image.RGBA

	for i, src := range g.Image {
		// Save canvas BEFORE drawing this frame if disposal asks to restore previous
		if len(g.Disposal) > i && g.Disposal[i] == gif.DisposalPrevious {
			prevCanvas = cloneRGBA(canvas)
		} else {
			prevCanvas = nil
		}

		draw.Draw(canvas, src.Bounds(), src, src.Bounds().Min, draw.Over)
		frames = append(frames, cloneRGBA(canvas))

		if len(g.Delay) > i {
			delays = append(delays, g.Delay[i])
		} else {
			delays = append(delays, 0)
		}

		// Apply disposal for next frame
		if len(g.Disposal) > i {
			switch g.Disposal[i] {
			case gif.DisposalBackground:
				draw.Draw(canvas, src.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				if prevCanvas != nil {
					canvas = prevCanvas
				}
			}
		}
	}

	return frames, delays, nil
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func exportAsciiToGifCmd(outPath string, frames []export.ASCIIGIFFrame, exportOptions export.ASCIIExportOptions) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				msg = gifExportDoneMsg{
					outPath: outPath,
					err:     fmt.Errorf("gif export panic: %v", rec),
				}
			}
		}()

		if len(frames) == 0 {
			return gifExportDoneMsg{
				outPath: outPath,
				err:     fmt.Errorf("no rendered gif frames available to export"),
			}
		}

		err := export.ASCIIFramesToGIF(frames, outPath, exportOptions)

		msg = gifExportDoneMsg{
			outPath: outPath,
			err:     err,
		}
		return msg
	}
}

func exportAsciiToPngCmd(outPath string, imgOutput renderedImgOutput, exportOptions export.ASCIIExportOptions) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				msg = pngExportDoneMsg{
					outPath: outPath,
					err:     fmt.Errorf("png export panic: %v", rec),
				}
			}
		}()

		err := export.ASCIIToPNG(imgOutput.renderedRunes, imgOutput.renderedColor, outPath, exportOptions)
		msg = pngExportDoneMsg{
			outPath: outPath,
			err:     err,
		}
		return msg
	}
}

func copyTextToClipboard(content string) error {
	cleanContent := content
	if len(cleanContent) == 0 {
		return fmt.Errorf("nothing to copy (render output is empty)")
	}

	if clipboardOK {
		if changed := clipboardWrite(clipboard.FmtText, []byte(cleanContent)); changed != nil {
			return nil
		}
	}

	for _, command := range clipboardCommands {
		if len(command) == 0 {
			continue
		}
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdin = strings.NewReader(cleanContent)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("clipboard not available (init failed)")
}

func (m *MezzotoneModel) toggleRenderViewFullscreen() {
	if m.style.isRenderViewFullscreen {
		m.renderView.Width = m.width - m.style.windowMargin
	} else {
		m.renderView.Width = m.width / 7 * 5
	}
}
