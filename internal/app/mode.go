package app

import "github.com/charmbracelet/lipgloss"

type Mode int

const (
	NormalMode Mode = iota
	InsertMode
	VisualMode
	VisualLineMode
	VisualBlockMode
	ReplaceMode
	OperatorMode
	CommandMode
)

var modeName = map[Mode]string{
	NormalMode:      "n",
	InsertMode:      "i",
	VisualMode:      "v",
	VisualLineMode:  "vi",
	VisualBlockMode: "vb",
	OperatorMode:    "o",
	ReplaceMode:     "r",
	CommandMode:     "c",
}

var fullName = map[Mode]string{
	NormalMode:      "NORMAL",
	InsertMode:      "INSERT",
	VisualMode:      "VISUAL",
	VisualLineMode:  "VISUAL LINE",
	VisualBlockMode: "VISUAL BLOCK",
	ReplaceMode:     "REPLACE",
	OperatorMode:    "",
	CommandMode:     "",
}

var colour = map[Mode]lipgloss.TerminalColor{
	NormalMode:      lipgloss.NoColor{},
	InsertMode:      lipgloss.Color("#7bb791"),
	VisualMode:      lipgloss.Color("#b7b27b"),
	VisualLineMode:  lipgloss.Color("#b7b27b"),
	VisualBlockMode: lipgloss.Color("#b7b27b"),
	ReplaceMode:     lipgloss.Color("#9e84b7"),
	OperatorMode:    lipgloss.NoColor{},
	CommandMode:     lipgloss.NoColor{},
}

func (m Mode) String() string {
	return modeName[m]
}

func (m Mode) FullString() string {
	return fullName[m]
}

func (m Mode) Colour() lipgloss.TerminalColor {
	return colour[m]
}

type ModeInstance struct {
	Current Mode
}

func (m ModeInstance) GetCurrent() Mode {
	return m.Current
}
