package app

import (
	"image/color"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
	"github.com/joaoheitorgarcia/Mezzotone/internal/ui"
	"golang.design/x/clipboard"
)

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

type renderedImgOutput struct {
	renderedRunes [][]rune
	renderedColor [][]color.NRGBA
}

type renderedGifOutput struct {
	renderedRunes [][][]rune
	renderedColor [][][]color.NRGBA
	delayTimes    []time.Duration
}

type MezzotoneModelConfig struct {
	ExportFontTTFPath string
}

var renderSettingsItemsSize int
var renderSettingsItemsMaxSize int

type MezzotoneModel struct {
	filePicker   filepicker.Model
	selectedFile string

	renderView      viewport.Model
	leftColumn      viewport.Model
	renderSettings  ui.SettingsPanel
	messageViewPort viewport.Model

	style styleVariables

	currentActiveMenu menuPhase
	currentRenderMode string
	helpVisible       bool
	helpPreviousMenu  menuPhase
	isQuitting        bool
	renderContent     string
	exportFontTTFPath string

	renderedImgOutput renderedImgOutput
	renderedGifOutput renderedGifOutput

	gifAnimation ui.AnimationRenderer

	width  int
	height int

	err error
}

func NewMezzotoneModel() *MezzotoneModel {
	return NewMezzotoneModelWithConfig(MezzotoneModelConfig{})
}

func NewMezzotoneModelWithConfig(config MezzotoneModelConfig) *MezzotoneModel {
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

	renderSettingsItems, err := renderSettingsItemsFromRenderMode(global.INITIAL_RENDER_MODE)

	if err != nil {
		panic(err)
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

	renderViewPort := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	leftColumn := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))

	messageViewPort := viewport.New(viewport.WithWidth(0), viewport.WithHeight(3))

	model := &MezzotoneModel{
		filePicker:        fp,
		renderView:        renderViewPort,
		messageViewPort:   messageViewPort,
		style:             windowStyles,
		leftColumn:        leftColumn,
		renderSettings:    renderSettingsModel,
		currentActiveMenu: filePickerMenu,
		currentRenderMode: global.INITIAL_RENDER_MODE,
		helpPreviousMenu:  filePickerMenu,
		isQuitting:        false,
		exportFontTTFPath: strings.TrimSpace(config.ExportFontTTFPath),
	}
	model.updateMessageViewPortContent("Select image or gif to convert:", false)

	if err = clipboard.Init(); err == nil {
		clipboardOK = true
	}

	return model
}
