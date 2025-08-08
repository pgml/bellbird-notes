package mode

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

type Mode int

const (
	Normal Mode = iota
	Insert
	Visual
	VisualLine
	VisualBlock
	Replace
	Operator
	Command
	Search
	SearchPrompt
)

var modeName = map[Mode]string{
	Normal:       "n",
	Insert:       "i",
	Visual:       "v",
	VisualLine:   "vl",
	VisualBlock:  "vb",
	Replace:      "r",
	Operator:     "o",
	Command:      "c",
	Search:       "s",
	SearchPrompt: "sp",
}

var fullName = map[Mode]string{
	Normal:       "normal",
	Insert:       "insert",
	Visual:       "visual",
	VisualLine:   "visual_line",
	VisualBlock:  "visual_block",
	Replace:      "replace",
	Operator:     "",
	Command:      "command",
	Search:       "/",
	SearchPrompt: "search",
}

var colour = map[Mode]color.Color{
	Normal:      lipgloss.NoColor{},
	Insert:      lipgloss.Color("#7bb791"),
	Visual:      lipgloss.Color("#b7b27b"),
	VisualLine:  lipgloss.Color("#b7b27b"),
	VisualBlock: lipgloss.Color("#b7b27b"),
	Replace:     lipgloss.Color("#9e84b7"),
	Operator:    lipgloss.NoColor{},
	Command:     lipgloss.NoColor{},

	Search:       lipgloss.NoColor{},
	SearchPrompt: lipgloss.NoColor{},
}

func (m Mode) String() string {
	return modeName[m]
}

func (m Mode) FullString(formatted bool) string {
	str := fullName[m]

	if formatted {
		name := strings.ReplaceAll(str, "_", " ")
		str = "-- " + strings.ToUpper(name) + "-- "
	}

	return str
}

func (m Mode) Colour() color.Color {
	return colour[m]
}

type ModeInstance struct {
	Current Mode
}

func (m *ModeInstance) IsAnyVisual() bool {
	return m.Current == Visual ||
		m.Current == VisualLine ||
		m.Current == VisualBlock
}

func SupportsMotion() []Mode {
	return []Mode{Normal, Visual, VisualLine, VisualBlock}
}
