package components

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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

	"github.com/charmbracelet/bubbles/v2/cursor"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"golang.design/x/clipboard"
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

type errMsg error

type Buffer struct {
	// Index is the index of the buffer
	Index int

	// CurrentLine is the index of line the cursor is currently on
	CurrentLine int

	// CurrentLineLength is the length of the current line
	CurrentLineLength int

	// CursorPos is the position of the cursor
	CursorPos textarea.CursorPos

	// path is the path of the buffer
	path    string
	Content string

	// History is the input history of the buffer per session
	History textarea.History

	// Dirty indicates whether the buffer has unsaved changes
	Dirty bool

	// LastSavedContent is the last saved content of the buffer
	LastSavedContent *string

	// Header is the title of the buffer
	// If not nil, the path as a breadcrumb is displayed
	header *string
}

// Name returns the name of the buffer without its suffix.
func (b *Buffer) Name() string {
	name := filepath.Base(b.Path(false))
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return name
}

// Path returns the path of the buffer.
// If encoded is true it returns a url path save for writing to a config file.
func (b *Buffer) Path(encoded bool) string {
	if encoded {
		p := &url.URL{
			Scheme: "file",
			Path:   filepath.ToSlash(b.path),
		}
		return p.String()
	}

	return b.path
}

func (b *Buffer) undo() (string, textarea.CursorPos) {
	return b.History.Undo()
}

func (b *Buffer) redo() (string, textarea.CursorPos) {
	return b.History.Redo()
}

type BuffersChangedMsg struct {
	Buffers *Buffers
}

func (e *Editor) SendBuffersChangedMsg() tea.Cmd {
	return func() tea.Msg {
		return BuffersChangedMsg{
			Buffers: e.Buffers,
		}
	}
}

type Buffers []Buffer

