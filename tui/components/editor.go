package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components/textarea"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/v2/cursor"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

const (
	charLimit = 0
	maxHeight = 0
)

var (
	cursorLine = lipgloss.NewStyle()

	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			Padding(0, 1).
			BorderForeground(theme.ColourBorderFocused)

	blurredStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			Padding(0, 1).
			BorderForeground(theme.ColourBorder)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")). // Green background
			Foreground(lipgloss.Color("0")).  // Black text
			Bold(true)
)

type Editor struct {
	Component

	Buffers       []Buffer
	CurrentBuffer *Buffer
	Textarea      textarea.Model
	Vim           Vim
	// CanInsert indicates whether textarea can receive input despite
	// vim mode being insert.
	CanInsert     bool
	isAtLineEnd   bool
	isAtLineStart bool

	StatusBarMsg message.StatusBarMsg

	ShowLineNumbers bool

	config config.Config
	err    error
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
	ta.Prompt = ""
	ta.Styles.Focused.CursorLine = cursorLine
	ta.Styles.Focused.Base = focusedStyle
	ta.Styles.Blurred.Base = blurredStyle
	ta.CharLimit = charLimit
	ta.MaxHeight = maxHeight

	editor := &Editor{
		Vim: Vim{
			Mode: mode.ModeInstance{Current: mode.Normal},
			Pending: Input{
				keyinput.Input{Ctrl: false, Alt: false},
				"",
				"",
			},
		},
		CanInsert:       false,
		Textarea:        ta,
		Component:       Component{},
		Buffers:         []Buffer{},
		CurrentBuffer:   &Buffer{},
		isAtLineEnd:     false,
		isAtLineStart:   false,
		ShowLineNumbers: false,
		err:             nil,
		config:          *config.New(),
	}

	editor.Textarea.ShowLineNumbers = editor.ShowLineNumbers
	editor.Textarea.ResetSelection()

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

	e.Textarea.Selection.Cursor.Blur()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !e.Textarea.Focused() {
			cmd = e.Textarea.Focus()
		}

		origCnt := e.Textarea.Value()

		switch e.Vim.Mode.Current {
		case mode.Insert:
			cmd = e.handleInsertMode(msg)

		case mode.Replace:
			cmd = e.handleReplaceMode(msg)

		case mode.Command:
			cmd = e.handleCommandMode(msg)
		}

		pos := e.Textarea.CursorPos()
		e.config.SetMetaValue(
			e.CurrentBuffer.Path,
			config.CursorPosition,
			strconv.Itoa(pos.Row)+","+strconv.Itoa(pos.ColumnOffset),
		)
		e.checkDirtySince(origCnt)

	case tea.WindowSizeMsg:
		termWidth, termHeight := theme.GetTerminalSize()
		e.Size.Width = termWidth
		e.Size.Height = termHeight
	case errMsg:
		e.err = msg
		return e, nil
	}

	e.setTextareaSize()
	//_, selectionCmd := e.Textarea.Cursor.Update(msg)
	//cmds = append(cmds, cmd, selectionCmd)
	cmds = append(cmds, cmd)

	return e, tea.Batch(cmds...)
}

func (e *Editor) View() string {
	if !e.Focused() {
		e.Textarea.Blur()
	}

	e.Textarea.Selection.Cursor.SetMode(cursor.CursorStatic)
	e.Textarea.Selection.Cursor.TextStyle = e.Textarea.SelectionStyle()
	e.Textarea.Selection.Cursor.Style = e.Textarea.SelectionStyle()
	//e.Textarea.Cursor.UpdateStyle()

	return e.build()
}

