package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"os"
	"strings"
	"unicode"

	"slices"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Editor struct {
	Component

	Buffers       []Buffer
	CurrentBuffer Buffer
	Textarea      textarea.Model
	Mode          app.ModeInstance

	err error
}
type errMsg error

type Buffer struct {
	Index       int
	CurrentLine int
	CursorPos   int
	Path        string
	Content     string
}

type EditorText struct {
	text      string
	cursorPos CursorPos
}

type CursorPos struct {
	ColOffset int
	Line      int
}

const (
	charLimit       = 0
	maxHeight       = 0
	showLineNumbers = false
)

var (
	cursorLine          = lipgloss.NewStyle()
	borderColour        = lipgloss.Color("#424B5D")
	focusedBorderColour = lipgloss.Color("#69c8dc")

	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(focusedBorderColour)

	blurredStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColour)
)

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (e *Editor) Init() tea.Cmd {
	return textarea.Blink
}

func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !e.Textarea.Focused() {
			cmd = e.Textarea.Focus()
		}

		switch e.Mode.Current {
		case app.InsertMode:
			if msg.String() == "esc" {
				e.Mode.Current = app.NormalMode
				return e, nil
			}

			e.Textarea, cmd = e.Textarea.Update(msg)
			//app.LogDebug(e.Textarea.LineInfo().Width)
			//editorText := e.wordWrap(e.Textarea.Value(), e.Size.Width-5)
			//e.Textarea.InsertString(editorText.text)
			//e.Textarea.SetCursor(editorText.cursorPos.ColOffset)

			return e, cmd

		case app.NormalMode:
			cursorPos := e.Textarea.LineInfo().ColumnOffset
			switch msg.String() {
			case "i":
				e.EnterInsertMode()
			case "h":
				if cursorPos < 0 {
					e.Textarea.CursorUp()
					e.Textarea.CursorEnd()
				} else {
					e.Textarea.SetCursor(cursorPos - 1)
				}
			case "l":
				if cursorPos > e.Textarea.LineInfo().Width-3 {
					e.Textarea.CursorDown()
					e.Textarea.SetCursor(0)
				} else {
					e.Textarea.SetCursor(cursorPos + 1)
				}
			case "j":
				e.Textarea.CursorDown()
			case "k":
				e.Textarea.CursorUp()
			}
		}

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height - 1
	case errMsg:
		e.err = msg
		return e, nil
	}

	e.Textarea.SetWidth(e.Size.Width)
	e.Textarea.SetHeight(e.Size.Height - 3)

	//e.Textarea, cmd = e.Textarea.Update(msg)
	cmds = append(cmds, cmd)
	// Handle keyboard and mouse events in the viewport
	//_, cmd = e.viewport.Update(msg)
	//cmds = append(cmds, cmd)

	return e, tea.Batch(cmds...)
}

func (e *Editor) View() string {
	if !e.Focused {
		e.Textarea.Blur()
	}

	return e.build()
}

func NewEditor() *Editor {
	textarea := textarea.New()
	textarea.ShowLineNumbers = showLineNumbers
	textarea.Prompt = ""
	textarea.FocusedStyle.CursorLine = cursorLine
	textarea.FocusedStyle.Base = focusedStyle
	textarea.BlurredStyle.Base = blurredStyle
	textarea.CharLimit = charLimit
	textarea.MaxHeight = maxHeight

	editor := &Editor{
		Mode:     app.ModeInstance{Current: app.NormalMode},
		Textarea: textarea,
	}

	return editor
}

func (e *Editor) build() string {
	return e.Textarea.View()
}

func (e *Editor) NewBuffer(path string) messages.StatusBarMsg {
	note, err := os.ReadFile(path)

	if err != nil {
		app.LogErr(err)
		return messages.StatusBarMsg{Content: err.Error()}
	}

	noteString := string(note)
	noteString = e.wordWrap(noteString, e.Size.Width-5).text
	buffer := Buffer{
		Index:   len(e.Buffers) + 1,
		Path:    path,
		Content: noteString,
	}

	e.Buffers = append(e.Buffers, buffer)
	e.CurrentBuffer = buffer

	content := ""
	if e.CurrentBuffer != (Buffer{}) {
		content = e.CurrentBuffer.Content
	}
	e.Textarea.SetValue("")
	e.Textarea.CursorStart()
	e.Textarea.InsertString(content)
	e.Textarea.CursorStart()
	e.Textarea.SetWidth(e.Size.Width)
	e.Textarea.SetHeight(e.Size.Height - 3)

	return messages.StatusBarMsg{}
}

// wordWrap wraps the given str by width
//
// This is a go translation of https://stackoverflow.com/a/17635
func (e *Editor) wordWrap(str string, width int) EditorText {
	splitChars := []rune{' ', '-', '\t', '.'}
	words := explode(str, splitChars)

	lineLength := e.Textarea.LineInfo().Width
	var strBuilder strings.Builder

	for _, word := range words {
		// If adding the new word to the current line would be too long,
		// then put it on a new line (and split it up if it's too long).
		if lineLength > width {
			// Only move down to a new line if we have text on the current line.
			// Avoids situation where
			// wrapped whitespace causes emptylines in text.
			if lineLength > 0 {
				strBuilder.WriteString("\n")
				//lin = 0
			}

			// If the current word is too long
			// to fit on a line (even on its own),
			// then split the word up.
			for len(word) > width {
				strBuilder.WriteString(word[:width-1] + "-\n")
				word = word[width-1:]
			}

			// Remove leading whitespace from the word,
			// so the new line starts flush to the left.
			word = strings.TrimLeftFunc(word, unicode.IsSpace)
		}
		strBuilder.WriteString(word)
		//currLineLength += len(word)
	}

	e.Textarea.LineInfo()
	return EditorText{
		text: strBuilder.String(),
		cursorPos: CursorPos{
			e.Textarea.LineInfo().ColumnOffset,
			e.Textarea.Line(),
		},
	}
}

// This is a go translation of https://stackoverflow.com/a/17635
func explode(str string, splitChars []rune) []string {
	parts := []string{}
	startIndex := 0

	for i, r := range str {
		if slices.Contains(splitChars, r) {
			word := str[startIndex:i]
			if word != "" {
				parts = append(parts, word)
			}

			// Dashes and the like should stick to the word occuring before it.
			// Whitespace doesn't have to.
			if unicode.IsSpace(r) {
				parts = append(parts, string(r))
			} else {
				if len(parts) > 0 {
					parts[len(parts)-1] += string(r)
				} else {
					parts = append(parts, string(r))
				}
			}

			startIndex = i + 1
		}
	}

	if startIndex < len(str) {
		parts = append(parts, str[startIndex:])
	}

	return parts
}

func (e *Editor) EnterInsertMode() messages.StatusBarMsg {
	if e.Focused {
		e.Mode.Current = app.InsertMode
	}
	return messages.StatusBarMsg{}
}

func (e *Editor) ExitInsertMode() messages.StatusBarMsg {
	e.Mode.Current = app.NormalMode
	return messages.StatusBarMsg{}
}
