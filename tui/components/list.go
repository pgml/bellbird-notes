package components

import (
	"bellbird-notes/app/config"
	"bellbird-notes/tui/message"

	"github.com/charmbracelet/bubbles/v2/textinput"
)

// EditState is the state in which the DirectoryTree.editor is in
// when the Insert mode is active
type EditState int

const (
	EditStateNone EditState = iota
	EditStateCreate
	EditStateRename
	EditStateDelete
)

var EditStates = struct {
	None, Create, Rename, Delete EditState
}{
	None:   EditStateNone,
	Create: EditStateCreate,
	Rename: EditStateRename,
	Delete: EditStateDelete,
}

type ListActions interface {
	Refresh()
}

type ListItem interface {
	Index() int
	Path() string
}

// List represents a bubbletea model
type List[T ListItem] struct {
	Component

	// The currently selector directory row
	selectedIndex int
	// The text input that is used for renaming or creating directories
	editor textinput.Model
	// The index of the currently edited directory row
	editIndex *int
	// States if directory is being created or renamed
	EditState EditState

	// Stores the list items
	items []T

	// We set the length manually because len(items) won't be possible
	// for the directories since the multidimensial []items
	// doesn't reflect the actual displayed list items when T is Dir
	// In this case the real length would come from `
	// DirectoryTree.dirsListFlat`
	length    int
	lastIndex int

	firstVisibleLine int
	lastVisibleLine  int
	visibleLines     int

	conf *config.Config
}

type Item struct {
	// The row's index is primarily used to determine the indentation
	// of a directory.
	index int

	name     string
	path     string
	selected bool
	input    *textinput.Model

	styles styles
	width  int

	nerdFonts bool
}

type statusMsg string

const reservedLines = 1

// UpdateViewportInfo synchronises the list's internal visible line count
// with the actual height of the viewport, subtracting `reservedLines`
// because I couldn't figure out why `VisibleLineCount()` seem to be more
// that it should be
//
// This ensures scrolling and item selection logic remain accurate after
// layout changes or terminal resizes.
func (l *List[T]) UpdateViewportInfo() {
	if l.visibleLines != l.viewport.VisibleLineCount() {
		l.visibleLines = l.viewport.VisibleLineCount() - reservedLines
		l.lastVisibleLine = l.visibleLines
	}
}

// SelectedItem returns the currently selected item of the list
func (l *List[T]) SelectedItem(items []T) *T {
	if l.length == 0 {
		return nil
	}

	if items == nil {
		items = l.items
	}

	if l.selectedIndex >= 0 && l.selectedIndex < l.length {
		return &items[l.selectedIndex]
	}

	return nil
}

// indexByPath returns the interal list index by the given path
// if no items are provided it takes the cached items
func (l List[T]) indexByPath(path string, items *[]T) int {
	if items == nil {
		items = &l.items
	}
	for _, item := range *items {
		if item.Path() == path {
			return item.Index()
		}
	}
	return 0
}

///
/// keyboard shortcut commands
///

// LineUp decrements `m.selectedIndex`
func (l *List[T]) LineUp() message.StatusBarMsg {
	if l.selectedIndex > 0 {
		l.selectedIndex--
	}

	// scroll up
	if l.selectedIndex < l.firstVisibleLine {
		l.firstVisibleLine = l.selectedIndex
		l.viewport.LineUp(1)
	}

	return message.StatusBarMsg{}
}

// LineDown increments `m.selectedIndex`
func (l *List[T]) LineDown() message.StatusBarMsg {
	if l.selectedIndex < l.length-1 {
		l.selectedIndex++
	}

	l.lastVisibleLine = l.visibleLines + l.firstVisibleLine

	// scroll down
	if l.selectedIndex > l.lastVisibleLine {
		l.firstVisibleLine = l.selectedIndex - l.visibleLines
		l.viewport.LineDown(1)
	}

	return message.StatusBarMsg{}
}

// GoToTop moves the selection and viewport to the top of the tree
func (l *List[T]) GoToTop() message.StatusBarMsg {
	l.selectedIndex = 0
	l.firstVisibleLine = 0
	l.viewport.GotoTop()
	return message.StatusBarMsg{}
}

// GoToBottom moves the selection and viewport to the bottom of the tree
func (l *List[T]) GoToBottom() message.StatusBarMsg {
	l.selectedIndex = l.lastIndex
	l.firstVisibleLine = l.length - l.visibleLines
	l.viewport.GotoBottom()
	return message.StatusBarMsg{}
}

func RefreshList[T interface{ Refresh() }](a T) {
	a.Refresh()
}

// Rename renames the currently selected directory and
// returns a message that is displayed in the status bar
func (l *List[T]) Rename(origName string) message.StatusBarMsg {
	if l.editIndex == nil {
		l.EditState = EditStates.Rename
		l.editIndex = &l.selectedIndex
		l.editor.SetValue(origName)
		// set cursor to last position
		l.editor.CursorEnd()
	}
	return message.StatusBarMsg{}
}

// CancelAction cancels the current action and blurs the editor
func (l *List[T]) CancelAction(cb func()) message.StatusBarMsg {
	l.resetEditor()
	cb()
	return message.StatusBarMsg{}
}

func (l *List[T]) resetEditor() {
	if l.EditState != EditStates.None {
		l.editIndex = nil
		l.EditState = EditStates.None
		l.editor.Blur()
	}
}

func (l *List[T]) SelectedIndex() int {
	return l.selectedIndex
}
