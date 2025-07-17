package message

import (
	"image/color"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	statusbarcolumn "bellbird-notes/tui/types/statusbar_column"
)

type Type int

const (
	Success Type = iota
	Error
	Prompt
	PromptError
)

var msgColours = map[Type]color.Color{
	Success:     lipgloss.NoColor{},
	Error:       lipgloss.Color("#d75a7d"),
	Prompt:      lipgloss.NoColor{},
	PromptError: lipgloss.Color("#d75a7d"),
}

func (m Type) Colour() color.Color {
	return msgColours[m]
}

type Sender int

const (
	SenderDirTree Sender = iota
	SenderNotesList
)

type StatusBarMsg struct {
	Content string
	Type    Type
	Sender  Sender
	Arg     any
	Cmd     tea.Cmd
	Column  statusbarcolumn.Column
}
