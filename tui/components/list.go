package components

import (
	"bellbird-notes/tui/messages"

	"github.com/charmbracelet/bubbles/textinput"
)

// EditState is the state in which the DirectoryTree.editor is in
// when the Insert mode is active
type EditState int

const (
	EditNone EditState = iota
	EditCreate
	EditRename
)

type ListActions interface {
	Refresh()
}

type ListItem interface {
	GetIndex() int
	GetPath() string
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
	editState EditState

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
}

type Item struct {
	// The row's index is primarily used to determine the indentation
	// of a directory.
	index int

	Name     string
	Path     string
	selected bool

	styles styles
}

type statusMsg string

const reservedLines = 3

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

	if l.selectedIndex >= 0 && l.selectedIndex <= l.length {
		return &items[l.selectedIndex]
	}

	return nil
}

// indexByPath returns the interal list index by the given path
func (l List[T]) indexByPath(path string, items *[]T) int {
	if items == nil {
		items = &l.items
	}
	for _, item := range *items {
		if item.GetPath() == path {
			return item.GetIndex()
		}
	}
	return 0
}

///
/// keyboard shortcut commands
///

// Decrements `m.selectedIndex`
func (l *List[T]) LineUp() messages.StatusBarMsg {
	if l.selectedIndex > 0 {
		l.selectedIndex--
	}

	// scroll up
	if l.selectedIndex < l.firstVisibleLine {
		l.firstVisibleLine = l.selectedIndex
		l.lastVisibleLine = l.visibleLines + l.firstVisibleLine
		l.viewport.LineUp(1)
	}

	return messages.StatusBarMsg{Column: 2}
}

// Increments `m.selectedIndex`
func (l *List[T]) LineDown() messages.StatusBarMsg {
	if l.selectedIndex < l.length-1 {
		l.selectedIndex++
	}

	// scroll down
	if l.selectedIndex > l.visibleLines {
		l.firstVisibleLine = l.selectedIndex - l.visibleLines
		l.lastVisibleLine = l.selectedIndex
		l.viewport.LineDown(1)
	}

	return messages.StatusBarMsg{Column: 2}
}

// GoToTop moves the selection and viewport to the top of the tree
func (l *List[T]) GoToTop() messages.StatusBarMsg {
	l.selectedIndex = 0
	l.viewport.GotoTop()
	return messages.StatusBarMsg{}
}

// GoToBottom moves the selection and viewport to the bottom of the tree
func (l *List[T]) GoToBottom() messages.StatusBarMsg {
	l.selectedIndex = l.lastIndex
	l.viewport.GotoBottom()
	return messages.StatusBarMsg{}
}

func RefreshList[T interface{ Refresh() }](a T) {
	a.Refresh()
}

// Rename renames the currently selected directory and
// returns a message that is displayed in the status bar
func (l *List[T]) Rename(origName string) messages.StatusBarMsg {
	if l.editIndex == nil {
		l.editState = EditRename
		l.editIndex = &l.selectedIndex
		l.editor.SetValue(origName)
		// set cursor to last position
		l.editor.CursorEnd()
	}
	return messages.StatusBarMsg{}
}

// Cancel the current action and blurs the editor
func (l *List[T]) CancelAction(cb func()) messages.StatusBarMsg {
	if l.editState != EditNone {
		l.editIndex = nil
		l.editState = EditNone
		l.editor.Blur()
	}
	cb()
	return messages.StatusBarMsg{}
}
