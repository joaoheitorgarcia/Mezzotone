package components

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

type ClearErrorMsg struct{} //FIXME this maybe could be separated if needed on other components

type FileInput struct {
	FilePicker   filepicker.Model
	SelectedFile string
	Err          error
}

func NewFileInput(height int, allowedTypes []string) FileInput {
	fp := filepicker.New()

	fp.AllowedTypes = allowedTypes
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.SetHeight(height)

	return FileInput{
		FilePicker: fp,
	}
}

func (fi *FileInput) ClearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return ClearErrorMsg{}
	})
}
