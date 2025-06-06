package components

import (
	"fmt"
	"os"
	"strings"

	"bellbird-notes/app/debug"
	"bellbird-notes/app/notes"
	"bellbird-notes/tui/components/textarea"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	sbc "bellbird-notes/tui/types/statusbar_column"

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
	CurrentBuffer *Buffer
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
	History     textarea.History
	Dirty       bool
}

type Input struct {
	keyinput.Input
	key      string
	operator string
}

type Vim struct {
	Mode    mode.ModeInstance
	Pending Input
}

func (b *Buffer) undo() (string, textarea.CursorPos) {
	return b.History.Undo()
}

func (b *Buffer) redo() (string, textarea.CursorPos) {
	return b.History.Redo()
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
			Mode: mode.ModeInstance{Current: mode.Normal},
			Pending: Input{
				keyinput.Input{Ctrl: false, Alt: false, Shift: false},
				"",
				"",
			},
		},
		Textarea:      ta,
		Component:     Component{},
		Buffers:       []Buffer{},
		CurrentBuffer: &Buffer{},
		err:           nil,
	}

	return editor
}

// Init initialises the Model on program load.
// It partially implements the tea.Model interface.
func (e *Editor) Init() tea.Cmd {
	return textarea.Blink
}

func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

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

		origCnt := e.Textarea.Value()

		switch e.Vim.Mode.Current {
		// -- NORMAL --
		case mode.Normal:
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
				e.Vim.Mode.Current = mode.Replace
			//case "v":
			//	e.Vim.Mode.Current = app.VisualMode
			case "h":
				e.Textarea.CharacterLeft(false)

			case "l":
				e.Textarea.CharacterRight()

			case "j":
				e.lineDown()

			case "k":
				e.lineUp()

			case "u":
				val, cursorPos := e.CurrentBuffer.undo()
				e.Textarea.SetValue(val)
				e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)

			case "ctrl+r":
				val, cursorPos := e.CurrentBuffer.redo()
				e.Textarea.SetValue(val)
				defer e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)

			case "w":
				e.wordRightStart()

			case "e":
				e.wordRightEnd()

			case "b":
				e.Textarea.WordLeft()

			case "^", "_":
				e.Textarea.CursorInputStart()

			case "0":
				e.Textarea.CursorStart()

			case "$":
				e.Textarea.CursorEnd()

			case "o":
				e.insertLineBelow()

			case "O":
				e.insertLineAbove()

			case "d":
				e.operator("d")

			case "D":
				e.Textarea.DeleteAfterCursor()

			case "g":
				e.operator("g")

			case "G":
				e.goToBottom()

			case "ctrl+d":
				e.Textarea.DownHalfPage()

			case "ctrl+u":
				e.Textarea.UpHalfPage()
			case ":":
				e.Vim.Mode.Current = mode.Command
			}

			e.Vim.Pending.ResetKeysDown()

		// -- INSERT --
		case mode.Insert:
			if msg.String() == "esc" {
				e.enterNormalMode()
				return e, nil
			}

			e.Textarea, cmd = e.Textarea.Update(msg)
			e.checkDirty(e.CurrentBuffer.Content)

			return e, cmd

		// -- REPLACE --
		case mode.Replace:
			if msg.String() == "esc" {
				e.enterNormalMode()
				return e, nil
			}
			// replace current charater in simple replace mode
			// convert string character to rune
			rune := []rune(msg.String())[0]

			oldCnt := e.CurrentBuffer.Content
			e.Textarea.ReplaceRune(rune)
			e.checkDirty(oldCnt)
			e.enterNormalMode()

			return e, nil

		// -- COMMAND --
		case mode.Command:
			switch msg.String() {
			case "esc", "enter":
				e.enterNormalMode()
				return e, nil
			}

		// -- OPERATOR --
		// handles the double key thingy like dd, yy, gg
		case mode.Operator:
			//origCnt = e.CurrentBuffer.Content
			if e.Vim.Pending.operator == "d" {
				switch msg.String() {
				case "d":
					e.Textarea.DeleteLine()

				case "j":
					e.Textarea.DeleteLines(2, false)

				case "k":
					e.Textarea.DeleteLines(2, true)

				case "w":
					e.Textarea.DeleteWordRight()
				}

				e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
			}

			if e.Vim.Pending.operator == "g" {
				switch msg.String() {
				case "g":
					e.goToTop()
				}
			}

			e.Vim.Pending.ResetKeysDown()
			e.Vim.Mode.Current = mode.Normal
			e.Vim.Pending.operator = ""
		}
		e.checkDirty(origCnt)

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height - 1
	case errMsg:
		e.err = msg
		return e, nil
	}

	e.CurrentBuffer.CursorPos = e.Textarea.CursorPos()
	e.setTextareaSize()

	cmds = append(cmds, cmd)

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

