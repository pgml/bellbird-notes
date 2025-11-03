package shared

import (
	"slices"
	"strings"

	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/directories"
	"bellbird-notes/app/notes"
	"bellbird-notes/tui/message"

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
	//SetName()
	Path() string
	//SetPath()
	IsCut() bool
	SetIsCut(isCut bool)
	//Title() string
	//RefreshStyles()
	//BuildHeader(width int, rebuild bool) string
}

type Item struct {
	// The row's index is primarily used to determine the indentation
	// of a directory.
	index int

	name       string
	path       string
	inputModel *textinput.Model

	Styles Styles
	width  int

	NerdFonts bool

	IsPinned   bool
	isCut      bool
	IsSelected bool
}

// Index returns the index of a Note-Item
func (i Item) Index() int { return i.index }

// SetIndex returns the index of a Note-Item
func (i *Item) SetIndex(index int) { i.index = index }

// Path returns the index of a Note-Item
func (i Item) Path() string { return i.path }

// Path returns the index of a Note-Item
func (i *Item) SetPath(path string) { i.path = path }

// IsCut returns the index of a Note-Item
func (i Item) IsCut() bool { return i.isCut }

// SetIsCut returns the index of a Note-Item
func (i *Item) SetIsCut(isCut bool) { i.isCut = isCut }

// Name returns the name of a Note-Item
func (i Item) Name() string { return i.name }

// Name returns the name of a Note-Item
func (i *Item) SetName(name string) { i.name = name }

// InputModel returns the name of a Note-Item
func (i Item) InputModel() *textinput.Model { return i.inputModel }

// SetInputModel returns the name of a Note-Item
func (i *Item) SetInputModel(model textinput.Model) { i.inputModel = &model }

// Width returns the index of a Note-Item
func (i Item) Width() int { return i.width }

// SetWidth returns the index of a Note-Item
func (i *Item) SetWidth(width int) { i.width = width }

type PinnedItem interface {
	Path() string
}

type PinnedItems[T PinnedItem] struct {
	Items []T

	// indicates whether notes has been fully populated with the pinned notes
	// of the current directory.
	// This should only be true after the directory is IsLoaded
	IsLoaded bool
}

func (p *PinnedItems[T]) Add(item T) {
	p.Items = append(p.Items, item)
}

// Contains returns whether a NoteItem is in `notes`
func (p PinnedItems[T]) Contains(item T) bool {
	for _, n := range p.Items {
		if n.Path() == item.Path() {
			return true
		}
	}
	return false
}

func (p *PinnedItems[T]) Remove(item T) {
	for i, n := range p.Items {
		if n.Path() == item.Path() {
			p.Items = slices.Delete(p.Items, i, i+1)
			return
		}
	}
}

// Toggle adds or removes the given note to the pinned notes
// depending on whether it's already in the slice
func (p *PinnedItems[T]) Toggle(item T) {
	if !p.Contains(item) {
		p.Add(item)
	} else {
		p.Remove(item)
	}
}

const ReservedLines = 1

// List represents a bubbletea model and holds items that implement ListItem.
// Important: T must be a pointer type, otherwise methods like SetIsCut() won't work.
// This allows us to mutate fields inside the list items when needed.
type List[T ListItem] struct {
	Component

	Title string

	// The currently selector directory row
	SelectedIndex int

	// The text input that is used for renaming or creating directories
	InputModel textinput.Model

	// The index of the currently edited directory row
	EditIndex *int

	// States if directory is being created or renamed
	EditState EditState

	// Stores the list items
	Items []T

	YankedItems []T

	// Contains all pinned notes of the current directory
	PinnedItems PinnedItems[T]

	// We set the Length manually because len(items) won't be possible
	// for the directories since the multidimensial []items
	// doesn't reflect the actual displayed list items when T is Dir
	// In this case the real Length would come from `
	// DirectoryTree.dirsListFlat`
	Length    int
	LastIndex int

	FirstVisibleLine int
	LastVisibleLine  int
	VisibleLines     int

	Conf *config.Config
}

func (l List[T]) MakeEmpty() List[T] {
	list := List[T]{
		SelectedIndex:    0,
		EditIndex:        nil,
		EditState:        EditStates.None,
		Items:            make([]T, 0),
		PinnedItems:      PinnedItems[T]{},
		FirstVisibleLine: 0,
		LastVisibleLine:  0,
	}

	return list
}

// UpdateViewportInfo synchronises the list's internal visible line count
// with the actual height of the viewport, subtracting `reservedLines`
// because I couldn't figure out why `VisibleLineCount()` seem to be more
// that it should be
//
// This ensures scrolling and item selection logic remain accurate after
// layout changes or terminal resizes.
func (l *List[T]) UpdateViewportInfo() {
	if l.VisibleLines != l.Viewport.VisibleLineCount() {
		l.VisibleLines = l.Viewport.VisibleLineCount() - ReservedLines
		l.LastVisibleLine = l.VisibleLines
	}
}

