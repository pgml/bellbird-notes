package editor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/app/utils/clipboard"
	"bellbird-notes/tui/components/textarea"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

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

	// LastSavedContentHash is the hash of the last saved content of the buffer
	LastSavedContentHash string

	// header is the title of the buffer
	// If not nil, the path as a breadcrumb is displayed
	header *string

	// Writeable indicates whether the buffer can be written to
	Writeable bool

	// IsScratch indicates whether the buffer is a temporary scratch buffer
	IsScratch bool
}

// Name returns the name of the buffer without its suffix.
func (buf *Buffer) Name() string {
	name := filepath.Base(buf.Path(false))
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return name
}

// Path returns the buffer's path.
// If encoded is true it returns a file:// URL safe for writing to a config file.
func (buf *Buffer) Path(encoded bool) string {
	if encoded {
		p := &url.URL{
			Scheme: "file",
			Path:   filepath.ToSlash(buf.path),
		}
		return p.String()
	}
	return buf.path
}

// SetPath sets the buffer's path.
func (buf *Buffer) SetPath(path string) {
	buf.path = path
}

// undo applies the last undo patch, returning the restored content
// and cursor position.
func (buf *Buffer) undo() (string, textarea.CursorPos) {
	patch, hash, pos := buf.History.Undo()

	if patch != nil && hash != buf.hash() {
		restored, _ := buf.History.Dmp.PatchApply(patch, buf.Content)
		return restored, pos
	}

	return "", textarea.CursorPos{}
}

// redo reapplies the most recently undone change.
func (buf *Buffer) redo() (string, textarea.CursorPos) {
	patch, hash, pos := buf.History.Redo()

	if patch != nil && hash == buf.hash() {
		restored, _ := buf.History.Dmp.PatchApply(patch, buf.Content)
		return restored, pos
	}

	return "", textarea.CursorPos{}
}

// hash returns the hash of the buffer content
func (buf Buffer) hash() string {
	return utils.HashContent(buf.Content)
}

// BufferSavedMsg is sent when a buffer has been saved
type BufferSavedMsg struct {
	Buffer *Buffer
}

// SendBufferSavedMsg returns a command that sends a BufferSavedMsg
// containing the given buffer.
func SendBufferSavedMsg(buffer *Buffer) tea.Cmd {
	return func() tea.Msg {
		return BufferSavedMsg{
			Buffer: buffer,
		}
	}
}

// RefreshBufferMsg is sent when a buffer should be refreshed.
type RefreshBufferMsg struct {
	Path string
}

// SendRefreshBufferMsg returns a command that sends a RefreshBufferMsg
// for the specific path.
func SendRefreshBufferMsg(path string) tea.Cmd {
	return func() tea.Msg {
		return RefreshBufferMsg{
			Path: path,
		}
	}
}

// SwitchBufferMsg requests switching to a different buffer, optionally
// focusing the editor afterward
type SwitchBufferMsg struct {
	Path        string
	FocusEditor bool
}

// SendSwitchBufferMsg returns a command that sends a SwitchBufferMsg
// for the given path and focus setting
func SendSwitchBufferMsg(path string, focusEditor bool) tea.Cmd {
	return func() tea.Msg {
		return SwitchBufferMsg{
			Path:        path,
			FocusEditor: focusEditor,
		}
	}
}

// BuffersChangedMsg is sent when the set of buffers has changed.
type BuffersChangedMsg struct {
	Buffers *Buffers
}

// SendBuffersChangedMsg returns a commant that sends a BuffersChangedMsg
// containing the updated buffer list.
func SendBuffersChangedMsg(buffers *Buffers) tea.Cmd {
	return func() tea.Msg {
		return BuffersChangedMsg{
			Buffers: buffers,
		}
	}
}

type Buffers []Buffer

// Find returns the buffer if a buffer with the given path exists.
// It returns nil if no buffer could be found.
func (bufs Buffers) Find(path string) *Buffer {
	for i := range bufs {
		if bufs[i].path == path {
			return &bufs[i]
		}
	}
	return nil
}

// Contain returns whether the buffer with the given path exists.
func (bufs Buffers) Contain(path string) bool {
	buf := bufs.Find(path)
	return buf != nil
}

type Styles struct {
	focused lipgloss.Style
	blurred lipgloss.Style
}

type Editor struct {
	shared.Component

	// Buffers holds all the open buffers
	Buffers *Buffers

	// CurrentBuffer is the currently active buffer
	CurrentBuffer *Buffer

	// Textarea is the bubbletea textarea component
	Textarea textarea.Model

	// Holds the current vim mode
	Mode mode.ModeInstance

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

	// conf indicates whether to show column numbers
	conf *config.Config

	err error

	LastOpenNoteLoaded bool

	KeyInput keyinput.Input

	styles Styles
}

func New(title string, conf *config.Config) *Editor {
	theme := theme.New(conf)

	editor := &Editor{
		CanInsert:          false,
		Component:          shared.Component{},
		CurrentBuffer:      &Buffer{},
		isAtLineEnd:        false,
		isAtLineStart:      false,
		err:                nil,
		conf:               conf,
		LastOpenNoteLoaded: false,
	}

	editor.SetTitle(title)
	editor.SetTheme(theme)

	editor.ShowLineNumbers = editor.LineNumbers()
	editor.Textarea = editor.NewTextarea()
	editor.OnFocus = editor.onFocus
	editor.OnBlur = editor.onBlur

	if err := clipboard.Init(); err != nil {
		debug.LogErr(err)
	}

	return editor
}

// DefaultStyles returns the default styles for focused and blurred
// states for the editor.
func defaultStyles(t theme.Theme) Styles {
	style := lipgloss.NewStyle().
		Border(t.BorderStyle()).
		BorderTop(false).
		Padding(0, 1)

	return Styles{
		focused: style.BorderForeground(theme.ColourBorderFocused),
		blurred: style.BorderForeground(theme.ColourBorder),
	}
}

