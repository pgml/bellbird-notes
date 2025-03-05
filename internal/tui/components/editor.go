package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"os"
	"strings"
	"unicode"

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
	cursorPos int
}

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	cursorLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230"))

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

	endOfBufferStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("235"))

	focusedPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder())
)

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (e *Editor) Init() tea.Cmd {
	return textarea.Blink
}

func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	//termWidth, termHeight := theme.GetTerminalSize()

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
			//editorText := e.wordWrap(e.Textarea.Value(), e.Size.Width-5)
			//e.Textarea.SetValue(editorText.text)
			//e.Textarea.SetCursor(editorText.cursorPos)
			//app.LogDebug(editorText.cursorPos)

			return e, cmd

		case app.NormalMode:
			cursorPos := e.Textarea.LineInfo().ColumnOffset
			switch msg.String() {
			//case "i":
			//	e.EnterInsertMode()
			case "h":
				e.Textarea.SetCursor(cursorPos - 1)
			case "l":
				e.Textarea.SetCursor(cursorPos + 1)
			case "j":
				e.Textarea.CursorDown()
			case "k":
				e.Textarea.CursorUp()
			}
		}
		//return e.vim(msg)

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height - 1
	//if !e.ready {
	//	e.viewport = viewport.New(termWidth, termHeight-1)
	//	e.viewport.SetContent(e.build())
	//	e.viewport.KeyMap = viewport.KeyMap{}
	//	e.ready = true
	//} else {
	//	e.viewport.Width = termWidth
	//	e.viewport.Height = termHeight - 1
	//}
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
	//if !e.ready {
	//	return "\n  Initializing..."
	//}

	//e.viewport.SetContent(e.build())
	//e.viewport.Style = theme.BaseColumnLayout(e.Size, e.Focused)

	if !e.Focused {
		e.Textarea.Blur()
	}

	app.LogDebug(len(e.Buffers))

	return e.build()
	//return e.viewport.View()
}

func NewEditor() *Editor {
	textarea := textarea.New()
	//textarea.ShowLineNumbers = false
	textarea.Prompt = ""
	//app.LogDebug(termWidth, termHeight, "asd")
	//textarea.FocusedStyle.CursorLine = lipgloss.NewStyle()

	//conf := config.New()

	borderColour := lipgloss.Color("#424B5D")
	focusedBorderColour := lipgloss.Color("#69c8dc")
	textarea.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(focusedBorderColour)

	textarea.BlurredStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColour)
	textarea.CharLimit = 0
	textarea.MaxHeight = 0

	editor := &Editor{
		Mode:     app.ModeInstance{Current: app.NormalMode},
		Textarea: textarea,
	}

	//editor.Refresh(false)
	return editor
}

func (e *Editor) build() string {
	return e.Textarea.View()
}

func (e *Editor) NewBuffer(path string) messages.StatusBarMsg {
	note, err := os.ReadFile(path)

	//e.Buffers = []Buffer{}

	//runtime.GC()
	if err != nil {
		app.LogErr(err)
		return messages.StatusBarMsg{Content: err.Error()}
	}
	cleanedText := strings.ReplaceAll(string(note), "\r\n", "\n")
	buffer := Buffer{
		Index:   len(e.Buffers) + 1,
		Path:    path,
		Content: cleanedText,
	}

	e.Buffers = append(e.Buffers, buffer)
	e.CurrentBuffer = buffer

	content := ""
	if e.CurrentBuffer != (Buffer{}) {
		content = e.CurrentBuffer.Content
	}
	e.Textarea.SetValue(content)

	return messages.StatusBarMsg{}
}
func cleanText(input string) string {
	// Remove non-printable characters
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			return r
		}
		return -1
	}, input)
}

// wordWrap wraps the given str by width
//
// This is a go translation of https://stackoverflow.com/a/17635
func (e *Editor) wordWrap(str string, width int) EditorText {
	splitChars := []rune{' ', '-', '\t', '.'}
	words := explode(str, splitChars)

	currLineLength := 0
	var strBuilder strings.Builder

	for _, word := range words {
		// If adding the new word to the current line would be too long,
		// then put it on a new line (and split it up if it's too long).
		if currLineLength+len(word) > width {
			// Only move down to a new line if we have text on the current line.
			// Avoids situation where
			// wrapped whitespace causes emptylines in text.
			if currLineLength > 0 {
				strBuilder.WriteString("\n")
				currLineLength = 0
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
		currLineLength += len(word)
	}

	return EditorText{
		text:      strBuilder.String(),
		cursorPos: e.Textarea.LineInfo().ColumnOffset,
	}
}

// This is a go translation of https://stackoverflow.com/a/17635
func explode(str string, splitChars []rune) []string {
	parts := []string{}
	startIndex := 0

	for i, r := range str {
		for _, char := range splitChars {
			if r == char {
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
				break
			}
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
