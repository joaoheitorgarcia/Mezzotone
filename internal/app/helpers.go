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
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/joaoheitorgarcia/Mezzotone/internal/export"
	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
	"github.com/joaoheitorgarcia/Mezzotone/internal/services"
	"github.com/joaoheitorgarcia/Mezzotone/internal/termtext"
	"github.com/joaoheitorgarcia/Mezzotone/internal/ui"
	"golang.design/x/clipboard"
)

func normalizeRenderOptionsForService(settingsValues []ui.SettingItem) (services.RenderOptions, error) {
	var textSize int
	var fontAspect float64
	var renderColor bool
	var renderMode string

	var edgeThreshold float64
	var directionalRender, reverseChars, highContrast bool
	var currentRuneMode string

	var density float64
	var dithering bool

	for _, item := range settingsValues {
		switch item.Key {
		case "renderMode":
			renderMode = item.Value
		case "textSize":
			textSize, _ = strconv.Atoi(item.Value)
		case "fontAspect":
			fontAspect, _ = strconv.ParseFloat(item.Value, 2)
		case "renderColor":
			renderColor, _ = strconv.ParseBool(item.Value)

		case "edgeThreshold":
			edgeThreshold, _ = strconv.ParseFloat(item.Value, 2)
		case "directionalRender":
			directionalRender, _ = strconv.ParseBool(item.Value)
		case "reverseChars":
			reverseChars, _ = strconv.ParseBool(item.Value)
		case "highContrast":
			highContrast, _ = strconv.ParseBool(item.Value)
		case "runeMode":
			currentRuneMode = item.Value

		case "density":
			density, _ = strconv.ParseFloat(item.Value, 2)
		case "dithering":
			dithering, _ = strconv.ParseBool(item.Value)
		}
	}
	options, err := services.NewRenderOptions(renderMode, textSize, fontAspect, renderColor, directionalRender, edgeThreshold, reverseChars, highContrast, currentRuneMode, density, dithering)
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
	currentMessage = messageViewContent

	if isError {
		messageViewContent = m.style.messageViewStyle.errorStyle.Render(messageViewContent)
	} else {
		messageViewContent = m.style.messageViewStyle.messageStyle.Render(messageViewContent)
	}

	m.messageViewPort.SetContent(
		termtext.TruncateLinesANSI(
			lipgloss.JoinVertical(lipgloss.Top, messageViewContent, m.style.messageViewStyle.helpStyle.Render("\nPress h to toggle Help. Press esc to Quit.")),
			max(1, m.style.leftColumnWidth-2),
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

func exportAsciiToGif(outPath string, frames []export.ASCIIGIFFrame, exportOptions export.ASCIIExportOptions) tea.Cmd {
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

func exportAsciiToPng(outPath string, imgOutput renderedImgOutput, exportOptions export.ASCIIExportOptions) tea.Cmd {
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
		m.renderView.SetWidth(m.width - m.style.windowMargin)
	} else {
		m.renderView.SetWidth(m.width / 7 * 5)
	}
}

func renderSettingsItemsFromRenderMode(renderModeType string) ([]ui.SettingItem, error) {
	var renderSettingsItems []ui.SettingItem

	switch renderModeType {

	case global.CHARACTER_RAMP:
		renderSettingsItems = append(
			renderSettingsItems,
			ui.SettingItem{Label: "Render Mode", Key: "renderMode", Type: ui.TypeEnum, Value: global.CHARACTER_RAMP, Enum: global.AvailableRenderModes()},
			ui.SettingItem{Label: "Text Size", Key: "textSize", Type: ui.TypeInt, Value: "10"},
			ui.SettingItem{Label: "Font Aspect", Key: "fontAspect", Type: ui.TypeFloat, Value: "2.3"},
			ui.SettingItem{Label: "Directional Render", Key: "directionalRender", Type: ui.TypeBool, Value: "FALSE"},
			ui.SettingItem{Label: "Edge Threshold", Key: "edgeThreshold", Type: ui.TypeFloat, Value: "0.6"},
			ui.SettingItem{Label: "Reverse Chars", Key: "reverseChars", Type: ui.TypeBool, Value: "TRUE"},
			ui.SettingItem{Label: "High Contrast", Key: "highContrast", Type: ui.TypeBool, Value: "TRUE"},
			ui.SettingItem{Label: "Render Color", Key: "renderColor", Type: ui.TypeBool, Value: "FALSE"},
			ui.SettingItem{Label: "Rune Mode", Key: "runeMode", Type: ui.TypeEnum, Value: "ASCII", Enum: global.AvailableRuneModes()},
		)

	case global.BRAILLE:
		renderSettingsItems = append(
			renderSettingsItems,
			ui.SettingItem{Label: "Render Mode", Key: "renderMode", Type: ui.TypeEnum, Value: global.BRAILLE, Enum: global.AvailableRenderModes()},
			ui.SettingItem{Label: "Text Size", Key: "textSize", Type: ui.TypeInt, Value: "10"},
			ui.SettingItem{Label: "Font Aspect", Key: "fontAspect", Type: ui.TypeFloat, Value: "2.3"},
			ui.SettingItem{Label: "Density", Key: "density", Type: ui.TypeFloat, Value: "0.5"},
			ui.SettingItem{Label: "Dithering", Key: "dithering", Type: ui.TypeBool, Value: "TRUE"},
			ui.SettingItem{Label: "Render Color", Key: "renderColor", Type: ui.TypeBool, Value: "FALSE"},
		)

	default:
		return renderSettingsItems, fmt.Errorf("unsupported render mode: %q", renderModeType)

	}

	return renderSettingsItems, nil
}
