package screens

import (
	"errors"
	"strings"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/services"
	"codeberg.org/JoaoGarcia/Mezzotone/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
)

type ImagePreview struct {
	loadingAnimation components.LoadingScreen
	loading          bool
	err              error
	outputArray      [][]rune
}

type ConvertDoneMsg struct {
	outputArray [][]rune
	Err         error
}

func NewImagePreview() ImagePreview {
	//TODO experiment with animation - change later
	loadingAnimation := components.NewLoadingScreen(
		[]string{
			"      \n      \n      \n      \n      \n      ",
			"o     \n      \n      \n      \n      \n      ",
			"oo    \no     \n      \n      \n      \n      ",
			"oooo  \noo    \no     \n      \n      \n      ",
			"ooooo \nooo   \noo    \no     \n      \n      ",
			"oooooo\noooo  \nooo   \noo    \no     \n      ",
			" ooooo\nooooo \noooo  \nooo   \noo    \no     ",
			"  oooo\noooooo\nooooo \noooo  \nooo   \noo    ",
			"   ooo\n ooooo\noooooo\nooooo \noooo  \nooo   ",
			"    oo\n  oooo\n ooooo\noooooo\nooooo \noooo  ",
			"     o\n   ooo\n  oooo\n ooooo\noooooo\nooooo ",
			"      \n    oo\n   ooo\n  oooo\n ooooo\noooooo",
			"      \n     o\n    oo\n   ooo\n  oooo\n ooooo",
			"      \n      \n      \n    oo\n   ooo\n  oooo",
			"      \n      \n      \n      \n    oo\n   ooo",
			"      \n      \n      \n      \n      \n    oo",
			"      \n      \n      \n      \n      \n     o",
			"      \n      \n      \n      \n      \n      ",
		},
		20,
	)

	return ImagePreview{
		loadingAnimation: loadingAnimation,
		loading:          true,
	}
}

func (m ImagePreview) Init() tea.Cmd {
	return tea.Batch(
		m.loadingAnimation.Spinner.Tick,
		convertImageCmd(),
	)
}

func (m ImagePreview) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {

	case ConvertDoneMsg:
		m.loading = false
		m.err = msg.Err
		if msg.Err != nil {
			_ = services.Logger().Error(msg.Err.Error())
			return m, nil
		}
		m.outputArray = msg.outputArray
		return m, nil

	default:
		if m.loading {
			var cmd tea.Cmd
			m.loadingAnimation.Spinner, cmd = m.loadingAnimation.Spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

func (m ImagePreview) View() string {
	if m.err != nil {
		return "Conversion failed:\n" + m.err.Error() + "\n"
	}
	if m.loading {
		return m.loadingAnimation.Spinner.View()
	}

	var outputBuilder strings.Builder
	for col := 0; col < len(m.outputArray); col++ {
		for row := 0; row < len(m.outputArray[col]); row++ {
			outputBuilder.WriteRune(m.outputArray[col][row])
		}
		outputBuilder.WriteByte('\n')
	}

	return outputBuilder.String()
}

func convertImageCmd() tea.Cmd {
	return func() tea.Msg {
		selectedFileAny, ok := services.Shared().Get("selectedFile")
		if !ok || selectedFileAny == nil {
			return ConvertDoneMsg{Err: errors.New("selectedFile not set")}
		}

		selectedFile, ok := selectedFileAny.(string)
		if !ok || selectedFile == "" {
			return ConvertDoneMsg{Err: errors.New("selectedFile is not a string")}
		}

		//TODO: get this from user input
		services.Shared().Set("textSize", 10)
		services.Shared().Set("fontAspect", 2.3)
		services.Shared().Set("useUnicode", true)
		services.Shared().Set("directionalRender", true)
		services.Shared().Set("edgeThresholdPercentile", 0.6)
		services.Shared().Set("reverseChars", true)
		services.Shared().Set("highContrast", false)

		convertedImage, err := services.ConvertImageToString(selectedFile)
		if err != nil {
			return ConvertDoneMsg{
				Err: err,
			}
		}

		return ConvertDoneMsg{
			outputArray: convertedImage,
		}
	}
}