// NewTextarea returns a new textarea instance with default settings
func (editor Editor) NewTextarea() textarea.Model {
	styles := defaultStyles(editor.Theme())

	ta := textarea.New()
	ta.Prompt = ""
	ta.Styles.Focused.CursorLine = cursorLine
	ta.Styles.Focused.Base = styles.focused
	ta.Styles.Blurred.Base = styles.blurred

	ta.CharLimit = charLimit
	ta.MaxHeight = maxHeight
	ta.ShowLineNumbers = editor.ShowLineNumbers

	ta.Selection.Cursor.SetMode(cursor.CursorStatic)
	ta.Selection.Cursor.TextStyle = ta.SelectionStyle()
	ta.Selection.Cursor.Style = ta.SelectionStyle()

	ta.Search.IgnoreCase = editor.SearchIgnoreCase()
	ta.ResetSelection()

	return ta
}

// Init initialises the Model on program load.
// It partially implements the tea.Model interface.
func (editor *Editor) Init() tea.Cmd {
	return textarea.Blink
}

// Update is the Bubble Tea update loop.
func (editor *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	editor.Textarea.Selection.Cursor.Blur()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch editor.Mode.Current {
		case mode.Normal:
			cmd = editor.handleNormalMode(msg)

		case mode.Insert:
			cmd = editor.handleInsertMode(msg)

		case mode.Visual, mode.VisualLine, mode.VisualBlock:
			cmd = editor.handleVisualMode(msg)

		case mode.Replace:
			cmd = editor.handleReplaceMode(msg)

		case mode.Command:
			cmd = editor.handleCommandMode(msg)

		case mode.SearchPrompt, mode.Search:
			cmd = editor.handleSearchMode(msg)
		}

		editor.checkDirty()

	case tea.WindowSizeMsg:
		editor.Size.Width = msg.Width
		editor.Size.Height = msg.Height

		if !editor.IsReady {
			editor.IsReady = true
		}

	case errMsg:
		editor.err = msg
		return editor, nil

	case SearchMsg:
		caseOverride := strings.HasPrefix(msg.SearchTerm, "\\c")

		editor.Textarea.Search.IgnoreCase = editor.SearchIgnoreCase()

		if caseOverride {
			msg.SearchTerm = msg.SearchTerm[2:]
			editor.Textarea.Search.IgnoreCase = true
		}

		editor.Textarea.Search.Query = msg.SearchTerm

	case SearchConfirmedMsg:
		match := editor.Textarea.Search.FirstMatch()
		editor.Textarea.MoveCursor(match.Row, match.RowOffset, match.ColumnOffset)

	case RefreshBufferMsg:
		editor.BuildHeader(editor.Size.Width, true)
		editor.UpdateMetaInfo()

	case SwitchBufferMsg:
		if buf := editor.Buffers.Find(msg.Path); buf != nil {
			editor.SwitchBuffer(buf)
			editor.BuildHeader(editor.Size.Width, true)
		} else {
			editor.OpenBuffer(msg.Path)
		}
	}

	editor.RefreshSize()
	cmds = append(cmds, cmd)

	return editor, tea.Batch(cmds...)
}

// View renders the editor in its current state.
func (editor *Editor) View() tea.View {
	var view tea.View
	view.SetContent(editor.Content())
	return view
}

func (editor *Editor) Content() string {
	var view strings.Builder
	view.WriteString(editor.BuildHeader(editor.Size.Width, false))
	view.WriteString(editor.Textarea.View())

	return view.String()
}

// SetContent updates the textarea with the current buffer's content
// and sets the cursor to the last known position
func (editor *Editor) SetContent() {
	buf := editor.CurrentBuffer
	editor.Textarea.SetValue(buf.Content)
	editor.Textarea.MoveCursor(
		buf.CursorPos.Row,
		buf.CursorPos.RowOffset,
		buf.CursorPos.ColumnOffset,
	)
	editor.Textarea.RepositionView()
}

// RefreshSize update the textarea height and width to match
// the height and width of the editor
func (editor *Editor) RefreshSize() {
	if editor.Textarea.Width() != editor.Size.Width && editor.Textarea.Height() != editor.Size.Height {
		const reserverdLines = 1
		editor.Textarea.SetWidth(editor.Size.Width)
		editor.Textarea.SetHeight(editor.Size.Height - reserverdLines)
	}
}

func (editor *Editor) SetWidth(w int) {
	editor.Viewport.SetWidth(w)
	editor.Textarea.SetWidth(w)
	editor.Size.Width = w
}

func (editor *Editor) SetBuffers(b *Buffers) {
	editor.Buffers = b
}

// NewBuffer creates a new buffer, sets the textareas content
// and creates a new history for the buffer
func (editor *Editor) NewBuffer(path string) message.StatusBarMsg {
	note, err := os.ReadFile(path)

	if err != nil {
		debug.LogErr(err)
		return message.StatusBarMsg{Content: err.Error()}
	}

	noteContent := string(note)
	cursorPos := editor.cursorPosFromConf(path)

	// Create a new scratch buffer
	editor.NewScratchBuffer("", noteContent)

	// Fill scratch buffer with the note's data
	buf := editor.CurrentBuffer
	buf.IsScratch = false
	buf.path = path
	buf.CursorPos = cursorPos
	buf.History = textarea.NewHistory()
	buf.LastSavedContentHash = buf.hash()

	editor.SetContent()
	editor.saveLineLength()
	editor.UpdateMetaInfo()

	return message.StatusBarMsg{}
}

// NewScratchBuffer creates a new temporary buffer
func (editor *Editor) NewScratchBuffer(
	title string,
	content string,
) message.StatusBarMsg {
	notesRoot, _ := app.NotesRootDir()
	path := editor.scratchPath(notesRoot + "/" + title)

	buf := Buffer{
		Index:                len(*editor.Buffers) + 1,
		path:                 path,
		Content:              content,
		History:              textarea.NewHistory(),
		CurrentLine:          0,
		CurrentLineLength:    0,
		LastSavedContentHash: "",
		Writeable:            true,
		IsScratch:            true,
	}

	*editor.Buffers = append(*editor.Buffers, buf)
	buffers := *editor.Buffers
	editor.CurrentBuffer = &buffers[len(buffers)-1]

	return message.StatusBarMsg{}
}

