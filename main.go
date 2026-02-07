package main

import (
	"flag"
	"fmt"
	"os"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/app"
	"codeberg.org/JoaoGarcia/Mezzotone/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	if *debug {
		err := services.InitLogger("logs.log")
		if err != nil {
			return
		}
	}

	p := tea.NewProgram(app.NewMezzotoneModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		_ = services.Logger().Error("Unexpected Error. Unable to recover")
		fmt.Printf("An unexpected error has occurred.\n")
		os.Exit(1)
	}
}