// Contains returns whether a buffer is in memory
func (b Buffers) Contains(path string) (*Buffer, bool, int) {
	for i := range b {
		if b[i].path == path {
			return &b[i], true, i
		}
	}
	return nil, false, 0
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

type Editor struct {
	Component

	// Buffers holds all the open buffers
	Buffers *Buffers

	// CurrentBuffer is the currently active buffer
	CurrentBuffer *Buffer

	// Textarea is the bubbletea textarea component
	Textarea textarea.Model

	// Vim holds the current vim mode
	Vim Vim

	// CanInsert indicates whether textarea can receive input
	// regardless of the current input mode
	CanInsert bool

	// isAtLineEnd indicates whether the cursor is at the end of the line
	isAtLineEnd bool

	// isAtLineStart indicates whether the cursor is at the beginning of the line
	isAtLineStart bool

	// StatusBarMsg is the message to be displayed in the status bar
	StatusBarMsg message.StatusBarMsg

	// ShowLineNumbers indicates whether to show line numbers
	ShowLineNumbers bool

	ListBuffers bool

	// conf indicates whether to show column numbers
	conf *config.Config

	err error

	LastOpenNoteLoaded bool
}

func NewEditor(conf *config.Config) *Editor {
	ta := textarea.New()
	ta.Prompt = ""
	ta.Styles.Focused.CursorLine = cursorLine
	ta.Styles.Focused.Base = focusedStyle
	ta.Styles.Blurred.Base = blurredStyle
	ta.CharLimit = charLimit
	ta.MaxHeight = maxHeight
	ta.Selection.Cursor.SetMode(cursor.CursorStatic)
	ta.Selection.Cursor.TextStyle = ta.SelectionStyle()
	ta.Selection.Cursor.Style = ta.SelectionStyle()

	editor := &Editor{
		Vim: Vim{
			Mode: mode.ModeInstance{Current: mode.Normal},
			Pending: Input{
				keyinput.Input{Ctrl: false, Alt: false},
				"",
				"",
			},
		},
		CanInsert:          false,
		Textarea:           ta,
		Component:          Component{},
		CurrentBuffer:      &Buffer{},
		isAtLineEnd:        false,
		isAtLineStart:      false,
		err:                nil,
		conf:               conf,
		LastOpenNoteLoaded: false,
	}

	editor.ShowLineNumbers = editor.LineNumbers()
	editor.Textarea.ShowLineNumbers = editor.ShowLineNumbers
	editor.Textarea.ResetSelection()

	if err := clipboard.Init(); err != nil {
		debug.LogErr(err)
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

	e.Textarea.Selection.Cursor.Blur()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch e.Vim.Mode.Current {
		case mode.Normal:
			e.saveCursorPosToConf()

		case mode.Insert:
			cmd = e.handleInsertMode(msg)

		case mode.Replace:
			cmd = e.handleReplaceMode(msg)

		case mode.Command:
			cmd = e.handleCommandMode(msg)
		}

		e.checkDirty()

	case tea.WindowSizeMsg:
		e.Size.Width = msg.Width
		e.Size.Height = msg.Height

		if !e.Ready {
			e.Ready = true
		}

	case errMsg:
		e.err = msg
		return e, nil
	}

	e.setTextareaSize()
	cmds = append(cmds, cmd)

	return e, tea.Batch(cmds...)
}

func (e *Editor) View() string {
	if !e.Focused() {
		e.Textarea.Blur()
	}

	return e.build()
}

func (e *Editor) build() string {
	var view strings.Builder
	view.WriteString(e.BuildHeader(e.Size.Width, false))
	view.WriteString(e.Textarea.View())
	return view.String()
}

func (e *Editor) RefreshSize() {
	e.setTextareaSize()
}

func (e *Editor) SetBuffers(b *Buffers) {
	e.Buffers = b
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
	cursorPos := e.cursorPosFromConf(path)

	buf := Buffer{
		Index:             len(*e.Buffers) + 1,
		path:              path,
		Content:           noteContent,
		History:           textarea.NewHistory(),
		CurrentLine:       0,
		CurrentLineLength: 0,
		CursorPos:         cursorPos,
	}

	*e.Buffers = append(*e.Buffers, buf)
	buffers := *e.Buffers
	e.CurrentBuffer = &buffers[len(buffers)-1]

	content := ""
	if e.CurrentBuffer.path == path {
		content = e.CurrentBuffer.Content
	}

	e.newHistoryEntry()

	e.CurrentBuffer.History.UpdateEntry(content, buf.CursorPos)
	e.CurrentBuffer.LastSavedContent = &content

	e.Textarea.SetValue(content)
	e.Textarea.MoveCursor(
		cursorPos.Row,
		cursorPos.RowOffset,
		cursorPos.ColumnOffset,
	)
	e.Textarea.RepositionView()

	e.saveLineLength()
	e.UpdateMetaInfo()

	return message.StatusBarMsg{}
}

// OpenBuffer attempts to open the buffer with the given path.
// If no buffer is found a new buffer is created
func (e *Editor) OpenBuffer(path string) message.StatusBarMsg {
	relPath := utils.RelativePath(path, true)
	icon := theme.Icon(theme.IconNote, e.conf.NerdFonts())

	statusMsg := message.StatusBarMsg{
		Content: icon + " " + relPath,
		Column:  sbc.FileInfo,
	}

	buf, exists, _ := e.Buffers.Contains(path)
	// create new buffer if we can't find anything
	if len(*e.Buffers) <= 0 || !exists {
		e.NewBuffer(path)
		return statusMsg
	}

	e.CurrentBuffer = buf

	e.Textarea.SetValue(buf.Content)
	e.Textarea.MoveCursor(
		buf.CursorPos.Row,
		buf.CursorPos.RowOffset,
		buf.CursorPos.ColumnOffset,
	)
	e.Textarea.RepositionView()

	e.saveLineLength()
	e.UpdateMetaInfo()

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
	path := e.CurrentBuffer.path
	relativePath := strings.ReplaceAll(path, rootDir+"/", "")
	bufContent := e.Textarea.Value()
	bytes, err := notes.Write(path, bufContent)

	if err != nil {
		debug.LogErr(err)
		return statusMsg
	}

	e.CurrentBuffer.Dirty = false

	resultMsg := fmt.Sprintf(
		message.StatusBar.FileWritten,
		relativePath, e.Textarea.LineCount(), bytes,
	)

	e.CurrentBuffer.LastSavedContent = &bufContent

	statusMsg.Content = resultMsg
	return statusMsg
}

func (e *Editor) DeleteCurrentBuffer() message.StatusBarMsg {
	return e.DeleteBuffer(e.CurrentBuffer.path)
}

// DeleteBuffer closes the currently active buffer or resets the editor if
// none is available
func (e *Editor) DeleteBuffer(path string) message.StatusBarMsg {
	if _, ok, index := e.Buffers.Contains(path); ok {
		*e.Buffers = slices.Delete(*e.Buffers, index, index+1)
	}

	buffers := *e.Buffers
	if len(buffers) > 0 {
		lastBuf := buffers[len(buffers)-1]
		e.OpenBuffer(lastBuf.path)
	} else {
		e.reset()
	}

	return message.StatusBarMsg{}
}

func (e *Editor) DeleteAllBuffers() message.StatusBarMsg {
	e.reset()
	return message.StatusBarMsg{}
}

// DirtyBuffers collects all the dirty, dirty buffers
func (e *Editor) DirtyBuffers() Buffers {
	dirty := make(Buffers, 0)
	buffers := *e.Buffers

	for i := range buffers {
		if buffers[i].Dirty {
			dirty = append(dirty, buffers[i])
		}
	}

	return dirty
}

// BuildHeader builds title of the editor column
func (e *Editor) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if e.CurrentBuffer.header != nil && !rebuild {
		if width == lipgloss.Width(*e.CurrentBuffer.header) {
			return *e.CurrentBuffer.header
		}
	}

	title := "EDITOR"
	if e.CurrentBuffer.path != "" {
		title = e.breadcrumb()
	}

	header := theme.Header(title, width, e.Focused()) + "\n"
	e.CurrentBuffer.header = &header
	return header
}

