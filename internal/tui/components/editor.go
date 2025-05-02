package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/components/textarea"
	"bellbird-notes/internal/tui/keyinput"
	"bellbird-notes/internal/tui/messages"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

type Editor struct {
	Component

	Buffers       []Buffer
	CurrentBuffer Buffer
	Textarea      textarea.Model
	Vim           Vim

	err error
}

type errMsg error

type Buffer struct {
	Index       int
	CurrentLine int
	CursorPos   textarea.CursorPos
	Path        string
	Content     string

	History textarea.History
}

type Input struct {
	keyinput.Input
	key      string
	operator string
}

type Vim struct {
	Mode    app.ModeInstance
	Pending Input
}

func NewEditor() *Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = showLineNumbers
	ta.Prompt = ""
	ta.FocusedStyle.CursorLine = cursorLine
	ta.FocusedStyle.Base = focusedStyle
	ta.BlurredStyle.Base = blurredStyle
	ta.CharLimit = charLimit
	ta.MaxHeight = maxHeight

	editor := &Editor{
		Vim: Vim{
			Mode: app.ModeInstance{Current: app.NormalMode},
			Pending: Input{
				keyinput.Input{Ctrl: false, Alt: false, Shift: false},
				"",
				"",
			},
		},
		Textarea: ta,
	}

	return editor
}

func (e *Editor) NewBuffer(path string) messages.StatusBarMsg {
	note, err := os.ReadFile(path)

	if err != nil {
		app.LogErr(err)
		return messages.StatusBarMsg{Content: err.Error()}
	}

	buffer := Buffer{
		Index:   len(e.Buffers) + 1,
		Path:    path,
		Content: string(note),
		History: textarea.NewHistory(),
	}

	e.Buffers = append(e.Buffers, buffer)
	e.CurrentBuffer = buffer

	content := ""
	if e.CurrentBuffer.Path == path {
		content = e.CurrentBuffer.Content
	}

	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
	e.CurrentBuffer.History.UpdateEntry(content, textarea.CursorPos{})

	e.Textarea.SetValue(content)
	e.Textarea.MoveToBegin()
	e.Textarea.SetWidth(e.Size.Width)
	e.Textarea.SetHeight(e.Size.Height - 3)

	return messages.StatusBarMsg{}
}

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

		key := msg.String()
		if strings.Contains(key, "ctrl+") {
			e.Vim.Pending.Ctrl = true
			e.Vim.Pending.key = strings.Split(key, "+")[1]
		}
		if strings.Contains(key, "alt+") {
			e.Vim.Pending.Alt = true
			e.Vim.Pending.key = strings.Split(key, "+")[1]
		}
		if strings.Contains(key, "shift+") {
			e.Vim.Pending.Shift = true
			e.Vim.Pending.key = strings.Split(key, "+")[1]
		}

		switch e.Vim.Mode.Current {
		case app.InsertMode:
			if msg.String() == "esc" {
				e.Vim.Mode.Current = app.NormalMode
				e.CurrentBuffer.History.UpdateEntry(
					e.Textarea.Value(),
					e.Textarea.CursorPos(),
				)
				return e, nil
			}

			e.Textarea, cmd = e.Textarea.Update(msg)

			return e, cmd

		case app.NormalMode:
			switch msg.String() {
			case "i":
				e.enterInsertMode()
			case "I":
				e.Textarea.CursorInputStart()
				e.enterInsertMode()
			case "a":
				e.Textarea.CharacterRight()
				e.enterInsertMode()
			case "A":
				e.Textarea.CursorEnd()
				e.enterInsertMode()
			case "r":
				e.Vim.Mode.Current = app.ReplaceMode
			//case "v":
			//	e.Vim.Mode.Current = app.VisualMode
			case "h":
				e.Textarea.CharacterLeft(false)
			case "l":
				e.Textarea.CharacterRight()
			case "j":
				e.Textarea.CursorDown()
				e.Textarea.RepositionView()
			case "k":
				e.Textarea.CursorUp()
				e.Textarea.RepositionView()
			case "u":
				val, cursorPos := e.CurrentBuffer.History.Undo()
				e.Textarea.SetValue(val)
				e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
			case "ctrl+r":
				val, cursorPos := e.CurrentBuffer.History.Redo()
				e.Textarea.SetValue(val)
				defer e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
			case "w":
				e.Textarea.WordRight()
				e.Textarea.CharacterRight()
			case "e":
				e.Textarea.WordRight()
				e.Textarea.CharacterRight()
				e.Textarea.CharacterLeft(false)
			case "b":
				e.Textarea.WordLeft()
			case "^", "_":
				e.Textarea.CursorInputStart()
			case "0":
				e.Textarea.CursorStart()
			case "$":
				e.Textarea.CursorEnd()
			case "o":
				e.Textarea.CursorEnd()
				e.Textarea.InsertRune('\n')
				e.Textarea.RepositionView()
				e.enterInsertMode()
			case "O":
				e.Textarea.CursorUp()
				e.Textarea.CursorEnd()
				e.Textarea.InsertRune('\n')
				e.Textarea.RepositionView()
				e.enterInsertMode()
			case "d":
				e.operator("d")
			case "D":
				e.Textarea.DeleteAfterCursor()
			case "g":
				e.operator("g")
			case "G":
				e.Textarea.MoveToEnd()
				e.Textarea.RepositionView()
			case "ctrl+d":
				e.Textarea.DownHalfPage()
			case "ctrl+u":
				e.Textarea.UpHalfPage()
			}
			e.Vim.Pending.ResetKeysDown()
			//app.LogDebug(
			//	e.Vim.Pending.key,
			//	e.Vim.Pending.Ctrl,
			//	e.Vim.Pending.Alt,
			//	e.Vim.Pending.Shift,
			//)

		//case app.ReplaceMode:

		// handles the double key thingy like dd, yy, gg
		case app.OperatorMode:
			if e.Vim.Pending.operator == "d" {
				switch msg.String() {
				case "d":
					e.Textarea.DeleteLine()
				case "j":
					e.Textarea.DeleteLines(2, false)
				case "k":
					e.Textarea.DeleteLines(2, true)
				}
			}
			if e.Vim.Pending.operator == "g" {
				switch msg.String() {
				case "g":
					e.Textarea.MoveToBegin()
					e.Textarea.RepositionView()
				}
			}
			e.Vim.Pending.ResetKeysDown()
			e.Vim.Mode.Current = app.NormalMode
			e.Vim.Pending.operator = ""
		}

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height - 1
	case errMsg:
		e.err = msg
		return e, nil
	}

	e.CurrentBuffer.CursorPos = e.Textarea.CursorPos()
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

func (e *Editor) build() string {
	return e.Textarea.View()
}

func (e *Editor) enterInsertMode() {
	e.Vim.Mode.Current = app.InsertMode
	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
}

func (e *Editor) operator(c string) {
	e.Vim.Mode.Current = app.OperatorMode
	e.Vim.Pending.operator = c
}