func (e *Editor) NewBuffer(path string) message.StatusBarMsg {
	note, err := os.ReadFile(path)

	if err != nil {
		debug.LogErr(err)
		return message.StatusBarMsg{Content: err.Error()}
	}

	noteContent := string(note)

	buf := Buffer{
		Index:       len(e.Buffers) + 1,
		Path:        path,
		Content:     noteContent,
		History:     textarea.NewHistory(),
		CurrentLine: 0,
		CursorPos:   textarea.CursorPos{},
	}

	e.Buffers = append(e.Buffers, buf)
	e.CurrentBuffer = &e.Buffers[len(e.Buffers)-1]

	content := ""
	if e.CurrentBuffer.Path == path {
		content = e.CurrentBuffer.Content
	}

	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
	e.CurrentBuffer.History.UpdateEntry(content, textarea.CursorPos{})

	e.Textarea.SetValue(content)
	e.Textarea.MoveToBegin()
	e.setTextareaSize()

	return message.StatusBarMsg{}
}

func (e *Editor) OpenBuffer(path string) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	buf, exists := e.bufferExists(path)
	if len(e.Buffers) <= 0 || !exists {
		e.NewBuffer(path)
		return statusMsg
	}

	e.CurrentBuffer = buf

	e.Textarea.SetValue(buf.Content)
	e.Textarea.MoveCursor(buf.CursorPos.Row, buf.CursorPos.ColumnOffset)
	e.setTextareaSize()

	return statusMsg
}

func (e *Editor) SaveBuffer() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{
		Type:   message.Success,
		Column: sbc.General,
	}

	path := e.CurrentBuffer.Path
	bytes, err := notes.Write(path, e.Textarea.Value())

	if err != nil {
		debug.LogErr(err)
		return statusMsg
	}

	e.CurrentBuffer.Dirty = false

	resultMsg := fmt.Sprintf(
		message.StatusBar.FileWritten,
		path, 0, bytes,
	)

	statusMsg.Content = resultMsg
	return statusMsg
}

func (e *Editor) DirtyBuffers() []Buffer {
	bufs := make([]Buffer, 0)

	for i := range e.Buffers {
		if e.Buffers[i].Dirty {
			bufs = append(bufs, e.Buffers[i])
		}
	}

	return bufs
}

func (e *Editor) bufferExists(path string) (*Buffer, bool) {
	for i := range e.Buffers {
		if e.Buffers[i].Path == path {
			return &e.Buffers[i], true
		}
	}
	return nil, false
}

func (e *Editor) enterInsertMode() {
	e.Vim.Mode.Current = mode.Insert
	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
}

func (e *Editor) enterNormalMode() {
	e.Vim.Mode.Current = mode.Normal

	if e.CurrentBuffer == nil {
		return
	}

	e.CurrentBuffer.Content = e.Textarea.Value()
	e.CurrentBuffer.History.UpdateEntry(
		e.Textarea.Value(),
		e.Textarea.CursorPos(),
	)
}

func (e *Editor) checkDirty(origCnt string) {
	val := e.Textarea.Value()
	if origCnt != val {
		e.CurrentBuffer.Content = val
		e.CurrentBuffer.Dirty = true
	}
}

func (e *Editor) setTextareaSize() {
	e.Textarea.SetWidth(e.Size.Width)
	e.Textarea.SetHeight(e.Size.Height - 3)
}

func (e *Editor) operator(c string) {
	e.Vim.Mode.Current = mode.Operator
	e.Vim.Pending.operator = c
}

func (e *Editor) insertLineAbove() {
	e.Textarea.CursorUp()
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.enterInsertMode()
}

func (e *Editor) insertLineBelow() {
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.enterInsertMode()
}

func (e *Editor) lineUp() {
	e.Textarea.CursorUp()
	e.Textarea.RepositionView()
}

func (e *Editor) lineDown() {
	e.Textarea.CursorDown()
	e.Textarea.RepositionView()
}

func (e *Editor) goToTop() {
	e.Textarea.MoveToBegin()
	e.Textarea.RepositionView()
}

func (e *Editor) goToBottom() {
	e.Textarea.MoveToEnd()
	e.Textarea.RepositionView()
}

func (e *Editor) wordRightEnd() {
	e.Textarea.CharacterRight()
	e.Textarea.WordRight()
	e.Textarea.CharacterLeft(false)
}

func (e *Editor) wordRightStart() {
	e.Textarea.WordRight()
	e.Textarea.CharacterRight()
}
