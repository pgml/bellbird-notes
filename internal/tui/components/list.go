package components

import (
	"bellbird-notes/internal/tui/messages"

	"github.com/charmbracelet/bubbles/textinput"
)

// EditState is the state in which the DirectoryTree.editor is
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

	selectedIndex int // The currently selector directory row

	editor       textinput.Model // The text input that is used for renaming or creating directories
	editingIndex *int            // The index of the currently edited directory row
	editingState EditState       // States if directory is being created or renamed

	items []T // Stores the list items

	// We set the length manually because len(items) won't be possible
	// for the directories since the multidimensial []items doesn't reflect
	// the actual displayed list items when T is Dir
	// In this case the real length would come from `DirectoryTree.dirsListFlat`
	length    int
	lastIndex int

	firstVisibleLine int
	lastVisibleLine  int
	visibleLineCount int
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

func (l *List[T]) UpdateViewportInfo() {
	if l.visibleLineCount != l.viewport.VisibleLineCount() {
		l.visibleLineCount = l.viewport.VisibleLineCount() - 3
		l.lastVisibleLine = l.visibleLineCount
	}
}

// SelectedDir returns the currently selected directory in the directory tree
func (l List[T]) SelectedItem(items []T) T {
	var empty T

	if l.length == 0 {
		return empty
	}

	if items == nil {
		items = l.items
	}

	if l.selectedIndex >= 0 && l.selectedIndex < l.length {
		return items[l.selectedIndex]
	}
	return empty
}

func (l List[T]) indexByPath(path string, items []T) int {
	if items == nil {
		items = l.items
	}
	for _, item := range items {
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
		l.lastVisibleLine = l.visibleLineCount + l.firstVisibleLine
		l.viewport.LineUp(1)
	}

	return messages.StatusBarMsg{}
}

// Increments `m.selectedIndex`
func (l *List[T]) LineDown() messages.StatusBarMsg {
	if l.selectedIndex < l.length-1 {
		l.selectedIndex++
	}

	// scroll down
	if l.selectedIndex > l.visibleLineCount {
		l.firstVisibleLine = l.selectedIndex - l.visibleLineCount
		l.lastVisibleLine = l.selectedIndex
		l.viewport.LineDown(1)
	}

	return messages.StatusBarMsg{}
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
	if l.editingIndex == nil {
		l.editingState = EditRename
		l.editingIndex = &l.selectedIndex
		l.editor.SetValue(origName)
		// set cursor to last position
		l.editor.CursorEnd()
	}
	return messages.StatusBarMsg{}
}

// Cancel the current action and blurs the editor
func (l *List[T]) CancelAction(cb func()) messages.StatusBarMsg {
	if l.editingState != EditNone {
		l.editingIndex = nil
		l.editingState = EditNone
		l.editor.Blur()
	}
	cb()
	return messages.StatusBarMsg{}
}
