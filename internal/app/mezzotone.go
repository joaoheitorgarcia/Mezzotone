package app

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joaoheitorgarcia/Mezzotone/internal/export"
	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
	"github.com/joaoheitorgarcia/Mezzotone/internal/services"
	"github.com/joaoheitorgarcia/Mezzotone/internal/termtext"
	"github.com/joaoheitorgarcia/Mezzotone/internal/ui"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"
	"golang.design/x/clipboard"
)

type menuPhase int

const (
	filePickerMenu menuPhase = iota
	renderOptionsMenu
	renderView
)

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
	white    color.Color
	primary  color.Color
	selected color.Color
	gray     color.Color
	black    color.Color
	error    color.Color
}

var renderSettingsItemsSize int
var currentMessage string

var clipboardOK bool
var clipboardWrite = func(format clipboard.Format, data []byte) (<-chan struct{}, error) {
	return clipboard.Write(context.Background(), format, data)
}
var clipboardCommands = [][]string{
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
}

var newUUID = uuid.New

type MezzotoneModelConfig struct {
	ExportFontTTFPath string
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
		m.updateMessageTextOnMenuChange()
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

		m.messageViewPort.SetWidth(max(1, m.style.leftColumnWidth-2))

		m.renderView.SetHeight(m.height - m.style.windowMargin)

		computedFilePickerHeight := m.renderView.Height() -
			(renderSettingsItemsSize + 4) - //renderSettings header and end
			(m.messageViewPort.Height() + 2) - //message render view
			(m.style.windowMargin + 3) //inputFile Title

		m.filePicker.SetHeight(computedFilePickerHeight)

		m.toggleRenderViewFullscreen()
		m.updateMessageViewPortContent(currentMessage, false)

		return m, nil

	case tea.KeyMsg:
		if m.style.isRenderViewFullscreen && msg.String() != "f" && msg.String() != "ctrl+c" {
			return m, nil
		}
		if m.currentActiveMenu == filePickerMenu && m.isQuitting && msg.String() != "esc" {
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
					FontTTFPath:  m.exportFontTTFPath,
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
				return m, exportAsciiToPng(outPath, render, exportOptions)
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
					FontTTFPath:  m.exportFontTTFPath,
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
				return m, exportAsciiToGif(outPath, gifFrames, exportOptions)
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
				//confirm and process render
				if !m.renderSettings.Editing && m.renderSettings.IsConfirmSelected() {
					m.incrementCurrentActiveMenu()

					normalizedOptions, err := normalizeRenderOptionsForService(m.renderSettings.Items)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
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
						var gifDelaysDuration []time.Duration
						for i, frame := range frameArray {
							runeArray, colorArray, err := services.ConvertImageToString(frame, normalizedOptions)
							if err != nil {
								m.updateMessageViewPortContent("⚠ "+err.Error(), true)
								return m, cmd
							}
							gifRuneArrays = append(gifRuneArrays, runeArray)
							gifColorArrays = append(gifColorArrays, colorArray)

							gifDelaysDuration = append(gifDelaysDuration, time.Duration(delays[i])*10*time.Millisecond)
						}
						m.renderedGifOutput.renderedRunes = gifRuneArrays
						m.renderedGifOutput.renderedColor = gifColorArrays
						m.renderedGifOutput.delayTimes = gifDelaysDuration

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

				//update render settings items from render Moder
				currItem, ok := m.renderSettings.CurrentItem()
				if ok && !m.renderSettings.Editing && currItem.Key == "renderMode" {
					m.currentRenderMode = global.NextRenderMode(currItem.Value)
					items, err := renderSettingsItemsFromRenderMode(m.currentRenderMode)
					m.renderSettings.Items = items
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
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
				return m, cmd
			}
			if m.currentActiveMenu == renderView {
				m.renderView.PageDown()
				return m, cmd
			}
		case "pgup":
			if m.currentActiveMenu == renderOptionsMenu {
				m.renderSettings.SetActive(0)
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

func (m *MezzotoneModel) View() tea.View {
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
		v := tea.NewView(m.style.renderViewStyle.Render(m.renderView.View()))
		v.AltScreen = true
		return v
	}

	innerW := m.style.leftColumnWidth - 2
	messageViewportRender := m.style.messageViewStyle.renderStyle.Width(m.style.leftColumnWidth).Render(m.messageViewPort.View())

	fpView := termtext.TruncateLinesANSI(m.filePicker.View(), innerW)
	filePickerRender := m.style.filePickerStyle.renderStyle.Width(m.style.leftColumnWidth).Render(fpView)

	renderSettingsRender := m.style.renderSettingsStyle.renderStyle.Width(m.style.leftColumnWidth).Render(m.renderSettings.View())

	lefColumnRender := lipgloss.JoinVertical(lipgloss.Top, messageViewportRender, filePickerRender, renderSettingsRender)

	renderViewRender := m.style.renderViewStyle.Render(m.renderView.View())

	v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Left, lefColumnRender, renderViewRender))
	v.AltScreen = true
	return v
}