// OpenBuffer attempts to open the buffer with the given path.
// If no buffer is found a new buffer is created
func (editor *Editor) OpenBuffer(path string) message.StatusBarMsg {
	relPath := utils.RelativePath(path, true)
	icon := theme.Icon(theme.IconNote, editor.conf.NerdFonts())

	statusMsg := message.StatusBarMsg{
		Content: icon + " " + relPath,
		Column:  sbc.FileInfo,
	}

	buf := editor.Buffers.Find(path)

	// create new buffer if we can't find anything
	if len(*editor.Buffers) <= 0 || buf == nil {
		editor.NewBuffer(path)
		return statusMsg
	}

	editor.CurrentBuffer = buf

	editor.SetContent()
	editor.saveLineLength()
	editor.UpdateMetaInfo()

	return statusMsg
}

// SwitchBuffer replaces the current editor view with the content of
// the given buffer
func (editor *Editor) SwitchBuffer(buf *Buffer) message.StatusBarMsg {
	if !editor.Buffers.Contain(buf.path) {
		return message.StatusBarMsg{}
	}

	editor.CurrentBuffer = buf

	editor.SetContent()
	editor.Textarea.RepositionView()
	editor.saveLineLength()
	editor.UpdateMetaInfo()

	if !editor.Focused() {
		editor.Focus()
	}

	return message.StatusBarMsg{}
}

// CheckTime re-reads the current buffer's content from file and
// updates the textarea
func (editor *Editor) CheckTime() {
	buf := editor.CurrentBuffer
	note, err := os.ReadFile(buf.path)

	if err != nil {
		debug.LogErr(err)
		return
	}

	content := string(note)
	buf.Content = content
	buf.LastSavedContentHash = buf.hash()

	editor.SetContent()
	editor.updateHistoryEntry()
}

// SaveBuffer writes the current buffer's content to the corresponding
// file on the disk and resets the dirty state
func (editor *Editor) SaveBuffer() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{
		Type:   message.Success,
		Column: sbc.General,
	}

	buf := editor.CurrentBuffer
	rootDir, _ := app.NotesRootDir()

	path := buf.path
	relativePath := strings.ReplaceAll(path, rootDir+"/", "")
	bufContent := editor.Textarea.Value()
	forceCreate := buf.IsScratch

	bytes, err := notes.Write(path, bufContent, forceCreate)

	if err != nil {
		debug.LogErr(err)
		return statusMsg
	}

	buf.Dirty = false

	resultMsg := fmt.Sprintf(
		message.StatusBar.FileWritten,
		relativePath, editor.Textarea.LineCount(), bytes,
	)

	buf.LastSavedContentHash = buf.hash()

	statusMsg.Content = resultMsg
	statusMsg.Cmd = SendBufferSavedMsg(editor.CurrentBuffer)

	return statusMsg
}

// DeleteBuffer closes the currently active buffer or resets the editor if
// none is available
func (editor *Editor) DeleteBuffer(path string) message.StatusBarMsg {
	if buf := editor.Buffers.Find(path); buf != nil {
		index := buf.Index - 1
		*editor.Buffers = slices.Delete(*editor.Buffers, index, index+1)
	}

	buffers := *editor.Buffers
	if len(buffers) > 0 {
		lastBuf := buffers[len(buffers)-1]
		editor.OpenBuffer(lastBuf.path)
	} else {
		editor.reset()
	}

	return message.StatusBarMsg{}
}

func (editor *Editor) DeleteCurrentBuffer() message.StatusBarMsg {
	return editor.DeleteBuffer(editor.CurrentBuffer.path)
}

func (editor *Editor) DeleteAllBuffers() message.StatusBarMsg {
	editor.reset()
	return message.StatusBarMsg{}
}

// DirtyBuffers collects all the dirty, dirty buffers
func (editor *Editor) DirtyBuffers() Buffers {
	dirty := make(Buffers, 0)
	buffers := *editor.Buffers

	for i := range buffers {
		if buffers[i].Dirty {
			dirty = append(dirty, buffers[i])
		}
	}

	return dirty
}

// BuildHeader builds title of the editor column
func (editor *Editor) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if editor.CurrentBuffer.header != nil && !rebuild {
		if width == lipgloss.Width(*editor.CurrentBuffer.header) {
			return *editor.CurrentBuffer.header
		}
	}

	title := editor.Title()
	if editor.CurrentBuffer.path != "" {
		title = editor.breadcrumb()
	}

	theme := editor.Theme()
	header := theme.Header(title, width, editor.Focused()) + "\n"
	editor.CurrentBuffer.header = &header
	return header
}

func (editor *Editor) OpenLastNotes() {
	lastNotes, lastNotesErr := editor.conf.MetaValue("", config.LastNotes)
	lastNote, err := editor.conf.MetaValue("", config.LastOpenNote)

	if lastNotesErr == nil && lastNotes != "" {
		for n := range strings.SplitSeq(lastNotes, ",") {
			editor.OpenBuffer(utils.PathFromUrl(n))
		}
	}

	if err == nil && lastNote != "" {
		editor.OpenBuffer(utils.PathFromUrl(lastNote))
	}
}

// scratchPath returns a valid path for a scratch note.
// If the path already exists as a physical note or is already virtually
// present as a buffer it appends "Copy" to the last found scratch name
func (editor *Editor) scratchPath(path string) string {
	path = notes.GetValidPath(path, true)

	if buf := editor.Buffers.Find(path); buf != nil {
		path = filepath.Dir(buf.Path(false)) + "/" + buf.Name() + " Copy" + notes.Ext
	}

	if editor.Buffers.Contain(path) {
		path = editor.scratchPath(path)
	}

	return path
}

func (editor *Editor) onFocus() {
	if !editor.Textarea.Focused() {
		editor.Textarea.Focus()
	}
}

func (editor *Editor) onBlur() {
	if !editor.Focused() {
		editor.Textarea.Blur()
	}
}