func (e *Editor) build() string {
	title := "EDITOR"

	if e.CurrentBuffer.Path != "" {
		title = e.breadcrumb()
	}

	e.header = theme.Header(title, e.Size.Width, e.Focused())
	return fmt.Sprintf("%s\n%s", e.header, e.Textarea.View())
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

	cursorPos := textarea.CursorPos{}
	if pos := e.config.MetaValue(path, config.CursorPosition); pos != "" {
		p := strings.Split(pos, ",")
		row, _ := strconv.Atoi(p[0])
		col, _ := strconv.Atoi(p[1])
		cursorPos = textarea.CursorPos{Row: row, ColumnOffset: col}
	}

	buf := Buffer{
		Index:       len(e.Buffers) + 1,
		Path:        path,
		Content:     noteContent,
		History:     textarea.NewHistory(),
		CurrentLine: 0,
		CursorPos:   cursorPos,
	}

	e.Buffers = append(e.Buffers, buf)
	e.CurrentBuffer = &e.Buffers[len(e.Buffers)-1]

	content := ""
	if e.CurrentBuffer.Path == path {
		content = e.CurrentBuffer.Content
	}

	e.Textarea.SetValue(content)
	e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
	e.Textarea.RepositionView()

	return message.StatusBarMsg{}
}

