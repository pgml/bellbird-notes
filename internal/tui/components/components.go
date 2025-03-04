package components

import (
	"bellbird-notes/internal/app"

	"github.com/charmbracelet/bubbles/viewport"
	bl "github.com/winder/bubblelayout"
)

type Component struct {
	Id   bl.ID
	Size bl.Size

	// The current mode the directory tree is in
	// Possible modes are Normal, Insert, Command
	Mode app.Mode

	// Indicates hether the directory tree column is focused.
	// Used to determine if the directory tree should receive keyboard shortcuts
	Focused bool

	statusMessage string         // For displaying useful information in the status bar
	viewport      viewport.Model // The tree viewport that allows scrolling
	ready         bool
}