func (e *Editor) OpenLastNotes() {
	lastNotes, lastNotesErr := e.conf.MetaValue("", config.LastNotes)
	lastNote, err := e.conf.MetaValue("", config.LastOpenNote)

	if lastNotesErr == nil && lastNotes != "" {
		for n := range strings.SplitSeq(lastNotes, ",") {
			e.OpenBuffer(utils.PathFromUrl(n))
		}
	}

	if err == nil && lastNote != "" {
		e.OpenBuffer(utils.PathFromUrl(lastNote))
	}
}

func (e *Editor) Focused() bool {
	return e.focused
}

func (e *Editor) SetFocus(focus bool) {
	e.focused = focus
	if !e.Textarea.Focused() {
		e.Textarea.Focus()
	}
}

func (e *Editor) breadcrumb() string {
	noteName := e.CurrentBuffer.Name()
	pathSeparator := string(os.PathSeparator)
	breadcrumbSeparator := " › "

	p := filepath.Dir(e.CurrentBuffer.Path(false))
	relPath := utils.RelativePath(p, false)
	breadcrumb := strings.ReplaceAll(relPath, pathSeparator, breadcrumbSeparator)

	iconDir := theme.Icon(theme.IconDirClosed, e.conf.NerdFonts())
	iconNote := theme.Icon(theme.IconNote, e.conf.NerdFonts())

	return iconDir + breadcrumb + breadcrumbSeparator + iconNote + " " + noteName
}

