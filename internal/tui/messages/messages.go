package messages

import "github.com/charmbracelet/lipgloss"

type MsgType int

const (
	Success MsgType = iota
	Error
	Prompt
	PromptError
)

var msgColours = map[MsgType]lipgloss.TerminalColor{
	Success:     lipgloss.NoColor{},
	Error:       lipgloss.Color("#d75a7d"),
	Prompt:      lipgloss.NoColor{},
	PromptError: lipgloss.Color("#d75a7d"),
}

func (m MsgType) Colour() lipgloss.TerminalColor {
	return msgColours[m]
}

type StatusBarMsg struct {
	Content string
	Type    MsgType
}