func (editor *Editor) breadcrumb() string {
	noteName := editor.CurrentBuffer.Name()
	pathSeparator := string(os.PathSeparator)
	breadcrumbSeparator := " › "

	p := filepath.Dir(editor.CurrentBuffer.Path(false))
	relPath := utils.RelativePath(p, false)
	breadcrumb := strings.ReplaceAll(relPath, pathSeparator, breadcrumbSeparator)

	iconDir := theme.Icon(theme.IconDirClosed, editor.conf.NerdFonts())
	iconNote := theme.Icon(theme.IconNote, editor.conf.NerdFonts())

	return iconDir + breadcrumb + breadcrumbSeparator + iconNote + " " + noteName
}

// reset puts the editor to default by clearing the textarea, resetting the
// meta value for current note and deleting the current buffer
func (editor *Editor) reset() {
	editor.Textarea.SetValue("")
	editor.conf.SetMetaValue("", config.LastOpenNote, "")
	editor.conf.SetMetaValue("", config.LastNotes, "")
	editor.CurrentBuffer = &Buffer{}
}

// EnterNormalMode sets the current editor mode to normal,
// checks if the cursor position exceeds the line length and
// saves the cursor position.
// It also updates the current history entry
func (editor *Editor) EnterNormalMode(withHistory bool) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{
		Content: "",
		Column:  sbc.General,
	}

	// We need to remember if the cursor is at the and of the line
	// so that lineup and linedown moves the cursor to the end
	// when it's supposed to do so
	isInsertMode := editor.Mode.Current == mode.Insert

	// check if we're at the end of a non empty line and set isAtLineEnd flag
	if !isInsertMode &&
		editor.Textarea.IsExceedingLine() &&
		editor.Textarea.LineInfo().Width > 1 {

		editor.Textarea.CursorLineVimEnd()
		editor.isAtLineEnd = true
	} else if isInsertMode {
		editor.MoveCharacterLeft()
	}

	if editor.Mode.IsAnyVisual() {
		statusMsg.Column = sbc.KeyInfo
	}

	editor.Mode.Current = mode.Normal

	if editor.CurrentBuffer == nil {
		return statusMsg
	}

	editor.saveCursorPos()

	buf := editor.CurrentBuffer
	currHash := utils.HashContent(editor.Textarea.Value())

	// only update if there's a change otherwise
	// remove the entry we added in newHistoryEntry
	if currHash != buf.hash() {
		editor.updateBufferContent(withHistory)
	}

	editor.Textarea.ResetSelection()
	editor.Textarea.SetCursorColor(mode.Normal.Colour())

	return statusMsg
}

// SendEnterNormalModeDeferredMsg returns a bubbletea command that enters
// normal mode after a 150ms delay
func (editor *Editor) SendEnterNormalModeDeferredMsg() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(150 * time.Millisecond)
		editor.EnterNormalMode(true)
		return shared.DeferredActionMsg{}
	}
}

// EnterInsertMode sets the current editor mode to insert
// and creates a new history entry
func (editor *Editor) EnterInsertMode(withHistory bool) message.StatusBarMsg {
	msg := message.StatusBarMsg{}

	if !editor.CurrentBuffer.Writeable {
		return msg
	}

	editor.Mode.Current = mode.Insert
	if withHistory {
		editor.newHistoryEntry()
	}
	editor.Textarea.SetCursorColor(mode.Insert.Colour())
	editor.Textarea.ResetSelection()

	return msg
}

// EnterReplaceMode sets the current editor mode to replace
// and creates a new history entry
func (editor *Editor) EnterReplaceMode() message.StatusBarMsg {
	msg := message.StatusBarMsg{}

	if !editor.CurrentBuffer.Writeable {
		return msg
	}

	editor.Mode.Current = mode.Replace
	editor.newHistoryEntry()
	editor.Textarea.SetCursorColor(mode.Replace.Colour())

	return msg
}

// EnterVisualMode sets the current editor mode to replace
// and creates a new history entry
func (editor *Editor) EnterVisualMode(
	selectionMode textarea.SelectionMode,
) message.StatusBarMsg {
	editor.Textarea.StartSelection(selectionMode)

	vimMode := mode.Visual
	if selectionMode == textarea.SelectVisualLine {
		vimMode = mode.VisualLine
	}

	editor.Mode.Current = vimMode
	editor.Textarea.SetCursorColor(mode.VisualBlock.Colour())
	return editor.UpdateSelectedRowsCount()
}

// newHistoryEntry creates a new history entry for the current Buffers
// saving the correct undo cursor position
func (editor *Editor) newHistoryEntry() {
	editor.CurrentBuffer.History.NewTmpEntry(editor.Textarea.CursorPos())
}

// updateHistoryEntry update the history entry saving the undo/redo
// patch, the current cursor position and the hash of the buffer content
func (editor *Editor) updateHistoryEntry() {
	buf := editor.CurrentBuffer
	editor.saveCursorPos()

	redoPatch := buf.History.MakePatch(buf.Content, editor.Textarea.Value())
	undoPatch := buf.History.MakePatch(editor.Textarea.Value(), buf.Content)

	buf.History.UpdateEntry(
		redoPatch,
		undoPatch,
		editor.Textarea.CursorPos(),
		buf.hash(),
	)
}

// checkDirty marks the current buffer as dirty if the current
// buffer is unsaved and the content differs from the saved content's file
func (editor *Editor) checkDirty() bool {
	if saved := editor.CurrentBuffer.LastSavedContentHash; saved != "" {
		isDirty := utils.HashContent(editor.Textarea.Value()) != saved
		editor.CurrentBuffer.Dirty = isDirty
		return isDirty
	}

	return false
}

// fileProgress returns the scroll progression of the file in percent
func (editor *Editor) fileProgress() int {
	pc := float32(editor.Textarea.Line()+1) / float32(editor.Textarea.LineCount())
	return int(pc * 100.0)
}

// fileProgressStr returns the scroll progression of the file as a string
func (editor *Editor) fileProgressStr() string {
	var p strings.Builder
	p.WriteString(strconv.Itoa(editor.fileProgress()))
	p.WriteByte('%')
	return p.String()
}

