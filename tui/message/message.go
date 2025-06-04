package message

import (
	"github.com/charmbracelet/lipgloss"

	statusbarcolumn "bellbird-notes/tui/types/statusbar_column"
)

type Type int

const (
	Success Type = iota
	Error
	Prompt
	PromptError
)

var msgColours = map[Type]lipgloss.TerminalColor{
	Success:     lipgloss.NoColor{},
	Error:       lipgloss.Color("#d75a7d"),
	Prompt:      lipgloss.NoColor{},
	PromptError: lipgloss.Color("#d75a7d"),
}

func (m Type) Colour() lipgloss.TerminalColor {
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
	Column  statusbarcolumn.Column
}
