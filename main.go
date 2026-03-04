package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joaoheitorgarcia/Mezzotone/internal/app"
	"github.com/joaoheitorgarcia/Mezzotone/internal/services"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	fontTTF := flag.String("font-ttf", "", "path to a .ttf font used for image/gif export rendering")
	flag.Parse()
	if *debug {
		err := services.InitLogger("logs.log")
		if err != nil {
			return
		}
	}

	p := tea.NewProgram(app.NewMezzotoneModelWithConfig(app.MezzotoneModelConfig{
		ExportFontTTFPath: *fontTTF,
	}), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		_ = services.Logger().Error("Unexpected Error. Unable to recover")
		fmt.Printf("An unexpected error has occurred.\n")
		os.Exit(1)
	}
}
