package app

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/JoaoGarcia/Mezzotone/internal/export"
	"github.com/JoaoGarcia/Mezzotone/internal/services"
	"github.com/JoaoGarcia/Mezzotone/internal/termtext"
	"github.com/JoaoGarcia/Mezzotone/internal/ui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"golang.design/x/clipboard"
)

// TODO REORDER Layout IF TERMINAL width < height
// FIXME for some fontsize image gets cut on right/bottom - DONE
// TODO add fullscreen to renderview - DONE
// TODO add color toggle (current is no color)
// todo make it ⭐prettier⭐

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
	asciiGIFFrames    []ui.AnimationFrame

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

type styleVariables struct {
	windowMargin           int
	leftColumnWidth        int
	isRenderViewFullscreen bool
}

var renderSettingsItemsSize int
var clipboardOK bool
var newUUID = uuid.New

const (
	filePickerMenu = iota
	renderOptionsMenu
	renderView
)

func NewMezzotoneModel() *MezzotoneModel {
	windowStyles := styleVariables{
		windowMargin:           2,
		leftColumnWidth:        10,
		isRenderViewFullscreen: false,
	}

	runeMode := []string{"ASCII", "UNICODE", "DOTS", "RECTANGLES", "BARS"}
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

	renderView := viewport.New(0, 0)
	leftColumn := viewport.New(0, 0)

	messageViewPort := viewport.New(0, 3)

	model := &MezzotoneModel{
		filePicker:        fp,
		renderView:        renderView,
		messageViewPort:   messageViewPort,
		style:             windowStyles,
		leftColumn:        leftColumn,
		renderSettings:    renderSettingsModel,
		currentActiveMenu: filePickerMenu,
		helpPreviousMenu:  filePickerMenu,
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
		switch msg.String() {
		case "c":
			if m.currentActiveMenu == renderView {
				if !clipboardOK {
					m.updateMessageViewPortContent("Clipboard not available (init failed)", true)
					return m, nil
				}

				clipboard.Write(clipboard.FmtText, []byte(m.renderContent))
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
				outPpath := filepath.Join(homeDir, "Mezzotone_"+generatedUuid.String()+".txt")

				err = export.ASCIItToTxT(outPpath, m.renderContent)
				if err != nil {
					m.updateMessageViewPortContent("⚠ "+err.Error(), true)
					return m, nil
				}

				m.updateMessageViewPortContent("Successfully exported to "+outPpath+" !", false)
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
				}
				m.updateMessageViewPortContent("Exporting image to "+outPath+" ...", false)
				return m, exportAsciiToPngCmd(outPath, m.renderContent, exportOptions)
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
				}

				gifFrames := make([]export.ASCIIGIFFrame, 0, len(m.asciiGIFFrames))
				for _, frame := range m.asciiGIFFrames {
					gifFrames = append(gifFrames, export.ASCIIGIFFrame{
						ASCII:    frame.Frame,
						Duration: frame.Duration,
					})
				}
				m.updateMessageViewPortContent("Exporting gif to "+outPath+" ...", false)
				return m, exportAsciiToGifCmd(outPath, m.renderContent, gifFrames, exportOptions)
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
						for _, frame := range frameArray {
							runeArray, err := services.ConvertImageToString(frame, normalizedOptions)
							if err != nil {
								m.updateMessageViewPortContent("⚠ "+err.Error(), true)
								return m, cmd
							}
							gifRuneArrays = append(gifRuneArrays, runeArray)
						}

						var animationFrames []ui.AnimationFrame
						for i, frameRuneArray := range gifRuneArrays {
							frameASCII := services.ImageRuneArrayIntoString(frameRuneArray)
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
						m.asciiGIFFrames = animationFrames

						return m, m.gifAnimation.StartAnimation
					}

					// else is Image
					inputImg, format, err := image.Decode(f)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
					}
					_ = services.Logger().Info(fmt.Sprintf("format: %s", format))

					runeArray, err := services.ConvertImageToString(inputImg, normalizedOptions)
					if err != nil {
						m.updateMessageViewPortContent("⚠ "+err.Error(), true)
						return m, cmd
					}

					m.gifAnimation.StopAnimation()
					m.asciiGIFFrames = nil
					m.renderContent = services.ImageRuneArrayIntoString(runeArray)
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

	if m.style.isRenderViewFullscreen {
		renderViewStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder())
		return renderViewStyle.Render(m.renderView.View())
	}

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
		return services.RenderOptions{}, err
	}
	return options, nil
}