// cursorInfo returns the string represenation of the
// current line and cursor position
func (editor *Editor) cursorInfo() string {
	var info strings.Builder
	info.WriteString(strconv.Itoa(editor.Textarea.Line() + 1))
	info.WriteByte(',')
	info.WriteString(strconv.Itoa(editor.Textarea.LineInfo().ColumnOffset))
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

// saveCursorPos saves the cursors current column offset and row
func (editor *Editor) saveCursorPos() {
	editor.CurrentBuffer.CursorPos = editor.Textarea.CursorPos()
}

// saveCursorRow saves the cursors current row
func (editor *Editor) saveCursorRow() {
	editor.CurrentBuffer.CursorPos.Row = editor.Textarea.CursorPos().Row
	editor.CurrentBuffer.CursorPos.RowOffset = editor.Textarea.CursorPos().RowOffset
}

// saveLineLength stores the length of the current line
func (editor *Editor) saveLineLength() {
	editor.CurrentBuffer.CurrentLineLength = editor.Textarea.LineLength(-1)
}

// saveCursorCol saves the cursors current column offset
//func (e *Editor) saveCursorCol() {
//	e.CurrentBuffer.CursorPos.ColumnOffset = e.Textarea.CursorPos().ColumnOffset
//}

func (editor *Editor) UpdateStatusBarInfo() {
	editor.StatusBarMsg = editor.StatusBarInfo()
}

func (editor *Editor) StatusBarInfo() message.StatusBarMsg {
	var info strings.Builder
	info.WriteString(editor.cursorInfo())
	info.WriteRune('\t')
	info.WriteString(editor.fileProgressStr())

	return message.StatusBarMsg{
		Content: info.String(),
		Column:  sbc.Progress,
	}
}

func (editor *Editor) SetNumbers() {
	editor.Textarea.ShowLineNumbers = true
	editor.Content()
}

func (editor *Editor) SetNoNumbers() {
	editor.Textarea.ShowLineNumbers = false
	editor.Content()
}

// OpenConfig opens the config file as a buffer
func (editor *Editor) OpenConfig() message.StatusBarMsg {
	if configFile, err := app.ConfigFile(false); err == nil {
		return editor.OpenBuffer(configFile)
	}
	return message.StatusBarMsg{}
}

// OpenConfig opens the config file as a buffer
func (editor *Editor) OpenUserKeyMap() message.StatusBarMsg {
	return editor.OpenBuffer(editor.KeyInput.KeyMap.Path())
}

// MoveCharacterLeft moves the cursor one character to the left
// and checks if the cursor is either at the end or the beginning
// of the line and saves it's position
func (editor *Editor) MoveCharacterLeft() message.StatusBarMsg {
	editor.Textarea.CharacterLeft(false)
	editor.isAtLineStart = editor.Textarea.IsAtLineStart()
	editor.isAtLineEnd = false
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// MoveCharacterRight moves the cursor one character to the right
// and checks if the cursor is either at the end or the beginning
// of the line and saves its position
func (editor *Editor) MoveCharacterRight() message.StatusBarMsg {
	editor.Textarea.CharacterRight(false)
	editor.isAtLineStart = editor.Textarea.IsAtLineStart()
	editor.isAtLineEnd = false
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

func (editor *Editor) GoToChar() message.StatusBarMsg {
	debug.LogDebug("asd")
	return message.StatusBarMsg{}
}

// InsertAfter enters insert mode one character after the current cursor's
// position and saves its position
func (editor *Editor) InsertAfter() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.Textarea.CharacterRight(true)
	editor.EnterInsertMode(true)
	editor.saveCursorPos()

	return message.StatusBarMsg{}
}

// InsertLineStart moves the cursor to the beginning of the line,
// enters insert mode and saves the cursor's position
func (editor *Editor) InsertLineStart() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.Textarea.CursorInputStart()
	editor.EnterInsertMode(true)
	editor.saveCursorPos()

	return message.StatusBarMsg{}
}

// InsertLineEnd moves the cursor to the end of the line,
// enters insert mode and saves the cursor's position
func (editor *Editor) InsertLineEnd() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.Textarea.CursorEnd()
	editor.EnterInsertMode(true)
	editor.saveCursorPos()

	return message.StatusBarMsg{}
}

// InsertLine creates and empty line below the current line
// and enters insert mode.
// If above is true it inserts the line above the current line
func (editor *Editor) InsertLine(above bool) message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()

	if above {
		editor.Textarea.EmptyLineAbove()
	} else {
		editor.Textarea.EmptyLineBelow()
	}

	editor.EnterInsertMode(false)
	return message.StatusBarMsg{}
}

