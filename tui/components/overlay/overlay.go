package overlay

// Most code taken from
// https://github.com/charmbracelet/lipgloss/pull/102
// with the necessary functions directly from lipgloss

import (
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/theme"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"
)

type Overlay struct {
	x  int
	y  int
	fg string
	bg string
}

func (o *Overlay) SetContent(content string) {
	o.fg = content
}

func (o *Overlay) SetBg(bg string) {
	o.bg = bg
}

func (o *Overlay) SetPosition(x int, y int) {
	o.x = x
	o.y = y
}

// String places fg on top of bg.
func (o *Overlay) String(opts ...WhitespaceOption) string {
	fgLines, fgWidth := getLines(o.fg)
	bgLines, bgWidth := getLines(o.bg)
	bgHeight := len(bgLines)
	fgHeight := len(fgLines)

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		// FIXME: return fg or bg?
		return o.fg
	}
	// TODO: allow placement outside of the bg box?
	x := utils.Clamp(o.x, 0, bgWidth-fgWidth)
	y := utils.Clamp(o.y, 0, bgHeight-fgHeight)

	ws := &whitespace{}
	for _, opt := range opts {
		opt(ws)
	}

	var b strings.Builder
	for i, bgLine := range bgLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			b.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := truncate.String(bgLine, uint(x))
			pos = ansi.StringWidth(left)
			b.WriteString(left)
			if pos < x {
				b.WriteString(ws.render(x - pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		b.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		right := ansi.TruncateLeft(bgLine, pos, "")
		bgWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgWidth-pos {
			b.WriteString(ws.render(bgWidth - rightWidth - pos))
		}

		b.WriteString(right)
	}

	return b.String()
}

// whitespace is a whitespace renderer.
type whitespace struct {
	chars string
	style lipgloss.Style
}

// Render whitespaces.
func (w whitespace) render(width int) string {
	if w.chars == "" {
		w.chars = " "
	}

	r := []rune(w.chars)
	j := 0
	b := strings.Builder{}

	// Cycle through runes and print them into the whitespace.
	for i := 0; i < width; {
		b.WriteRune(r[j])
		j++
		if j >= len(r) {
			j = 0
		}
		i += ansi.StringWidth(string(r[j]))
	}

	// Fill any extra gaps white spaces. This might be necessary if any runes
	// are more than one cell wide, which could leave a one-rune gap.
	short := width - ansi.StringWidth(b.String())
	if short > 0 {
		b.WriteString(strings.Repeat(" ", short))
	}

	return w.style.Render(b.String())
}

// overlayPosition returns the top center position of the application screen
func (o *Overlay) CalculatePosition(overlayWidth int) (int, int) {
	termW, _ := theme.TerminalSize()

	x := (termW / 2) - (overlayWidth / 2)
	y := 2

	return x, y
}

// WhitespaceOption sets a styling rule for rendering whitespace.
type WhitespaceOption func(*whitespace)

// Split a string into lines, additionally returning the size of the widest
// line.
func getLines(s string) (lines []string, widest int) {
	s = strings.ReplaceAll(s, "\t", "    ")
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		w := ansi.StringWidth(l)
		if widest < w {
			widest = w
		}
	}

	return lines, widest
}
