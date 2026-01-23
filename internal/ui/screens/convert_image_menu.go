package screens

import (
	"errors"
	"strings"
	"time"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
)

type ConvertImageMenuScreen struct {
	fileInput components.FileInput
}

const FilePickerHeight = 10

func NewConvertImageMenuScreen() ConvertImageMenuScreen {
	fp := components.NewFileInput(
		FilePickerHeight,
		[]string{".png", ".jpg", ".jpeg", ".bmp", ".webp", ".tiff"},
	)

	return ConvertImageMenuScreen{
		fileInput: fp,
	}
}

func (m ConvertImageMenuScreen) Init() tea.Cmd {
	return m.fileInput.FilePicker.Init()
}

func (m ConvertImageMenuScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m.fileInput.FilePicker.SetHeight(FilePickerHeight - 2)
	case components.ClearErrorMsg:
		m.fileInput.Err = nil
	}

	var cmd tea.Cmd
	m.fileInput.FilePicker, cmd = m.fileInput.FilePicker.Update(msg)

	if didSelect, path := m.fileInput.FilePicker.DidSelectFile(msg); didSelect {
		m.fileInput.SelectedFile = path
	}

	if didSelect, _ := m.fileInput.FilePicker.DidSelectDisabledFile(msg); didSelect {
		m.fileInput.Err = errors.New("Selected file need to be an image.\nAllowed types: .png, .jpg, .jpeg, .bmp, .webp, .tiff")
		m.fileInput.SelectedFile = ""
		return m, tea.Batch(cmd, m.fileInput.ClearErrorAfter(5*time.Second))
	}

	return m, cmd
}

func (m ConvertImageMenuScreen) View() string {
	var s strings.Builder

	s.WriteString("\nCurrent Directory:  " + m.fileInput.FilePicker.CurrentDirectory)
	s.WriteString("\n\nPick a file:")

	s.WriteString("\n\n" + m.fileInput.FilePicker.View() + "\n")

	if m.fileInput.Err != nil {
		s.WriteString("\n" + m.fileInput.Err.Error() + "\n")
	}
	return s.String()
}