// OpenBuffer attempts to open the buffer with the given path.
// If no buffer is found a new buffer is created
func (e *Editor) OpenBuffer(path string) message.StatusBarMsg {
	relPath := utils.RelativePath(path, true)
	icon := theme.Icon(theme.IconNote)

	statusMsg := message.StatusBarMsg{
		Content: icon + " " + relPath,
		Column:  sbc.FileInfo,
	}

	buf, exists := e.bufferExists(path)
	// create new buffer if we can't find anything
	if len(e.Buffers) <= 0 || !exists {
		e.NewBuffer(path)
		return statusMsg
	}

	e.CurrentBuffer = buf

	e.Textarea.SetValue(buf.Content)
	e.Textarea.MoveCursor(buf.CursorPos.Row, buf.CursorPos.ColumnOffset)
	e.Textarea.RepositionView()

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
	relative_path := strings.ReplaceAll(path, rootDir+"/", "")
	bytes, err := notes.Write(path, e.Textarea.Value())

	if err != nil {
		debug.LogErr(err)
		return statusMsg
	}

	e.CurrentBuffer.Dirty = false

	resultMsg := fmt.Sprintf(
		message.StatusBar.FileWritten,
		relative_path, e.Textarea.LineCount(), bytes,
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

func (e *Editor) Focused() bool {
	return e.focused
}

func (e *Editor) SetFocus(focus bool) {
	e.focused = focus
}

func (e *Editor) breadcrumb() string {
	noteName := filepath.Base(e.CurrentBuffer.Path)
	pathSeparator := string(os.PathSeparator)

	relPath := utils.RelativePath(e.CurrentBuffer.Path, false)
	relPath = strings.ReplaceAll(relPath, pathSeparator, " › ")
	breadcrumb := strings.ReplaceAll(relPath, noteName, "")

	iconDir := theme.Icon(theme.IconDirClosed)
	iconNote := theme.Icon(theme.IconNote)

	return iconDir + breadcrumb + iconNote + " " + noteName
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

// EnterNormalMode sets the current editor mode to normal,
// checks if the cursor position exceeds the line length and
// saves the cursor position.
// It also updates the current history entry
func (e *Editor) EnterNormalMode() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{
		Content: "",
		Column:  sbc.General,
	}

	e.Textarea.ResetSelection()

	// We need to remember if the cursor is at the and of the line
	// so that lineup and linedown moves the cursor to the end
	// when it's supposed to do so
	e.isAtLineEnd = false
	if e.Textarea.IsExceedingLine() {
		e.Textarea.CursorLineVimEnd()
		e.isAtLineEnd = true
	}

	if e.Vim.Mode.Current == mode.Visual {
		statusMsg.Column = sbc.KeyInfo
	}

	e.Vim.Mode.Current = mode.Normal

	if e.CurrentBuffer == nil {
		return statusMsg
	}

	e.saveCursorPos()

	e.CurrentBuffer.Content = e.Textarea.Value()
	e.updateHistoryEntry()
	e.Textarea.ResetSelection()
	e.Textarea.SetCursorColor(mode.Normal.Colour())

	return statusMsg
}

// EnterInsertMode sets the current editor mode to insert
// and creates a new history entry
func (e *Editor) EnterInsertMode(withHistory bool) message.StatusBarMsg {
	e.Vim.Mode.Current = mode.Insert
	if withHistory {
		e.newHistoryEntry()
	}
	e.Textarea.SetCursorColor(mode.Insert.Colour())
	return message.StatusBarMsg{}
}

// EnterReplaceMode() sets the current editor mode to replace
// and creates a new history entry
func (e *Editor) EnterReplaceMode() message.StatusBarMsg {
	e.Vim.Mode.Current = mode.Replace
	e.newHistoryEntry()
	e.Textarea.SetCursorColor(mode.Replace.Colour())
	return message.StatusBarMsg{}
}

// EnterVisualMode() sets the current editor mode to replace
// and creates a new history entry
func (e *Editor) EnterVisualMode() message.StatusBarMsg {
	e.Vim.Mode.Current = mode.Visual
	e.Textarea.StartSelection()
	e.Textarea.SetCursorColor(mode.VisualBlock.Colour())
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) newHistoryEntry() {
	e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
}

func (e *Editor) updateHistoryEntry() {
	e.saveCursorPos()
	e.CurrentBuffer.History.UpdateEntry(
		e.Textarea.Value(),
		e.Textarea.CursorPos(),
	)
}

// checkDirty marks the current buffer as dirty if the current
// buffer is unsaved and the content differs from the saved content's file
func (e *Editor) checkDirty(fn func()) bool {
	before := e.Textarea.Value()
	fn()
	after := e.Textarea.Value()

	if before != after {
		e.CurrentBuffer.Content = after
		e.CurrentBuffer.Dirty = true
	}

	return e.CurrentBuffer.Dirty
}

func (e *Editor) checkDirtySince(previous string) {
	current := e.Textarea.Value()
	if previous != current {
		e.CurrentBuffer.Content = current
		e.CurrentBuffer.Dirty = true
	}
}

func (e *Editor) fileProgress() int {
	pc := float32(e.Textarea.Line()+1) / float32(e.Textarea.LineCount())
	return int(pc * 100.0)
}

func (e *Editor) fileProgresStr() string {
	fileProgress := strconv.Itoa(e.fileProgress())
	return fileProgress + "%"
}

func (e *Editor) cursorInfo() string {
	line := strconv.Itoa(e.Textarea.Line() + 1)
	col := strconv.Itoa(e.Textarea.LineInfo().ColumnOffset)
	return line + "," + col
}

//func (e *Editor) filePosition() int {
//	firstVis := e.Textarea.FirstVisibleLine()
//	filePos := (firstVis - 1) * 100 / (e.Textarea.LineCount() - 1)
//	return int(filePos)
//}
//
//func (e *Editor) filePosStr() string {
//	debug.LogDebug(e.Textarea.FirstVisibleLine(), e.Textarea.LineCount())
//	pos := strconv.Itoa(e.filePosition())
//	return pos + "%"
//}

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

func (e *Editor) UpdateStatusBarInfo() {
	e.StatusBarMsg = e.StatusBarInfo()
}

func (e *Editor) StatusBarInfo() message.StatusBarMsg {
	return message.StatusBarMsg{
		Content: e.cursorInfo() + "\t" + e.fileProgresStr(),
		Column:  sbc.Progress,
	}
}

func (e *Editor) SetNumbers() {
	e.Textarea.ShowLineNumbers = true
	e.build()
}

func (e *Editor) SetNoNumbers() {
	e.Textarea.ShowLineNumbers = false
	e.build()
}

// moveCharacterLeft moves the cursor one character to the left
// and checks if the cursor is either at the end or the beginning
// of the line and saves it's position
func (e *Editor) MoveCharacterLeft() message.StatusBarMsg {
	e.Textarea.CharacterLeft(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = false
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// moveCharacterRight moves the cursor one character to the right
// and checks if the cursor is either at the end or the beginning
// of the line and saves its position
func (e *Editor) MoveCharacterRight() message.StatusBarMsg {
	e.Textarea.CharacterRight(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = false
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// inserAfter enters insert mode one character after the current cursor's
// position and saves its position
func (e *Editor) InsertAfter() message.StatusBarMsg {
	e.Textarea.CharacterRight(true)
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// insertLineStart moves the cursor to the beginning of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) InsertLineStart() message.StatusBarMsg {
	e.Textarea.CursorInputStart()
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// insertLineEnd moves the cursor to the end of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) InsertLineEnd() message.StatusBarMsg {
	e.Textarea.CursorEnd()
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// insertLineAbove creates and empty line above the current line
// and enters insert mode
func (e *Editor) InsertLineAbove() message.StatusBarMsg {
	e.Textarea.CursorUp()
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.EnterInsertMode(true)
	return message.StatusBarMsg{}
}

// insertLineBelow creates and empty line below the current line
// and enters insert mode
func (e *Editor) InsertLineBelow() message.StatusBarMsg {
	e.Textarea.CursorEnd()
	e.Textarea.InsertRune('\n')
	e.Textarea.RepositionView()
	e.EnterInsertMode(true)
	return message.StatusBarMsg{}
}

// LineUp moves the cursor one line up and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (e *Editor) LineUp() message.StatusBarMsg {
	e.Textarea.CursorUp()
	e.Textarea.RepositionView()

	pos := e.CurrentBuffer.CursorPos
	// if we have a wrapped line we skip the wrapped part of the line
	if pos.Row == e.Textarea.CursorPos().Row &&
		e.Textarea.Line() > 0 {
		// e.Textarea.CursorUp() doesn't work properly on some occasions
		// so I'm gonna be a little dirty
		e.LineUp()
	}

	e.Textarea.SetCursorColumn(pos.ColumnOffset)
	e.saveCursorRow()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorLineVimEnd()
	}

	if e.Vim.Mode.Current == mode.Visual {
		return e.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

// LineDown moves the cursor one line down and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (e *Editor) LineDown() message.StatusBarMsg {
	e.Textarea.CursorDown()
	e.Textarea.RepositionView()

	pos := e.CurrentBuffer.CursorPos
	// If we have a wrapped line we skip the wrapped part of the line
	if pos.Row == e.Textarea.CursorPos().Row &&
		e.Textarea.Line() < e.Textarea.LineCount()-1 {
		// e.Textarea.CursorDown() doesn't work properly for some reason
		// so I'm gonna be a little dirty again
		e.LineDown()
	}

	e.Textarea.SetCursorColumn(pos.ColumnOffset)
	e.saveCursorRow()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorLineVimEnd()
	}

	if e.Vim.Mode.Current == mode.Visual {
		return e.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

// goToLineStart moves the cursor to the beginning of the line,
// sets isAtLineStart and saves the cursor position
func (e *Editor) GoToLineStart() message.StatusBarMsg {
	e.Textarea.CursorStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// goToInputStart moves the cursor to the first character of the line,
// checks if the cursor is at the beginning of the line
// and saves the cursor position
func (e *Editor) GoToInputStart() message.StatusBarMsg {
	e.Textarea.CursorInputStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// goToLineEnd moves the cursor to the end of the line, sets isAtLineEnd
// and saves the cursor position
func (e *Editor) GoToLineEnd() message.StatusBarMsg {
	e.Textarea.CursorLineVimEnd()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// goToTop moves the cursor to the beginning of the buffer
func (e *Editor) GoToTop() message.StatusBarMsg {
	e.Textarea.MoveToBegin()
	e.Textarea.RepositionView()
	return e.UpdateSelectedRowsCount()
}

// goToTop moves the cursor to the bottom of the buffer
func (e *Editor) GoToBottom() message.StatusBarMsg {
	e.Textarea.MoveToEnd()
	e.Textarea.RepositionView()
	return e.UpdateSelectedRowsCount()
}

// wordRightEnd moves the cursor to the end of the next word
func (e *Editor) WordRightEnd() message.StatusBarMsg {
	e.Textarea.CharacterRight(false)
	e.Textarea.WordRight()
	e.Textarea.CharacterLeft(false)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// WordRightStart moves the cursor to the beginning of the next word
func (e *Editor) WordRightStart() message.StatusBarMsg {
	e.Textarea.WordRight()
	e.Textarea.CharacterRight(false)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// WordBack moves the cursor to the beginning of the next word
func (e *Editor) WordBack() message.StatusBarMsg {
	e.Textarea.WordLeft()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

func (e *Editor) DownHalfPage() message.StatusBarMsg {
	e.Textarea.DownHalfPage()
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) UpHalfPage() message.StatusBarMsg {
	e.Textarea.UpHalfPage()
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) DeleteLine() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(e.Textarea.DeleteLine)
	e.updateHistoryEntry()
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) DeleteInnerWord(enterInsertMode bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		e.Textarea.DeleteInnerWord()
	})
	e.updateHistoryEntry()
	if enterInsertMode {
		e.Vim.Mode.Current = mode.Insert
	}
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) DeleteOuterWord(enterInsertMode bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		e.Textarea.DeleteOuterWord()
	})
	e.updateHistoryEntry()
	if enterInsertMode {
		e.Vim.Mode.Current = mode.Insert
	}
	return e.UpdateSelectedRowsCount()
}

func (e *Editor) DeleteAfterCursor() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(e.Textarea.DeleteAfterCursor)
	e.updateHistoryEntry()
	return e.ResetSelectedRowsCount()
}

func (e *Editor) DeleteNLines(lines int, up bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		e.Textarea.DeleteLines(lines, up)
	})
	e.updateHistoryEntry()
	return e.ResetSelectedRowsCount()
}

func (e *Editor) DeleteWordRight() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(e.Textarea.DeleteWordRight)
	e.updateHistoryEntry()
	return message.StatusBarMsg{}
}

func (e *Editor) MergeLineBelow() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		e.Textarea.VimMergeLineBelow(e.CurrentBuffer.CursorPos.Row)
	})
	e.updateHistoryEntry()
	return message.StatusBarMsg{}
}

// DeleteRune the rune that the cursor is currently on.
// If buffer is in visual mode it takes the selection into account
func (e *Editor) DeleteRune() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		c := e.CurrentBuffer.CursorPos
		char := ""

		if minRange, maxRange := e.Textarea.SelectionRange(); minRange.Row > -1 {
			char = e.Textarea.SelectionStr()
			e.Textarea.DeleteRunesInRange(minRange, maxRange)
		} else {
			char = e.Textarea.DeleteRune(c.Row, c.ColumnOffset)
		}

		e.Yank(char)
	})

	e.EnterNormalMode()
	return e.ResetSelectedRowsCount()
}

