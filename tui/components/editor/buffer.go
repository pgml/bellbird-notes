package editor

import (
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components/textarea"
	"net/url"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
)

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
