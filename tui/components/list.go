package components

import (
	"slices"
	"strings"

	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/directories"
	"bellbird-notes/app/notes"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/theme"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/lipgloss/v2"
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
	Name() string
	Path() string
	//Title() string
	IsCut() bool
	SetIsCut(isCut bool)
	//RefreshStyles()
	//BuildHeader(width int, rebuild bool) string
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

	isPinned bool
	isCut    bool
}

// Index returns the index of a Note-Item
func (i Item) Index() int { return i.index }

// Path returns the index of a Note-Item
func (i Item) Path() string { return i.path }

// Name returns the name of a Note-Item
func (i Item) Name() string { return i.name }

// IsCut returns whether the note item is cut
func (i Item) IsCut() bool { return i.isCut }

// SetIsCut returns whether the note item is cut
func (i *Item) SetIsCut(isCut bool) { i.isCut = isCut }

type PinnedItem interface {
	Path() string
}

type PinnedItems[T PinnedItem] struct {
	items []T

	// indicates whether notes has been fully populated with the pinned notes
	// of the current directory.
	// This should only be true after the directory is loaded
	loaded bool
}

func (p *PinnedItems[T]) add(item T) {
	p.items = append(p.items, item)
}

// contains returns whether a NoteItem is in `notes`
func (p PinnedItems[T]) contains(item T) bool {
	for _, n := range p.items {
		if n.Path() == item.Path() {
			return true
		}
	}
	return false
}

func (p *PinnedItems[T]) remove(item T) {
	for i, n := range p.items {
		if n.Path() == item.Path() {
			p.items = slices.Delete(p.items, i, i+1)
			return
		}
	}
}

// toggle adds or removes the given note to the pinned notes
// depending on whether it's already in the slice
func (p *PinnedItems[T]) toggle(item T) {
	if !p.contains(item) {
		p.add(item)
	} else {
		p.remove(item)
	}
}

type statusMsg string

const reservedLines = 0

// List represents a bubbletea model and holds items that implement ListItem.
// Important: T must be a pointer type, otherwise methods like SetIsCut() won't work.
// This allows us to mutate fields inside the list items when needed.
type List[T ListItem] struct {
	Component

	title string

	// The currently selector directory row
	selectedIndex int

	// The text input that is used for renaming or creating directories
	input textinput.Model

	// The index of the currently edited directory row
	editIndex *int

	// States if directory is being created or renamed
	EditState EditState

	// Stores the list items
	items []T

	yankedItems []T

	// Contains all pinned notes of the current directory
	PinnedItems PinnedItems[T]

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

//func (l List[T]) Title() string { return "" }

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
func (l *List[T]) SelectedItem(items []T) T {
	var none T
	if l.length == 0 {
		return none
	}

	if items == nil {
		items = l.items
	}

	if l.selectedIndex >= 0 && l.selectedIndex <= l.length {
		return items[l.selectedIndex]
	}

	return none
}

// ItemsContain returns the ListItem with the given path from the List.
// If no such item exists, it returns a nil and an error.
func (l List[T]) ItemsContain(path string) (T, bool) {
	for i := range l.items {
		if l.items[i].Path() == path {
			return l.items[i], true
		}
	}

	var none T
	return none, false
}

func (l List[T]) YankedItemsContain(path string) (T, bool) {
	for i := range l.yankedItems {
		item := l.yankedItems[i]
		if item.Path() == path {
			return item, true
		}
	}

	var none T
	return none, false
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
		l.input.SetValue(origName)
		// set cursor to last position
		l.input.CursorEnd()
	}
	return message.StatusBarMsg{}
}

// TogglePinned pins or unpins the current selection
func (l *List[T]) togglePinned(item T) {
	path := item.Path()

	// check if the selection already has a state
	p, err := l.conf.MetaValue(path, config.Pinned)

	// set default state if not
	if err != nil {
		l.conf.SetMetaValue(path, config.Pinned, "false")
		debug.LogErr(err)
	}

	// write to metadata file
	if p == "true" {
		l.conf.SetMetaValue(path, config.Pinned, "false")
	} else {
		l.conf.SetMetaValue(path, config.Pinned, "true")
	}

	l.PinnedItems.toggle(item)
}

func (l *BufferList) ConfirmAction() message.StatusBarMsg {
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
		l.input.Blur()
	}
}

func (l *List[T]) YankSelection(markCut bool) {}

func (l *List[T]) pasteSelection(item T, dirPath string, cb func(string)) {
	name := item.Name()

	if item.IsCut() {
		if item, ok := l.ItemsContain(item.Path()); ok {
			l.selectedIndex = item.Index()
			item.SetIsCut(false)
			return
		}
	}

	newPath := dirPath + "/" + name

	// Ensure we always have a valid path
	if l.isNote(item.Path()) {
		newPath = notes.GetValidPath(newPath, true)
	} else {
		newPath = directories.GetValidPath(newPath)
	}

	cb(newPath)
}

func (l *List[T]) PasteSelection() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

// checkName ensures that the note name does not conflict with existing notes
// in the specified directory. If a conflict exists, it appends " Copy" to the name.
func (l *List[T]) CheckName(dirPath string, name string) string {
	if l.isNote(name) {
		// In case this is a note we check for the extension.
		// If it's not a name it should return
		newPath := notes.CheckPath(dirPath + "/" + name)

		if _, err := notes.Exists(newPath); err == nil {
			name += " Copy"
		}
	} else {
		if _, err := directories.Exists(dirPath + "/" + name); err == nil {
			name += " Copy"
		}
	}

	return name
}

func (l *List[T]) isNote(name string) bool {
	return strings.HasSuffix(name, notes.Ext) ||
		strings.HasSuffix(name, notes.LegacyExt) ||
		strings.HasSuffix(name, notes.ConfExt)
}

func (l *List[T]) SelectedIndex() int {
	return l.selectedIndex
}

func (l *List[T]) SetSelectedIndex(index int) {
	l.selectedIndex = index
}

func (l *List[T]) TogglePinned() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (l *List[T]) ConfirmRemove() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (l *List[T]) Remove() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (l *List[T]) Refresh(
	resetSelectedIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	return message.StatusBarMsg{}
}

// BuildHeader builds title of the directory tree column
func (l *List[T]) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if l.header != nil && !rebuild {
		if width == lipgloss.Width(*l.header) {
			return *l.header
		}
	}

	header := theme.Header(l.title, width, l.Focused()) + "\n"
	l.header = &header
	return header
}

func (l *List[T]) RefreshStyles() {
	l.viewport.Style = theme.BaseColumnLayout(
		l.Size,
		l.Focused(),
	)
	l.Remove()
	l.BuildHeader(l.Size.Width, true)
}
