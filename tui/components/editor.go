package components

import (
	"fmt"
	"os"
	"strings"

	"bellbird-notes/app"
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
	isAtLineEnd   bool
	isAtLineStart bool

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
		isAtLineEnd:   false,
		isAtLineStart: false,
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
			cmd = e.handleNormalMode(msg)
		// -- INSERT --
		case mode.Insert:
			cmd = e.handleInsertMode(msg)
		// -- REPLACE --
		case mode.Replace:
			cmd = e.handleReplaceMode(msg)
		// -- COMMAND --
		case mode.Command:
			cmd = e.handleCommandMode(msg)
		// -- OPERATOR --
		// handles the double key thingy like dd, yy, gg
		case mode.Operator:
			cmd = e.handleOperatorMode(msg)
		}
		e.checkDirty(origCnt)

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height - 1
	case errMsg:
		e.err = msg
		return e, nil
	}

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

// NewBuffer creates a new buffer, sets the textareas content
// and creates a new history for the buffer
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

// OpenBuffer attempts to open the buffer with the given path.
// If no buffer is found a new buffer is created
func (e *Editor) OpenBuffer(path string) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	buf, exists := e.bufferExists(path)
	// create new buffer if we can't find anything
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

// SaveBuffer writes the current buffer's content to the corresponding
// file on the disk and resets the dirty state
func (e *Editor) SaveBuffer() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{
		Type:   message.Success,
		Column: sbc.General,
	}

	rootDir, _ := app.NotesRootDir()
	path := e.CurrentBuffer.Path
	relative_path := strings.ReplaceAll(e.CurrentBuffer.Path, rootDir, ".")
	bytes, err := notes.Write(path, e.Textarea.Value())

	if err != nil {
		debug.LogErr(err)
		return statusMsg
	}

	e.CurrentBuffer.Dirty = false

	resultMsg := fmt.Sprintf(
		message.StatusBar.FileWritten,
		relative_path, 0, bytes,
	)

	statusMsg.Content = resultMsg
	return statusMsg
}

// DirtyBuffers collects all the dirty, dirty buffers
func (e *Editor) DirtyBuffers() []Buffer {
	bufs := make([]Buffer, 0)

	for i := range e.Buffers {
		if e.Buffers[i].Dirty {
			bufs = append(bufs, e.Buffers[i])
		}
	}

	return bufs
}

// bufferExists returns whether a buffer is in memory
func (e *Editor) bufferExists(path string) (*Buffer, bool) {
	for i := range e.Buffers {
		if e.Buffers[i].Path == path {
			return &e.Buffers[i], true
		}
	}
	return nil, false
}

// enterInsertMode sets the current editor mode to insert
// and creates a new history entry
func (e *Editor) enterInsertMode() {
	e.Vim.Mode.Current = mode.Insert
	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
}

// enterNormalMode sets the current editor mode to normal,
// checks if the cursor position exceeds the line length and
// saves the cursor position.
// It also updates the current history entry
func (e *Editor) enterNormalMode() {
	e.Vim.Mode.Current = mode.Normal

	// We need to remember if the cursor is at the and of the line
	// so that lineup and linedown moves the cursor to the end
	// when it's supposed to do so
	e.isAtLineEnd = false
	if e.Textarea.IsExceedingLine() {
		e.Textarea.CursorVimEnd()
		e.isAtLineEnd = true
	}

	if e.CurrentBuffer == nil {
		return
	}

	e.saveCursorPos()

	e.CurrentBuffer.Content = e.Textarea.Value()
	e.CurrentBuffer.History.UpdateEntry(
		e.Textarea.Value(),
		e.Textarea.CursorPos(),
	)
}

// checkDirty marks the current buffer as dirty if the current
// buffer is unsaved and the content differs from the saved content's file
func (e *Editor) checkDirty(origCnt string) {
	val := e.Textarea.Value()
	if origCnt != val {
		e.CurrentBuffer.Content = val
		e.CurrentBuffer.Dirty = true
	}
}

// setTextareaSize update the textarea height and width to match
// the height and width of the editor
func (e *Editor) setTextareaSize() {
	const reserverdLines = 3
	e.Textarea.SetWidth(e.Size.Width)
	e.Textarea.SetHeight(e.Size.Height - reserverdLines)
}

// saveCursorPos saves the cursors current column offset and row
func (e *Editor) saveCursorPos() {
	e.CurrentBuffer.CursorPos = e.Textarea.CursorPos()
}

// saveCursorRow saves the cursors current row
func (e *Editor) saveCursorRow() {
	e.CurrentBuffer.CursorPos.Row = e.Textarea.CursorPos().Row
}