func (e *Editor) ResetSelectedRowsCount() message.StatusBarMsg {
	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.KeyInfo,
	}
}

func (e *Editor) UpdateSelectedRowsCount() message.StatusBarMsg {
	return message.StatusBarMsg{
		Content: strconv.Itoa(e.SelectedRowsCount()),
		Column:  sbc.KeyInfo,
	}
}

func (e *Editor) SelectedRowsCount() int {
	startRow := e.Textarea.Selection.StartRow
	cursorRow := e.Textarea.CursorPos().Row
	minRow := min(startRow, cursorRow)
	maxRow := max(startRow, cursorRow)

	return (maxRow - minRow) + 1
}

// undo sets the buffer content to the previous history entry
func (e *Editor) Undo() message.StatusBarMsg {
	val, cursorPos := e.CurrentBuffer.undo()
	e.Textarea.SetValue(val)
	e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
	e.Textarea.RepositionView()
	return message.StatusBarMsg{}
}

// redo sets the buffer content to the next history entry
func (e *Editor) Redo() message.StatusBarMsg {
	val, cursorPos := e.CurrentBuffer.redo()
	e.Textarea.SetValue(val)
	e.Textarea.MoveCursor(cursorPos.Row, cursorPos.ColumnOffset)
	e.Textarea.RepositionView()
	return message.StatusBarMsg{}
}

func (e *Editor) Yank(str string) message.StatusBarMsg {
	clipboard.WriteAll(str)
	return message.StatusBarMsg{}
}

func (e *Editor) YankSelection() message.StatusBarMsg {
	sel := e.Textarea.SelectionStr()
	clipboard.WriteAll(sel)
	e.Textarea.ResetSelection()
	e.EnterNormalMode()
	return message.StatusBarMsg{}
}

func (e *Editor) Paste() message.StatusBarMsg {
	e.newHistoryEntry()
	e.checkDirty(func() {
		if cnt, err := clipboard.ReadAll(); err == nil {
			e.Textarea.InsertString(cnt)
		}
	})
	e.updateHistoryEntry()
	e.Textarea.RepositionView()
	return message.StatusBarMsg{}
}
