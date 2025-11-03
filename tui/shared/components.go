package shared

import (
	"github.com/charmbracelet/bubbles/v2/viewport"
	bl "github.com/winder/bubblelayout"

	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
)

type Component struct {
	ID   bl.ID
	Size bl.Size

	// The current mode the directory tree is in
	// Possible modes are Normal, Insert, Command
	Mode mode.Mode

	// For displaying useful information in the status bar
	statusMessage string
	Viewport      viewport.Model

	// Header is the title of the component
	header *string

	// IsReady indicates if the component has been initialized
	IsReady bool

	isVisible bool

	// Indicates whether the directory tree column is focused.
	// Used to determine if the directory tree should receive keyboard shortcuts
	isFocused bool

	OnFocus func()
	OnBlur  func()

	theme theme.Theme
}

func (c Component) Visible() bool {
	return c.isVisible
}

func (c *Component) Show() {
	c.isVisible = true
}

func (c *Component) Hide() {
	c.isVisible = false
}

func (c *Component) SetVisibility(visible bool) {
	if c.Visible() == visible {
		return
	}

	if visible {
		c.Show()
	} else {
		c.Hide()
	}
}

func (c *Component) ToggleVisibility() {
	if c.isVisible {
		c.Hide()
	} else {
		c.Show()
	}
}

func (c Component) Focused() bool {
	return c.isFocused
}

func (c *Component) Focus() {
	c.isFocused = true

	if c.OnFocus != nil {
		c.OnFocus()
	}
}

func (c *Component) Blur() {
	c.isFocused = false

	if c.OnBlur != nil {
		c.OnBlur()
	}
}

func (c *Component) ToggleFocus() {
	if c.isFocused {
		c.Blur()
	} else {
		c.Focus()
	}
}

func (c *Component) SetFocus(focus bool) {
	if c.Focused() == focus {
		return
	}

	if focus {
		c.Focus()
	} else {
		c.Blur()
	}
}

func (c Component) Theme() theme.Theme {
	return c.theme
}

func (c *Component) SetTheme(theme theme.Theme) {
	c.theme = theme
}