// saveCursorCol saves the cursors current column offset
//func (e *Editor) saveCursorCol() {
//	e.CurrentBuffer.CursorPos.ColumnOffset = e.Textarea.CursorPos().ColumnOffset
//}

func (e *Editor) operator(c string) {
	e.Vim.Mode.Current = mode.Operator
	e.Vim.Pending.operator = c
}

// moveCharacterLeft moves the cursor one character to the left
// and checks if the cursor is either at the end or the beginning
// of the line and saves it's position
func (e *Editor) moveCharacterLeft() {
	e.Textarea.CharacterLeft(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
}

// moveCharacterRight moves the cursor one character to the right
// and checks if the cursor is either at the end or the beginning
// of the line and saves its position
func (e *Editor) moveCharacterRight() {
	e.Textarea.CharacterRight(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
}

// inserAfter enters insert mode one character after the current cursor's
// position and saves its position
func (e *Editor) inserAfter() {
	e.Textarea.CharacterRight(true)
	e.enterInsertMode()
	e.saveCursorPos()
}

// insertLineStart moves the cursor to the beginning of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) insertLineStart() {
	e.Textarea.CursorInputStart()
	e.enterInsertMode()
	e.saveCursorPos()
}

// insertLineEnd moves the cursor to the end of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) insertLineEnd() {
	e.Textarea.CursorEnd()
	e.enterInsertMode()
	e.saveCursorPos()
}

// insertLineAbove creates and empty line above the current line
// and enters insert mode
func (e *Editor) insertLineAbove() {
	e.Textarea.CursorUp()
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.enterInsertMode()
}

// insertLineBelow creates and empty line below the current line
// and enters insert mode
func (e *Editor) insertLineBelow() {
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.enterInsertMode()
}

// lineUp moves the cursor one line up and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (e *Editor) lineUp() {
	e.Textarea.CursorUp()
	e.Textarea.RepositionView()

	pos := e.CurrentBuffer.CursorPos
	// if we have a wrapped line we skip the wrapped part of the line
	if pos.Row == e.Textarea.CursorPos().Row {
		e.Textarea.CursorUp()
	}

	e.Textarea.SetCursor(pos.ColumnOffset)
	e.saveCursorRow()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorVimEnd()
	}
}

// lineDown moves the cursor one line down and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (e *Editor) lineDown() {
	e.Textarea.CursorDown()
	e.Textarea.RepositionView()

	pos := e.CurrentBuffer.CursorPos
	// if we have a wrapped line we skip the wrapped part of the line
	if pos.Row == e.Textarea.CursorPos().Row {
		e.Textarea.CursorDown()
	}

	e.Textarea.SetCursor(pos.ColumnOffset)
	e.saveCursorRow()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorVimEnd()
	}
}

// goToLineStart moves the cursor to the beginning of the line,
// sets isAtLineStart and saves the cursor position
func (e *Editor) goToLineStart() {
	e.Textarea.CursorStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.saveCursorPos()
}

// goToInputStart moves the cursor to the first character of the line,
// checks if the cursor is at the beginning of the line
// and saves the cursor position
func (e *Editor) goToInputStart() {
	e.Textarea.CursorInputStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.saveCursorPos()
}

// goToLineEnd moves the cursor to the end of the line, sets isAtLineEnd
// and saves the cursor position
func (e *Editor) goToLineEnd() {
	e.Textarea.CursorVimEnd()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
}

// goToTop moves the cursor to the beginning of the buffer
func (e *Editor) goToTop() {
	e.Textarea.MoveToBegin()
	e.Textarea.RepositionView()
}

// goToTop moves the cursor to the bottom of the buffer
func (e *Editor) goToBottom() {
	e.Textarea.MoveToEnd()
	e.Textarea.RepositionView()
}

// wordRightEnd moves the cursor to the end of the next word
func (e *Editor) wordRightEnd() {
	e.Textarea.CharacterRight(false)
	e.Textarea.WordRight()
	e.Textarea.CharacterLeft(false)
}

// wordRightStart moves the cursor to the beginning of the next word
func (e *Editor) wordRightStart() {
	e.Textarea.WordRight()
	e.Textarea.CharacterRight(false)
}

// undo sets the buffer content to the previous history entry
func (e *Editor) undo() {
	val, cursorPos := e.CurrentBuffer.undo()
	e.Textarea.SetValue(val)
	e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
}

// redo sets the buffer content to the next history entry
func (e *Editor) redo() {
	val, cursorPos := e.CurrentBuffer.redo()
	e.Textarea.SetValue(val)
	defer e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
}
