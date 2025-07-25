package main

import (
	"flag"
	"fmt"
	"os"

	"bellbird-notes/app"
	"bellbird-notes/tui"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
	// parse flags for stuff like --debug etc.
	flag.Parse()

	if *app.ShowVersion {
		app.PrintVersion()
		return
	}

	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