// SelectedItem returns the currently selected item of the list
func (l *List[T]) SelectedItem(items []T) T {
	var none T
	if l.Length == 0 {
		return none
	}

	if items == nil {
		items = l.Items
	}

	if l.SelectedIndex >= 0 && l.SelectedIndex <= l.Length {
		return items[l.SelectedIndex]
	}

	return none
}

// ItemsContain returns the ListItem with the given path from the List.
// If no such item exists, it returns a nil and an error.
func (l List[T]) ItemsContain(path string) (T, bool) {
	for i := range l.Items {
		if l.Items[i].Path() == path {
			return l.Items[i], true
		}
	}

	var none T
	return none, false
}

func (l List[T]) YankedItemsContain(path string) (T, bool) {
	for i := range l.YankedItems {
		item := l.YankedItems[i]
		if item.Path() == path {
			return item, true
		}
	}

	var none T
	return none, false
}

// indexByPath returns the interal list index by the given path
// if no items are provided it takes the cached items
func (l List[T]) IndexByPath(path string, items *[]T) int {
	if items == nil {
		items = &l.Items
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
	if l.SelectedIndex > 0 {
		l.SelectedIndex--
	}

	// scroll up
	if l.SelectedIndex < l.FirstVisibleLine {
		l.FirstVisibleLine = l.SelectedIndex
		l.Viewport.LineUp(1)
	}

	return message.StatusBarMsg{}
}

// LineDown increments `m.selectedIndex`
func (l *List[T]) LineDown() message.StatusBarMsg {
	if l.SelectedIndex < l.Length-1 {
		l.SelectedIndex++
	}

	// scroll down
	if l.SelectedIndex > l.LastVisibleLine {
		l.FirstVisibleLine = l.SelectedIndex - l.VisibleLines
		l.Viewport.LineDown(1)
	}

	return message.StatusBarMsg{}
}

// GoToTop moves the selection and viewport to the top of the tree
func (l *List[T]) GoToTop() message.StatusBarMsg {
	l.SelectedIndex = 0
	l.FirstVisibleLine = 0
	l.Viewport.GotoTop()
	return message.StatusBarMsg{}
}

// GoToBottom moves the selection and viewport to the bottom of the tree
func (l *List[T]) GoToBottom() message.StatusBarMsg {
	l.SelectedIndex = l.LastIndex
	l.FirstVisibleLine = l.Length - l.VisibleLines
	l.Viewport.GotoBottom()
	return message.StatusBarMsg{}
}

func RefreshList[T interface{ Refresh() }](a T) {
	a.Refresh()
}

// Rename renames the currently selected directory and
// returns a message that is displayed in the status bar
func (l *List[T]) Rename(origName string) message.StatusBarMsg {
	if l.EditIndex == nil {
		l.EditState = EditStates.Rename
		l.EditIndex = &l.SelectedIndex
		l.InputModel.SetValue(origName)
		// set cursor to last position
		l.InputModel.CursorEnd()
	}
	return message.StatusBarMsg{}
}

// TogglePinned pins or unpins the current selection
func (l *List[T]) TogglePinned(item T) {
	path := item.Path()

	// check if the selection already has a state
	p, err := l.Conf.MetaValue(path, config.Pinned)

	// set default state if not
	if err != nil {
		l.Conf.SetMetaValue(path, config.Pinned, "false")
		debug.LogErr(err)
	}

	// write to metadata file
	if p == "true" {
		l.Conf.SetMetaValue(path, config.Pinned, "false")
	} else {
		l.Conf.SetMetaValue(path, config.Pinned, "true")
	}

	l.PinnedItems.Toggle(item)
}

// CancelAction cancels the current action and blurs the editor
func (l *List[T]) CancelAction(cb func()) message.StatusBarMsg {
	l.resetEditor()
	cb()
	return message.StatusBarMsg{}
}

func (l *List[T]) resetEditor() {
	if l.EditState != EditStates.None {
		l.EditIndex = nil
		l.EditState = EditStates.None
		l.InputModel.Blur()
	}
}

func (l *List[T]) YankSelection(markCut bool) {}

func (l *List[T]) PasteSelection(item T, dirPath string, cb func(string)) {
	name := item.Name()

	if item.IsCut() {
		if item, ok := l.ItemsContain(item.Path()); ok {
			l.SelectedIndex = item.Index()
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

	header := l.theme.Header(l.Title, width, l.Focused()) + "\n"
	l.header = &header
	return header
}

func (l *List[T]) RefreshStyles() {
	l.Viewport.Style = l.theme.BaseColumnLayout(
		l.Size,
		l.Focused(),
	)
	l.Remove()
	l.BuildHeader(l.Size.Width, true)
}