// LineUp moves the cursor one line up and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (editor *Editor) LineUp(multiline bool) message.StatusBarMsg {
	editor.Textarea.CursorUp()
	editor.Textarea.RepositionView()

	if !multiline {
		pos := editor.CurrentBuffer.CursorPos
		// if we have a wrapped line we skip the wrapped part of the line
		if pos.Row == editor.Textarea.CursorPos().Row &&
			editor.Textarea.Line() > 0 {
			// e.Textarea.CursorUp() doesn't work properly on some occasions
			// so I'm gonna be a little dirty
			editor.LineUp(false)
		}

		editor.Textarea.SetCursorColumn(pos.ColumnOffset)
		if editor.Textarea.IsExceedingLine() || editor.isAtLineEnd {
			editor.Textarea.CursorLineVimEnd()
		}
	}

	editor.saveCursorRow()
	editor.saveLineLength()

	if editor.Mode.IsAnyVisual() {
		return editor.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

// LineDown moves the cursor one line down and sets the column offset
// to the previous column's offset.
// If the column offset exceeds the line length, the offset is set
// to the end of the line
func (editor *Editor) LineDown(multiline bool) message.StatusBarMsg {
	editor.Textarea.CursorDown()
	editor.Textarea.RepositionView()

	if !multiline {
		pos := editor.CurrentBuffer.CursorPos

		// If we have a wrapped line we skip the wrapped part of the line
		if pos.Row == editor.Textarea.CursorPos().Row &&
			editor.Textarea.Line() < editor.Textarea.LineCount()-1 {
			// e.Textarea.CursorDown() doesn't work properly for some reason
			// so I'm gonna be a little dirty again
			editor.LineDown(false)
		}

		editor.Textarea.SetCursorColumn(pos.ColumnOffset)
		if editor.Textarea.IsExceedingLine() || editor.isAtLineEnd {
			editor.Textarea.CursorLineVimEnd()
		}
	}

	editor.saveCursorRow()
	editor.saveLineLength()

	if editor.Mode.IsAnyVisual() {
		return editor.UpdateSelectedRowsCount()
	}
	return message.StatusBarMsg{}
}

// GoToLineStart moves the cursor to the beginning of the line,
// sets isAtLineStart and saves the cursor position
func (editor *Editor) GoToLineStart() message.StatusBarMsg {
	editor.Textarea.CursorStart()
	editor.isAtLineStart = editor.Textarea.IsAtLineStart()
	editor.isAtLineEnd = editor.Textarea.IsAtLineEnd()
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToInputStart moves the cursor to the first character of the line,
// checks if the cursor is at the beginning of the line
// and saves the cursor position
func (editor *Editor) GoToInputStart() message.StatusBarMsg {
	editor.Textarea.CursorInputStart()
	editor.isAtLineStart = editor.Textarea.IsAtLineStart()
	editor.isAtLineEnd = editor.Textarea.IsAtLineEnd()
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToLineEnd moves the cursor to the end of the line, sets isAtLineEnd
// and saves the cursor position
func (editor *Editor) GoToLineEnd() message.StatusBarMsg {
	editor.Textarea.CursorLineVimEnd()
	editor.isAtLineStart = editor.Textarea.IsAtLineStart()
	editor.isAtLineEnd = editor.Textarea.IsAtLineEnd()
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// GoToTop moves the cursor to the beginning of the buffer
func (editor *Editor) GoToTop() message.StatusBarMsg {
	editor.Textarea.MoveToTop()
	editor.Textarea.RepositionView()
	editor.saveCursorPos()
	return editor.UpdateSelectedRowsCount()
}

// GoToBottom moves the cursor to the bottom of the buffer
func (editor *Editor) GoToBottom() message.StatusBarMsg {
	editor.Textarea.MoveToBottom()
	editor.Textarea.RepositionView()
	editor.saveCursorPos()
	return editor.UpdateSelectedRowsCount()
}

// WordRightStart moves the cursor to the beginning of the next word
func (editor *Editor) WordForward(end bool) message.StatusBarMsg {
	if end {
		editor.Textarea.WordRightEnd()
	} else {
		editor.Textarea.WordRight()
	}
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// WordBack moves the cursor to the beginning of the next word
func (editor *Editor) WordBack(end bool) message.StatusBarMsg {
	if end {
		editor.Textarea.WordLeft()
	} else {
		editor.Textarea.WordLeft()
	}
	editor.isAtLineEnd = false
	editor.saveCursorPos()
	return message.StatusBarMsg{}
}

// FindCharacter searches for the given character in the current line,
// If back is true if searches back otherwise forward.
// If found, it updates the cursor position
func (editor *Editor) FindCharacter(char string, back bool) message.StatusBarMsg {
	charPos := editor.Textarea.FindCharacter(char, back)

	if charPos != nil {
		editor.Textarea.SetCursorColumn(charPos.ColumnOffset)
		editor.saveCursorPos()
	}

	return message.StatusBarMsg{}
}

func (editor *Editor) DeleteBeforeCharacter(char string, back bool) message.StatusBarMsg {
	charPos := editor.Textarea.FindCharacter(char, back)

	if charPos != nil {
		editor.Textarea.SetCursorColumn(charPos.ColumnOffset)
		editor.saveCursorPos()
	}

	return message.StatusBarMsg{}
}

// DownHalfPage moves the cursor down half a page
func (editor *Editor) DownHalfPage() message.StatusBarMsg {
	editor.Textarea.DownHalfPage()
	return editor.UpdateSelectedRowsCount()
}

// UpHalfPage moves the cursor up half a page
func (editor *Editor) UpHalfPage() message.StatusBarMsg {
	editor.Textarea.UpHalfPage()
	return editor.UpdateSelectedRowsCount()
}

// SelectWord selects the  word the cursor is currently on.
// If outer is true it includes the whitespace after.
// Only effective if we're in visual mode
func (editor *Editor) SelectWord(outer bool) message.StatusBarMsg {
	if outer {
		editor.Textarea.SelectOuterWord()
	} else {
		editor.Textarea.SelectInnerWord()
	}
	return editor.UpdateSelectedRowsCount()
}

// DeleteLine deletes the current line and copies its content
// to the clipboard
func (editor *Editor) DeleteLine() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()

	editor.saveLineLength()
	editor.YankLine()
	editor.Textarea.DeleteLine()
	editor.updateBufferContent(true)
	editor.EnterNormalMode(true)

	return editor.ResetSelectedRowsCount()
}

// DeleteWord deletes the word the cursor is on.
// If outer is true it includes the trailing space.
// If enterInsertMode is true, we're going straight into inser mode.
func (editor *Editor) DeleteWord(outer bool, enterInsertMode bool) message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()

	if outer {
		editor.Textarea.DeleteOuterWord()
	} else {
		editor.Textarea.DeleteInnerWord()
	}

	editor.updateBufferContent(true)

	if enterInsertMode {
		editor.EnterInsertMode(false)
	}

	return editor.ResetSelectedRowsCount()
}

// DeleteAfterCursor deletes all characters after the cursor
func (editor *Editor) DeleteAfterCursor(overshoot bool) message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()
	editor.Textarea.DeleteAfterCursor(overshoot)
	editor.updateBufferContent(true)
	return editor.ResetSelectedRowsCount()
}

// DeleteNLines deletes n lines
func (editor *Editor) DeleteNLines(lines int, up bool) message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()
	editor.Textarea.DeleteLines(lines, up)
	editor.updateBufferContent(true)
	editor.Textarea.RepositionView()
	return editor.ResetSelectedRowsCount()
}

// DeleteWordRight deletes the rest of word after the cursor
func (editor *Editor) DeleteWordRight() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()
	editor.Textarea.DeleteWordRight()
	editor.updateHistoryEntry()
	return message.StatusBarMsg{}
}

