package mode

import (
	"image/color"

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
)

var modeName = map[Mode]string{
	Normal:      "n",
	Insert:      "i",
	Visual:      "v",
	VisualLine:  "vl",
	VisualBlock: "vb",
	Operator:    "o",
	Replace:     "r",
	Command:     "c",
}

var fullName = map[Mode]string{
	Normal:      "-- NORMAL --",
	Insert:      "-- INSERT --",
	Visual:      "-- VISUAL --",
	VisualLine:  "-- VISUAL LINE --",
	VisualBlock: "-- VISUAL BLOCK --",
	Replace:     "-- REPLACE --",
	Operator:    "",
	Command:     "",
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
}

func (m Mode) String() string {
	return modeName[m]
}

func (m Mode) FullString() string {
	return fullName[m]
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
