package theme

import (
	"bellbird-notes/app/config"
	"image/color"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	bl "github.com/winder/bubblelayout"
	"golang.org/x/term"
)

// @todo make this a theme.conf or whatever
// colors
var (
	ColourBorder        = lipgloss.Color("#606d87")
	ColourBorderFocused = lipgloss.Color("#69c8dc")
	ColourFg            = lipgloss.NoColor{}
	ColourBgSelected    = lipgloss.Color("#424B5D")
	ColourDirty         = lipgloss.Color("#c05d5f")
	ColourTitle         = lipgloss.Color("#999999")
	ColourSelection     = lipgloss.Color("#666")

	ColourSearchHighlight        = lipgloss.Color("#ffcb78")
	ColourSearchHighlightFocused = lipgloss.Color("#c59359")
	ColourSearchFg               = lipgloss.Color("#333")
)

type icon struct {
	Nerd string
	Alt  string
}

var (
	IconPen       = icon{Nerd: "", Alt: ">"}
	IconNote      = icon{Nerd: "󰎞", Alt: ""}
	IconDirOpen   = icon{Nerd: "", Alt: "▼"}
	IconDirClosed = icon{Nerd: "󰉋", Alt: "▶"}
	IconDot       = icon{Nerd: "", Alt: "*"}
	IconPin       = icon{Nerd: "󰐃", Alt: "#"}
)

func Icon(icon icon, nerdFont bool) string {
	icn := icon.Alt
	if nerdFont {
		icn = icon.Nerd
	}
	return icn
}

func BorderColour(focused bool) color.Color {
	borderColour := ColourBorder
	if focused {
		borderColour = ColourBorderFocused
	}
	return borderColour
}

type Theme struct {
	Conf *config.Config
}

func New(conf *config.Config) Theme {
	return Theme{
		Conf: conf,
	}
}

func (t *Theme) Header(title string, colWidth int, focused bool) string {
	borderColour := BorderColour(focused)
	titleColour := ColourTitle

	if focused {
		titleColour = ColourBorderFocused
	}

	// @todo clean this shit up
	b := t.BorderStyle()
	b.Left = t.BorderStyle().TopLeft
	ts := lipgloss.NewStyle().
		Border(b, false, false, false, true).
		BorderForeground(borderColour).
		Foreground(titleColour).
		Padding(0, 1).
		Bold(true)

	ls := lipgloss.NewStyle().Foreground(borderColour)
	title = ts.Render(title)
	line := ls.Render(strings.Repeat(
		t.BorderStyle().Top,
		max(0, colWidth-lipgloss.Width(title)-1)),
	)

	borderTopRight := ls.Render(t.BorderStyle().TopRight)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		line,
		borderTopRight,
	)
}

// BaseColumnLayout provides thae basic layout style for a column
func (t *Theme) BaseColumnLayout(size bl.Size, focused bool) lipgloss.Style {
	borderColour := BorderColour(focused)
	_, termHeight := TerminalSize()

	return lipgloss.NewStyle().
		Border(t.BorderStyle()).
		BorderTop(false).
		BorderForeground(borderColour).
		Foreground(ColourFg).
		Width(size.Width).
		Height(termHeight)
}

// TerminalSize determines the current
// Terminal providing a fallback and subtracting 1 from height
// because otherwise the upper part of the ui gets truncated
func TerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default width if terminal size can't be detected
		width = 80
	}
	return width, height
}

func (t *Theme) BorderStyle() lipgloss.Border {
	border, err := t.Conf.Value(config.Theme, config.Border)
	style := lipgloss.NormalBorder()

	if err != nil {
		return style
	}

	switch border.Value {
	case "normal":
		return lipgloss.NormalBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "rounded":
		return lipgloss.RoundedBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "none":
		return lipgloss.HiddenBorder()
	default:
		return style
	}
}