// MergeLineBelow merges the current line with the line below
func (editor *Editor) MergeLineBelow() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	editor.newHistoryEntry()
	editor.Textarea.VimMergeLineBelow(editor.CurrentBuffer.CursorPos.Row)
	editor.updateBufferContent(true)
	return message.StatusBarMsg{}
}

// DeleteRune the rune that the cursor is currently on.
// If buffer is in visual mode it takes the selection into account
// If keepMode is true this method doesn't enter normal mode
func (editor *Editor) DeleteRune(
	keepMode bool,
	withHistory bool,
	noYank bool,
) message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	if withHistory {
		editor.newHistoryEntry()
	}

	c := editor.CurrentBuffer.CursorPos
	char := ""

	cursorPos := editor.Textarea.CursorPos()
	minRange, maxRange := editor.Textarea.Selection.Range(cursorPos)

	if minRange.Row > -1 {
		char = editor.Textarea.SelectionStr()
		if editor.Textarea.Selection.Mode == textarea.SelectVisualLine {
			editor.Textarea.DeleteSelectedLines()
		} else {
			editor.Textarea.DeleteRunesInRange(minRange, maxRange)
		}
	} else {
		char = editor.Textarea.DeleteRune(c.Row, c.ColumnOffset)
	}

	if !noYank {
		editor.Yank(char)
	}

	if !keepMode {
		editor.EnterNormalMode(withHistory)
	}
	editor.Textarea.RepositionView()
	return editor.ResetSelectedRowsCount()
}

// ResetSelectedRowsCount resets the selected rows count in the status bar
func (editor *Editor) ResetSelectedRowsCount() message.StatusBarMsg {
	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.KeyInfo,
	}
}

// UpdateSelectedRowsCount updates the selected rows count in the status bar
func (editor *Editor) UpdateSelectedRowsCount() message.StatusBarMsg {
	if editor.Mode.IsAnyVisual() {
		return message.StatusBarMsg{
			Content: strconv.Itoa(editor.SelectedRowsCount()),
			Column:  sbc.KeyInfo,
		}
	}
	return message.StatusBarMsg{}
}

// SelectedRowsCount returns the number of selected rows
func (editor *Editor) SelectedRowsCount() int {
	startRow := editor.Textarea.Selection.StartRow
	cursorRow := editor.Textarea.CursorPos().Row
	minRow := min(startRow, cursorRow)
	maxRow := max(startRow, cursorRow)

	return (maxRow - minRow) + 1
}

// Undo sets the buffer content to the previous history entry
func (editor *Editor) Undo() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	if val, cursorPos := editor.CurrentBuffer.undo(); val != "" {
		curBuf := editor.CurrentBuffer

		// dirty check
		curBuf.Dirty = val != curBuf.LastSavedContentHash
		editor.Textarea.SetValue(val)

		editor.Textarea.MoveCursor(
			cursorPos.Row,
			cursorPos.RowOffset,
			cursorPos.ColumnOffset,
		)

		editor.Textarea.RepositionView()
		editor.CurrentBuffer.Content = editor.Textarea.Value()
		editor.isAtLineEnd = editor.Textarea.IsAtLineEnd()
		editor.isAtLineStart = editor.Textarea.IsAtLineStart()
		editor.saveCursorPos()
	}

	return message.StatusBarMsg{}
}

// Redo sets the buffer content to the next history entry
func (editor *Editor) Redo() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	if val, cursorPos := editor.CurrentBuffer.redo(); val != "" {
		// dirty check
		editor.CurrentBuffer.Dirty = val != editor.CurrentBuffer.LastSavedContentHash
		editor.Textarea.SetValue(val)

		editor.Textarea.MoveCursor(
			cursorPos.Row,
			cursorPos.RowOffset,
			cursorPos.ColumnOffset,
		)

		editor.Textarea.RepositionView()
		editor.CurrentBuffer.Content = editor.Textarea.Value()
	}

	return message.StatusBarMsg{}
}

// Yank copies the given string to the clipboard
func (editor *Editor) Yank(str string) message.StatusBarMsg {
	if err := clipboard.Write(str); err != nil {
		debug.LogDebug(err)
	}
	return message.StatusBarMsg{}
}

// YankSelection copies the current selection to the clipboard.
// If keepCursorPos is true the cursor position remains the same
// otherwise the cursor is moved to the beginning of the selection
func (editor *Editor) YankSelection(keepCursor bool) message.StatusBarMsg {
	sel := editor.Textarea.SelectionStr()
	editor.Yank(sel)

	var cursorDeferredCmd tea.Cmd

	buf := editor.CurrentBuffer
	startRow := editor.Textarea.Selection.StartRow
	startCol := editor.Textarea.Selection.StartCol

	if editor.Mode.Current == mode.VisualLine {
		startCol = 0
		// append a new line as a hacky indicator that we are supposed
		// to paste visual line selections on a new line
		sel += "\n"
	}

	cursor := textarea.CursorPos{
		Row:          startRow,
		RowOffset:    buf.CursorPos.RowOffset,
		ColumnOffset: startCol,
	}

	if keepCursor {
		cursor.ColumnOffset = editor.Textarea.Selection.StartCol
	}

	// move the cursor to the beginning of the selection after
	// a short delay to briefly show the selection
	cursorDeferredCmd = func() tea.Msg {
		time.Sleep(150 * time.Millisecond)
		editor.Textarea.MoveCursor(cursor.Row, cursor.RowOffset, cursor.ColumnOffset)
		editor.isAtLineEnd = editor.Textarea.IsAtLineEnd()

		return shared.DeferredActionMsg{}
	}

	return message.StatusBarMsg{
		Cmd: tea.Batch(
			cursorDeferredCmd,
			editor.SendEnterNormalModeDeferredMsg(),
		),
	}
}

func (editor *Editor) YankAfterCursor() message.StatusBarMsg {
	editor.saveCursorPos()
	editor.Textarea.StartSelection(textarea.SelectVisual)
	editor.GoToLineEnd()

	return editor.YankSelection(true)
}