// reset puts the editor to default by clearing the textarea, resetting the
// meta value for current note and deleting the current buffer
func (e *Editor) reset() {
	e.Textarea.SetValue("")
	e.conf.SetMetaValue("", config.LastOpenNote, "")
	e.conf.SetMetaValue("", config.LastNotes, "")
	e.CurrentBuffer = &Buffer{}
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

	// We need to remember if the cursor is at the and of the line
	// so that lineup and linedown moves the cursor to the end
	// when it's supposed to do so
	isInsertMode := e.Vim.Mode.Current == mode.Insert
	e.isAtLineEnd = false
	if e.Textarea.IsExceedingLine() {
		e.Textarea.CursorLineVimEnd()
		e.isAtLineEnd = true
	} else if isInsertMode {
		e.MoveCharacterLeft()
	}

	if e.Vim.Mode.IsAnyVisual() {
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
	e.Textarea.ResetSelection()
	return message.StatusBarMsg{}
}

// EnterReplaceMode sets the current editor mode to replace
// and creates a new history entry
func (e *Editor) EnterReplaceMode() message.StatusBarMsg {
	e.Vim.Mode.Current = mode.Replace
	e.newHistoryEntry()
	e.Textarea.SetCursorColor(mode.Replace.Colour())
	return message.StatusBarMsg{}
}

// EnterVisualMode sets the current editor mode to replace
// and creates a new history entry
func (e *Editor) EnterVisualMode(
	selectionMode textarea.SelectionMode,
) message.StatusBarMsg {
	e.Textarea.StartSelection(selectionMode)

	vimMode := mode.Visual
	if selectionMode == textarea.SelectVisualLine {
		vimMode = mode.VisualLine
	}

	e.Vim.Mode.Current = vimMode
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
// func (e *Editor) checkDirty(fn func()) bool {
func (e *Editor) checkDirty() bool {
	if saved := e.CurrentBuffer.LastSavedContent; saved != nil {
		isDirty := e.Textarea.Value() != *saved
		e.CurrentBuffer.Dirty = isDirty
		return isDirty
	}

	return false
}

func (e *Editor) fileProgress() int {
	pc := float32(e.Textarea.Line()+1) / float32(e.Textarea.LineCount())
	return int(pc * 100.0)
}

func (e *Editor) fileProgresStr() string {
	var p strings.Builder
	p.WriteString(strconv.Itoa(e.fileProgress()))
	p.WriteByte('%')
	return p.String()
}

func (e *Editor) cursorInfo() string {
	var info strings.Builder
	info.WriteString(strconv.Itoa(e.Textarea.Line() + 1))
	info.WriteByte(',')
	info.WriteString(strconv.Itoa(e.Textarea.LineInfo().ColumnOffset))
	return info.String()
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
	if e.Textarea.Width() != e.Size.Width && e.Textarea.Height() != e.Size.Height {
		const reserverdLines = 1
		e.Textarea.SetWidth(e.Size.Width)
		e.Textarea.SetHeight(e.Size.Height - reserverdLines)
	}
}

// saveCursorPos saves the cursors current column offset and row
func (e *Editor) saveCursorPos() {
	e.CurrentBuffer.CursorPos = e.Textarea.CursorPos()
}

// saveCursorRow saves the cursors current row
func (e *Editor) saveCursorRow() {
	e.CurrentBuffer.CursorPos.Row = e.Textarea.CursorPos().Row
	e.CurrentBuffer.CursorPos.RowOffset = e.Textarea.CursorPos().RowOffset
}

// saveLineLength stores the length of the current line
func (e *Editor) saveLineLength() {
	e.CurrentBuffer.CurrentLineLength = e.Textarea.LineLength(-1)
}

// saveCursorCol saves the cursors current column offset
//func (e *Editor) saveCursorCol() {
//	e.CurrentBuffer.CursorPos.ColumnOffset = e.Textarea.CursorPos().ColumnOffset
//}

func (e *Editor) UpdateStatusBarInfo() {
	e.StatusBarMsg = e.StatusBarInfo()
}

func (e *Editor) StatusBarInfo() message.StatusBarMsg {
	var info strings.Builder
	info.WriteString(e.cursorInfo())
	info.WriteRune('\t')
	info.WriteString(e.fileProgresStr())

	return message.StatusBarMsg{
		Content: info.String(),
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

// OpenConfig opens the config file as a buffer
func (e *Editor) OpenConfig() message.StatusBarMsg {
	if configFile, err := app.ConfigFile(false); err == nil {
		return e.OpenBuffer(configFile)
	}
	return message.StatusBarMsg{}
}

// MoveCharacterLeft moves the cursor one character to the left
// and checks if the cursor is either at the end or the beginning
// of the line and saves it's position
func (e *Editor) MoveCharacterLeft() message.StatusBarMsg {
	e.Textarea.CharacterLeft(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = false
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// MoveCharacterRight moves the cursor one character to the right
// and checks if the cursor is either at the end or the beginning
// of the line and saves its position
func (e *Editor) MoveCharacterRight() message.StatusBarMsg {
	e.Textarea.CharacterRight(false)
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = false
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// InsertAfter enters insert mode one character after the current cursor's
// position and saves its position
func (e *Editor) InsertAfter() message.StatusBarMsg {
	e.Textarea.CharacterRight(true)
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// InsertLineStart moves the cursor to the beginning of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) InsertLineStart() message.StatusBarMsg {
	e.Textarea.CursorInputStart()
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// InsertLineEnd moves the cursor to the end of the line,
// enters insert mode and saves the cursor's position
func (e *Editor) InsertLineEnd() message.StatusBarMsg {
	e.Textarea.CursorEnd()
	e.EnterInsertMode(true)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// InsertLineAbove creates and empty line above the current line
// and enters insert mode
func (e *Editor) InsertLineAbove() message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.EmptyLineAbove()
	e.EnterInsertMode(false)
	return message.StatusBarMsg{}
}

// InsertLineBelow creates and empty line below the current line
// and enters insert mode
func (e *Editor) InsertLineBelow() message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.EmptyLineBelow()
	e.EnterInsertMode(false)
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
	e.saveLineLength()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorLineVimEnd()
	}

	if e.Vim.Mode.IsAnyVisual() {
		return e.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

func (e *Editor) MultiLineUp() message.StatusBarMsg {
	e.Textarea.CursorUp()
	e.Textarea.RepositionView()
	e.saveCursorRow()
	e.saveLineLength()

	if e.Vim.Mode.IsAnyVisual() {
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
	e.saveLineLength()

	if e.Textarea.IsExceedingLine() || e.isAtLineEnd {
		e.Textarea.CursorLineVimEnd()
	}

	if e.Vim.Mode.IsAnyVisual() {
		return e.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

func (e *Editor) MultiLineDown() message.StatusBarMsg {
	e.Textarea.CursorDown()
	e.Textarea.RepositionView()
	e.saveCursorRow()
	e.saveLineLength()

	if e.Vim.Mode.IsAnyVisual() {
		return e.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

// GoToLineStart moves the cursor to the beginning of the line,
// sets isAtLineStart and saves the cursor position
func (e *Editor) GoToLineStart() message.StatusBarMsg {
	e.Textarea.CursorStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToInputStart moves the cursor to the first character of the line,
// checks if the cursor is at the beginning of the line
// and saves the cursor position
func (e *Editor) GoToInputStart() message.StatusBarMsg {
	e.Textarea.CursorInputStart()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToLineEnd moves the cursor to the end of the line, sets isAtLineEnd
// and saves the cursor position
func (e *Editor) GoToLineEnd() message.StatusBarMsg {
	e.Textarea.CursorLineVimEnd()
	e.isAtLineStart = e.Textarea.IsAtLineStart()
	e.isAtLineEnd = e.Textarea.IsAtLineEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToTop moves the cursor to the beginning of the buffer
func (e *Editor) GoToTop() message.StatusBarMsg {
	e.Textarea.MoveToBegin()
	e.Textarea.RepositionView()
	return e.UpdateSelectedRowsCount()
}

// GoToBottom moves the cursor to the bottom of the buffer
func (e *Editor) GoToBottom() message.StatusBarMsg {
	e.Textarea.MoveToEnd()
	e.Textarea.RepositionView()
	return e.UpdateSelectedRowsCount()
}

// WordRightEnd moves the cursor to the end of the next word
func (e *Editor) WordRightEnd() message.StatusBarMsg {
	e.Textarea.WordRightEnd()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// WordRightStart moves the cursor to the beginning of the next word
func (e *Editor) WordRightStart() message.StatusBarMsg {
	e.Textarea.WordRight()
	//e.Textarea.CharacterRight(false)
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// WordBack moves the cursor to the beginning of the next word
func (e *Editor) WordBack() message.StatusBarMsg {
	e.Textarea.WordLeft()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// DownHalfPage moves the cursor down half a page
func (e *Editor) DownHalfPage() message.StatusBarMsg {
	e.Textarea.DownHalfPage()
	return e.UpdateSelectedRowsCount()
}

// UpHalfPage moves the cursor up half a page
func (e *Editor) UpHalfPage() message.StatusBarMsg {
	e.Textarea.UpHalfPage()
	return e.UpdateSelectedRowsCount()
}

// SelectInnerWord selects the inner word.
// Only effective if we're in visual mode
func (e *Editor) SelectInnerWord() message.StatusBarMsg {
	e.Textarea.SelectInnerWord()
	return e.UpdateSelectedRowsCount()
}

// SelectOuterWord selects the outer word.
// Only effective if we're in visual mode
func (e *Editor) SelectOuterWord() message.StatusBarMsg {
	e.Textarea.SelectOuterWord()
	return e.UpdateSelectedRowsCount()
}

// DeleteLine deletes the current line and copies its content
// to the clipboard
func (e *Editor) DeleteLine() message.StatusBarMsg {
	e.newHistoryEntry()

	e.saveLineLength()
	e.YankLine()
	e.Textarea.DeleteLine()
	e.updateHistoryEntry()

	return e.ResetSelectedRowsCount()
}

// DeleteInnerWord deletes the word the cursor is on
// If enterInsertMode is true, we're going straight into inser mode
func (e *Editor) DeleteInnerWord(enterInsertMode bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.DeleteInnerWord()
	e.updateHistoryEntry()

	if enterInsertMode {
		e.EnterInsertMode(false)
	}

	return e.ResetSelectedRowsCount()
}

// DeleteOuterWord deletes the word the cursor is on including the trailing
// whitespace
// If enterInsertMode is true, we're going straight into inser mode
func (e *Editor) DeleteOuterWord(enterInsertMode bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.DeleteOuterWord()
	e.updateHistoryEntry()

	if enterInsertMode {
		e.EnterInsertMode(false)
	}

	return e.UpdateSelectedRowsCount()
}

// DeleteAfterCursor deletes all characters after the cursor
func (e *Editor) DeleteAfterCursor(overshoot bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.DeleteAfterCursor(overshoot)
	e.updateHistoryEntry()
	return e.ResetSelectedRowsCount()
}

// DeleteNLines deletes n lines
func (e *Editor) DeleteNLines(lines int, up bool) message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.DeleteLines(lines, up)
	e.updateHistoryEntry()
	e.Textarea.RepositionView()
	return e.ResetSelectedRowsCount()
}

// DeleteWordRight deletes the rest of word after the cursor
func (e *Editor) DeleteWordRight() message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.DeleteWordRight()
	e.updateHistoryEntry()
	return message.StatusBarMsg{}
}

// MergeLineBelow merges the current line with the line below
func (e *Editor) MergeLineBelow() message.StatusBarMsg {
	e.newHistoryEntry()
	e.Textarea.VimMergeLineBelow(e.CurrentBuffer.CursorPos.Row)
	e.updateHistoryEntry()
	return message.StatusBarMsg{}
}

// DeleteRune the rune that the cursor is currently on.
// If buffer is in visual mode it takes the selection into account
// If keepMode is true this method doesn't enter normal mode
func (e *Editor) DeleteRune(keepMode bool, withHistory bool) message.StatusBarMsg {
	if withHistory {
		e.newHistoryEntry()
	}

	c := e.CurrentBuffer.CursorPos
	char := ""

	if minRange, maxRange := e.Textarea.SelectionRange(); minRange.Row > -1 {
		char = e.Textarea.SelectionStr()
		if e.Textarea.Selection.Mode == textarea.SelectVisualLine {
			e.Textarea.DeleteSelectedLines()
		} else {
			e.Textarea.DeleteRunesInRange(minRange, maxRange)
		}
	} else {
		char = e.Textarea.DeleteRune(c.Row, c.ColumnOffset)
	}

	e.Yank(char)

	if !keepMode {
		e.EnterNormalMode()
	}
	e.Textarea.RepositionView()
	return e.ResetSelectedRowsCount()
}

// ResetSelectedRowsCount resets the selected rows count in the status bar
func (e *Editor) ResetSelectedRowsCount() message.StatusBarMsg {
	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.KeyInfo,
	}
}

// UpdateSelectedRowsCount updates the selected rows count in the status bar
func (e *Editor) UpdateSelectedRowsCount() message.StatusBarMsg {
	if e.Vim.Mode.IsAnyVisual() {
		return message.StatusBarMsg{
			Content: strconv.Itoa(e.SelectedRowsCount()),
			Column:  sbc.KeyInfo,
		}
	}
	return message.StatusBarMsg{}
}

// SelectedRowsCount returns the number of selected rows
func (e *Editor) SelectedRowsCount() int {
	startRow := e.Textarea.Selection.StartRow
	cursorRow := e.Textarea.CursorPos().Row
	minRow := min(startRow, cursorRow)
	maxRow := max(startRow, cursorRow)

	return (maxRow - minRow) + 1
}

// Undo sets the buffer content to the previous history entry
func (e *Editor) Undo() message.StatusBarMsg {
	val, cursorPos := e.CurrentBuffer.undo()
	curBuf := e.CurrentBuffer

	// dirty check
	curBuf.Dirty = val != *curBuf.LastSavedContent
	e.Textarea.SetValue(val)

	entryIndex := curBuf.History.EntryIndex
	// EntryIndex 0 means the time in the buffer history where the buffer was
	// opened to get the initial content of the buffer.
	// We don't want to move the cursor there - just accept it.
	if entryIndex == 0 {
		cursorPos = curBuf.CursorPos
		if entry := curBuf.History.Entry(entryIndex + 1); entry != nil {
			cursorPos = entry.UndoCursorPos
		}
	}

	e.Textarea.MoveCursor(
		cursorPos.Row,
		cursorPos.RowOffset,
		cursorPos.ColumnOffset,
	)
	e.Textarea.RepositionView()
	e.saveCursorPos()
	return message.StatusBarMsg{}
}

// Redo sets the buffer content to the next history entry
func (e *Editor) Redo() message.StatusBarMsg {
	val, cursorPos := e.CurrentBuffer.redo()
	// dirty check
	e.CurrentBuffer.Dirty = val != *e.CurrentBuffer.LastSavedContent
	e.Textarea.SetValue(val)
	e.Textarea.MoveCursor(
		cursorPos.Row,
		cursorPos.RowOffset,
		cursorPos.ColumnOffset,
	)
	e.Textarea.RepositionView()
	return message.StatusBarMsg{}
}

// Yank copies the given string to the clipboard
func (e *Editor) Yank(str string) message.StatusBarMsg {
	clipboard.Write(clipboard.FmtText, []byte(str))
	return message.StatusBarMsg{}
}

// YankSelection copies the current selection to the clipboard.
// If keepCursorPos is true the cursor position remains the same
// otherwise the cursor is moved to the beginning of the selection
func (e *Editor) YankSelection(keepCursorPos bool) message.StatusBarMsg {
	sel := e.Textarea.SelectionStr()
	clipboard.Write(clipboard.FmtText, []byte(sel))
	buf := e.CurrentBuffer

	if keepCursorPos {
		e.Textarea.SetCursorColumn(e.CurrentBuffer.CursorPos.ColumnOffset)
	} else {
		startRow := e.Textarea.Selection.StartRow
		startCol := e.Textarea.Selection.StartCol
		if e.Vim.Mode.Current == mode.VisualLine {
			startCol = 0
		}
		// move the cursor to the beginning of the selection
		e.Textarea.MoveCursor(startRow, buf.CursorPos.RowOffset, startCol)
	}

	e.EnterNormalMode()
	return message.StatusBarMsg{}
}

// YankLine copies the current line to the clipboard
func (e *Editor) YankLine() message.StatusBarMsg {
	e.saveCursorPos()
	row := e.CurrentBuffer.CursorPos.Row
	e.Textarea.SelectRange(
		textarea.SelectVisualLine,
		textarea.CursorPos{Row: row, ColumnOffset: 0},
		textarea.CursorPos{Row: row, ColumnOffset: e.Textarea.LineLength(-1)},
	)
	return e.YankSelection(true)
}

// YankInnerWord copies the current word to the clipboard
func (e *Editor) YankInnerWord() message.StatusBarMsg {
	e.Textarea.SelectInnerWord()
	return e.YankSelection(false)
}

// YankOuterWord copies the current word to the clipboard including the
// trailing whitespace
func (e *Editor) YankOuterWord() message.StatusBarMsg {
	e.Textarea.SelectOuterWord()
	return e.YankSelection(false)
}

// Paste pastes the clipboard content.
// If the selection exceeds the length of the current line
// it attempts to paste the clipboard content on a newline below
// the current line
func (e *Editor) Paste() message.StatusBarMsg {
	cp := clipboard.Read(clipboard.FmtText)
	if len(cp) > 0 {
		cnt := string(cp)
		// save the curren cursor position to adjust the correct position
		// after the clipboard content is pasted
		cursorPos := e.CurrentBuffer.CursorPos
		col := cursorPos.ColumnOffset
		row := cursorPos.Row
		rowOffset := cursorPos.RowOffset

		lineLen := e.CurrentBuffer.CurrentLineLength

		e.newHistoryEntry()

		// insert clipboard content on a newline below if it's larger than
		// than the current line
		if len(cp) >= lineLen {
			e.Textarea.EmptyLineBelow()

			// strip the last new line since we've already inserted
			// an empty line so we don't need it
			// otherwise it would produce an additional empty line
			cnt = strings.TrimRight(cnt, "\n")

			// set the cursor position at the beginning of the next row
			// which is the newly pasted content
			col = 0
			row++
		} else {
			// if the clipboard content is not a full line set the
			// add the length of the selection to the current column offset
			// to set the cursor to the end of the selection
			col += len(cp)
			e.Textarea.CharacterRight(false)
		}

		// insert clipboard content
		e.Textarea.InsertString(cnt)
		e.Textarea.MoveCursor(row, rowOffset, col)
		e.updateHistoryEntry()
		e.Textarea.RepositionView()
	}
	return message.StatusBarMsg{}
}

// cursorPosFromConf retrieves the cursor position of the given note
// from the meta config file.
// If the meta config value is invalid it returns the empty CursorPos which
// equals the beginning of the file
func (e *Editor) cursorPosFromConf(filepath string) textarea.CursorPos {
	cursorPos := textarea.CursorPos{}
	pos, err := e.conf.MetaValue(filepath, config.CursorPosition)

	if err == nil {
		p := strings.Split(pos, ",")

		if len(p) != 3 {
			return cursorPos
		}

		row, _ := strconv.Atoi(p[0])
		rowOffset, _ := strconv.Atoi(p[1])
		col, _ := strconv.Atoi(p[2])

		cursorPos = textarea.CursorPos{
			Row:          row,
			RowOffset:    rowOffset,
			ColumnOffset: col,
		}
	}

	return cursorPos
}

// saveCursorPosToConf saves the current cursor position to the config file
func (e *Editor) saveCursorPosToConf() {
	pos := e.Textarea.CursorPos()

	e.conf.SetMetaValue(
		e.CurrentBuffer.path,
		config.CursorPosition,
		pos.String(),
	)
}

// UpdateMetaInfo records the current state of the editor by updating
// metadata values for recently opened notes and the currently opened note.
func (e *Editor) UpdateMetaInfo() {
	notePaths := make([]string, 0, len(*e.Buffers))

	for _, buf := range *e.Buffers {
		notePaths = append(notePaths, buf.Path(true))
	}

	noteStr := strings.Join(notePaths[:], ",")

	e.conf.SetMetaValue("", config.LastNotes, noteStr)
	e.conf.SetMetaValue("", config.LastOpenNote, e.CurrentBuffer.Path(true))
}

// LineNumbers returns whether line numbers are enabled in the config file
func (e *Editor) LineNumbers() bool {
	n, err := e.conf.Value(config.Editor, config.ShowLineNumbers)

	if err != nil {
		return false
	}

	number := n == "true"

	return number
}
