package components

import (
	"github.com/charmbracelet/bubbles/v2/viewport"
	bl "github.com/winder/bubblelayout"

	"bellbird-notes/tui/mode"
)

type DeferredActionMsg struct{}

type Component struct {
	ID   bl.ID
	Size bl.Size

	// The current mode the directory tree is in
	// Possible modes are Normal, Insert, Command
	Mode mode.Mode

	// Indicates hether the directory tree column is focused.
	// Used to determine if the directory tree should receive keyboard shortcuts
	focused bool

	// For displaying useful information in the status bar
	statusMessage string
	viewport      viewport.Model

	// Header is the title of the component
	header *string

	// Ready indicates if the component has been initialized
	Ready bool
}

// Focused returns whether the component is focused
func (c *Component) Focused() bool {
	return c.focused
}

// SetFocus sets the focus state of the component
func (c *Component) SetFocus(focus bool) {
	c.focused = focus
}