func (m *MezzotoneModel) incrementCurrentActiveMenu() {
	m.currentActiveMenu++

	var messageViewContent string
	switch m.currentActiveMenu {
	case filePickerMenu:
		messageViewContent = "Select image or gif to convert:"
		break
	case renderOptionsMenu:
		messageViewContent = "Edit render options and confirm:"
		break
	case renderView:
		messageViewContent = "See export keybindings With h"
		break
	}

	m.messageViewPort.SetContent(
		termtext.TruncateLinesANSI(
			messageViewContent+lipgloss.NewStyle().Faint(true).Render("\nPress h to toggle Help. Press esc to Quit."),
			m.style.leftColumnWidth,
		),
	)
}

func (m *MezzotoneModel) decrementCurrentActiveMenu() {
	m.currentActiveMenu--

	var messageViewContent string
	switch m.currentActiveMenu {
	case filePickerMenu:
		messageViewContent = "Select image gif or video to convert:"
		break
	case renderOptionsMenu:
		messageViewContent = "Edit render options and confirm:"
		break
	case renderView:
		messageViewContent = "Rendered image"
		break
	}

	m.messageViewPort.SetContent(
		termtext.TruncateLinesANSI(
			messageViewContent+lipgloss.NewStyle().Faint(true).Render("\nPress h to toggle Help. Press esc to Quit."),
			m.style.leftColumnWidth,
		),
	)
}

func (m *MezzotoneModel) updateMessageViewPortContent(messageViewContent string, isError bool) {
	if isError {
		messageViewContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(messageViewContent)
	}

	m.messageViewPort.SetContent(
		termtext.TruncateLinesANSI(
			messageViewContent+lipgloss.NewStyle().Faint(true).Render("\nPress h to toggle Help. Press esc to Quit."),
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

// SplitAnimatedGIF decodes an animated GIF and returns frames plus per-frame delays.
// GIF frames are often partial/offset “patches”, so we simulate playback by drawing each frame onto a
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

func exportAsciiToGifCmd(outPath, renderContent string, frames []export.ASCIIGIFFrame, exportOptions export.ASCIIExportOptions) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				msg = gifExportDoneMsg{
					outPath: outPath,
					err:     fmt.Errorf("gif export panic: %v", rec),
				}
			}
		}()

		var err error
		if len(frames) == 0 {
			frames = []export.ASCIIGIFFrame{
				{
					ASCII:    renderContent,
					Duration: 100 * time.Millisecond,
				},
			}
		}
		err = export.ASCIIFramesToGIF(frames, outPath, exportOptions)

		msg = gifExportDoneMsg{
			outPath: outPath,
			err:     err,
		}
		return msg
	}
}

func exportAsciiToPngCmd(outPath, renderContent string, exportOptions export.ASCIIExportOptions) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				msg = pngExportDoneMsg{
					outPath: outPath,
					err:     fmt.Errorf("png export panic: %v", rec),
				}
			}
		}()

		err := export.ASCIIToPNG(renderContent, outPath, exportOptions)
		msg = pngExportDoneMsg{
			outPath: outPath,
			err:     err,
		}
		return msg
	}
}

func (m *MezzotoneModel) toggleRenderViewFullscreen() {
	if m.style.isRenderViewFullscreen {
		m.renderView.Width = m.width - m.style.windowMargin
	} else {
		m.renderView.Width = m.width / 7 * 5
	}
}