// YankLine copies the current line to the clipboard
func (editor *Editor) YankLine() message.StatusBarMsg {
	editor.saveCursorPos()
	editor.EnterVisualMode(textarea.SelectVisualLine)
	return editor.YankSelection(true)
}

// YankWord copies the current word to the clipboard.
// If outer is set to true it copies the space after the word.
func (editor *Editor) YankWord(outer bool) message.StatusBarMsg {
	editor.EnterVisualMode(textarea.SelectVisual)

	if outer {
		editor.Textarea.SelectOuterWord()
	} else {
		editor.Textarea.SelectInnerWord()
	}

	return editor.YankSelection(false)
}

// Paste pastes the clipboard content.
// If the selection exceeds the length of the current line
// it attempts to paste the clipboard content on a newline below
// the current line
func (editor *Editor) Paste() message.StatusBarMsg {
	if !editor.CurrentBuffer.Writeable {
		return message.StatusBarMsg{}
	}

	cnt, err := clipboard.Read()

	if err != nil {
		debug.LogDebug(err)
	}

	if len(cnt) > 0 {
		// save the curren cursor position to adjust the correct position
		// after the clipboard content is pasted
		var (
			cursorPos = editor.CurrentBuffer.CursorPos
			col       = cursorPos.ColumnOffset
			row       = cursorPos.Row
			rowOffset = cursorPos.RowOffset
		)

		editor.newHistoryEntry()

		r := []rune(cnt)
		pasteOnNewLine := r[len(r)-1] == '\n'

		if pasteOnNewLine {
			editor.Textarea.EmptyLineBelow()

			// strip the last new line since we've already inserted
			// an empty line so we don't need it
			// otherwise it would produce an additional empty line
			cnt = strings.TrimRight(cnt, "\n")

			// set the cursor position at the beginning of the next row
			// which is the newly pasted content
			col = 0
			row++
		} else {
			cnt = strings.TrimSpace(cnt)
			// if the clipboard content is not a full line set the
			// add the length of the selection to the current column offset
			// to set the cursor to the end of the selection
			col += len(cnt)
			editor.Textarea.CharacterRight(false)
		}

		// insert clipboard content
		editor.Textarea.InsertString(cnt)
		editor.Textarea.MoveCursor(row, rowOffset, col)
		editor.Textarea.RepositionView()

		editor.updateBufferContent(true)
	}
	return message.StatusBarMsg{}
}

// ChangeCaseOfSelection changes the case of the selected text to
// either lower- or uppercase depending on `toUpper`
func (editor *Editor) ChangeCaseOfSelection(toUpper bool) message.StatusBarMsg {
	editor.newHistoryEntry()

	cursorPos := editor.Textarea.CursorPos()
	selection := editor.Textarea.SelectionStr()
	start, end := editor.Textarea.Selection.Range(cursorPos)

	// If we're in visual line mode set the start column to the first
	// of the first line and the end column to the last column of the
	// last selected line
	if editor.Mode.Current == mode.VisualLine {
		start.ColumnOffset = 0
		end.ColumnOffset = editor.Textarea.LineLength(end.Row) - 1
	}
	editor.Textarea.DeleteRunesInRange(start, end)

	if toUpper {
		editor.Textarea.InsertString(strings.ToUpper(selection))
	} else {
		editor.Textarea.InsertString(strings.ToLower(selection))
	}

	editor.Textarea.MoveCursor(start.Row, start.RowOffset, start.ColumnOffset)
	editor.EnterNormalMode(true)

	return message.StatusBarMsg{}
}

// cursorPosFromConf retrieves the cursor position of the given note
// from the meta config file.
// If the meta config value is invalid it returns the empty CursorPos which
// equals the beginning of the file
func (editor *Editor) cursorPosFromConf(filepath string) textarea.CursorPos {
	cursorPos := textarea.CursorPos{}
	pos, err := editor.conf.MetaValue(filepath, config.CursorPosition)

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
func (editor *Editor) saveCursorPosToConf() {
	pos := editor.Textarea.CursorPos()

	editor.conf.SetMetaValue(
		editor.CurrentBuffer.path,
		config.CursorPosition,
		pos.String(),
	)
}

// updateBufferContent replaces the content of the current buffer with the
// current textarea value
func (editor *Editor) updateBufferContent(withHistory bool) {
	if withHistory {
		editor.updateHistoryEntry()
	}

	// set the content after we updated the buffer history
	// otherwise the undo/redo-patches won't be correct
	editor.CurrentBuffer.Content = editor.Textarea.Value()
}

// UpdateMetaInfo records the current state of the editor by updating
// metadata values for recently opened notes and the currently opened note.
func (editor *Editor) UpdateMetaInfo() {
	notePaths := make([]string, 0, len(*editor.Buffers))

	for _, buf := range *editor.Buffers {
		notePaths = append(notePaths, buf.Path(true))
	}

	noteStr := strings.Join(notePaths[:], ",")

	editor.conf.SetMetaValue("", config.LastNotes, noteStr)
	editor.conf.SetMetaValue("", config.LastOpenNote, editor.CurrentBuffer.Path(true))
}

// LineNumbers returns whether line numbers are enabled in the config file
func (editor *Editor) LineNumbers() bool {
	numbers, err := editor.conf.Value(config.Editor, config.LineNumbers)

	if err != nil {
		debug.LogErr(err)
		return false
	}

	return numbers.GetBool()
}

// SearchIgnoreCase returns true if the editor config enables
// case-insensitive search.
func (editor *Editor) SearchIgnoreCase() bool {
	ignoreCase, err := editor.conf.Value(config.Editor, config.SearchIgnoreCase)

	if err != nil {
		return false
	}

	return ignoreCase.GetBool()
}

func (editor *Editor) RefreshTextAreaStyles() {
	s := defaultStyles(editor.Theme())
	editor.Textarea.Styles.Blurred.Base = s.blurred
	editor.Textarea.Styles.Focused.Base = s.focused
	editor.Textarea.ShowLineNumbers = editor.LineNumbers()
	editor.BuildHeader(editor.Size.Width, true)
	editor.Content()
}
