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
func (item Item) Index() int { return item.index }

// SetIndex returns the index of a Note-Item
func (item *Item) SetIndex(index int) { item.index = index }

// Path returns the index of a Note-Item
func (item Item) Path() string { return item.path }

// Path returns the index of a Note-Item
func (item *Item) SetPath(path string) { item.path = path }

// IsCut returns the index of a Note-Item
func (item Item) IsCut() bool { return item.isCut }

// SetIsCut returns the index of a Note-Item
func (item *Item) SetIsCut(isCut bool) { item.isCut = isCut }

// Name returns the name of a Note-Item
func (item Item) Name() string { return item.name }

// Name returns the name of a Note-Item
func (item *Item) SetName(name string) { item.name = name }

// InputModel returns the name of a Note-Item
func (item Item) InputModel() *textinput.Model { return item.inputModel }

// SetInputModel returns the name of a Note-Item
func (item *Item) SetInputModel(model textinput.Model) { item.inputModel = &model }

// Width returns the index of a Note-Item
func (item Item) Width() int { return item.width }

// SetWidth returns the index of a Note-Item
func (item *Item) SetWidth(width int) { item.width = width }

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

func (pi *PinnedItems[T]) Add(item T) {
	pi.Items = append(pi.Items, item)
}

// Contains returns whether a NoteItem is in `notes`
func (pi PinnedItems[T]) Contains(item T) bool {
	for _, n := range pi.Items {
		if n.Path() == item.Path() {
			return true
		}
	}
	return false
}

func (pi *PinnedItems[T]) Remove(item T) {
	for i, n := range pi.Items {
		if n.Path() == item.Path() {
			pi.Items = slices.Delete(pi.Items, i, i+1)
			return
		}
	}
}

// Toggle adds or removes the given note to the pinned notes
// depending on whether it's already in the slice
func (pi *PinnedItems[T]) Toggle(item T) {
	if !pi.Contains(item) {
		pi.Add(item)
	} else {
		pi.Remove(item)
	}
}

const ReservedLines = 1

// List represents a bubbletea model and holds items that implement ListItem.
// Important: T must be a pointer type, otherwise methods like SetIsCut() won't work.
// This allows us to mutate fields inside the list items when needed.
type List[T ListItem] struct {
	Component

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

func (list List[T]) MakeEmpty() List[T] {
	self := List[T]{
		SelectedIndex:    0,
		EditIndex:        nil,
		EditState:        EditStates.None,
		Items:            make([]T, 0),
		PinnedItems:      PinnedItems[T]{},
		FirstVisibleLine: 0,
		LastVisibleLine:  0,
	}

	return self
}

// UpdateViewportInfo synchronises the list's internal visible line count
// with the actual height of the viewport, subtracting `reservedLines`
// because I couldn't figure out why `VisibleLineCount()` seem to be more
// that it should be
//
// This ensures scrolling and item selection logic remain accurate after
// layout changes or terminal resizes.
func (list *List[T]) UpdateViewportInfo() {
	if list.VisibleLines != list.Viewport.VisibleLineCount() {
		list.VisibleLines = list.Viewport.VisibleLineCount() - ReservedLines
		list.LastVisibleLine = list.VisibleLines
	}
}

// SelectedItem returns the currently selected item of the list
func (list *List[T]) SelectedItem(items []T) T {
	var none T
	if list.Length == 0 {
		return none
	}

	if items == nil {
		items = list.Items
	}

	if list.SelectedIndex >= 0 && list.SelectedIndex <= list.Length {
		return items[list.SelectedIndex]
	}

	return none
}

// ItemsContain returns the ListItem with the given path from the List.
// If no such item exists, it returns a nil and an error.
func (list List[T]) ItemsContain(path string) (T, bool) {
	for i := range list.Items {
		if list.Items[i].Path() == path {
			return list.Items[i], true
		}
	}

	var none T
	return none, false
}

func (list List[T]) YankedItemsContain(path string) (T, bool) {
	for i := range list.YankedItems {
		item := list.YankedItems[i]
		if item.Path() == path {
			return item, true
		}
	}

	var none T
	return none, false
}

// indexByPath returns the interal list index by the given path
// if no items are provided it takes the cached items
func (list List[T]) IndexByPath(path string, items *[]T) int {
	if items == nil {
		items = &list.Items
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
func (list *List[T]) LineUp() message.StatusBarMsg {
	if list.SelectedIndex > 0 {
		list.SelectedIndex--
	}

	// scroll up
	if list.SelectedIndex < list.FirstVisibleLine {
		list.FirstVisibleLine = list.SelectedIndex
		list.Viewport.LineUp(1)
	}

	return message.StatusBarMsg{}
}

// LineDown increments `m.selectedIndex`
func (list *List[T]) LineDown() message.StatusBarMsg {
	if list.SelectedIndex < list.Length-1 {
		list.SelectedIndex++
	}

	// scroll down
	if list.SelectedIndex > list.LastVisibleLine {
		list.FirstVisibleLine = list.SelectedIndex - list.VisibleLines
		list.Viewport.LineDown(1)
	}

	return message.StatusBarMsg{}
}

// GoToTop moves the selection and viewport to the top of the tree
func (list *List[T]) GoToTop() message.StatusBarMsg {
	list.SelectedIndex = 0
	list.FirstVisibleLine = 0
	list.Viewport.GotoTop()
	return message.StatusBarMsg{}
}

// GoToBottom moves the selection and viewport to the bottom of the tree
func (list *List[T]) GoToBottom() message.StatusBarMsg {
	list.SelectedIndex = list.LastIndex
	list.FirstVisibleLine = list.Length - list.VisibleLines
	list.Viewport.GotoBottom()
	return message.StatusBarMsg{}
}

func RefreshList[T interface{ Refresh() }](a T) {
	a.Refresh()
}

// Rename renames the currently selected directory and
// returns a message that is displayed in the status bar
func (list *List[T]) Rename(origName string) message.StatusBarMsg {
	if list.EditIndex == nil {
		list.EditState = EditStates.Rename
		list.EditIndex = &list.SelectedIndex
		list.InputModel.SetValue(origName)
		// set cursor to last position
		list.InputModel.CursorEnd()
	}
	return message.StatusBarMsg{}
}

// TogglePinned pins or unpins the current selection
func (list *List[T]) TogglePinned(item T) {
	path := item.Path()

	// check if the selection already has a state
	p, err := list.Conf.MetaValue(path, config.Pinned)

	// set default state if not
	if err != nil {
		list.Conf.SetMetaValue(path, config.Pinned, "false")
		debug.LogErr(err)
	}

	// write to metadata file
	if p == "true" {
		list.Conf.SetMetaValue(path, config.Pinned, "false")
	} else {
		list.Conf.SetMetaValue(path, config.Pinned, "true")
	}

	list.PinnedItems.Toggle(item)
}

// CancelAction cancels the current action and blurs the editor
func (list *List[T]) CancelAction(cb func()) message.StatusBarMsg {
	list.resetEditor()
	cb()
	return message.StatusBarMsg{}
}

func (list *List[T]) resetEditor() {
	if list.EditState != EditStates.None {
		list.EditIndex = nil
		list.EditState = EditStates.None
		list.InputModel.Blur()
	}
}

func (list *List[T]) YankSelection(markCut bool) {}

func (list *List[T]) PasteSelection(item T, dirPath string, cb func(string)) {
	name := item.Name()

	if item.IsCut() {
		if item, ok := list.ItemsContain(item.Path()); ok {
			list.SelectedIndex = item.Index()
			item.SetIsCut(false)
			return
		}
	}

	newPath := dirPath + "/" + name

	// Ensure we always have a valid path
	if list.isNote(item.Path()) {
		newPath = notes.GetValidPath(newPath, true)
	} else {
		newPath = directories.GetValidPath(newPath)
	}

	cb(newPath)
}

// checkName ensures that the note name does not conflict with existing notes
// in the specified directory. If a conflict exists, it appends " Copy" to the name.
func (list *List[T]) CheckName(dirPath string, name string) string {
	if list.isNote(name) {
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

func (list *List[T]) isNote(name string) bool {
	return strings.HasSuffix(name, notes.Ext) ||
		strings.HasSuffix(name, notes.LegacyExt) ||
		strings.HasSuffix(name, notes.ConfExt)
}

func (list *List[T]) ConfirmRemove() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (list *List[T]) Remove() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (list *List[T]) Refresh(
	resetSelectedIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	return message.StatusBarMsg{}
}

// BuildHeader builds title of the directory tree column
func (list *List[T]) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if list.header != nil && !rebuild {
		if width == lipgloss.Width(*list.header) {
			return *list.header
		}
	}

	header := list.theme.Header(list.Title(), width, list.Focused()) + "\n"
	list.header = &header
	return header
}

func (list *List[T]) RefreshStyles() {
	list.Viewport.Style = list.theme.BaseColumnLayout(
		list.Size,
		list.Focused(),
	)
	list.Remove()
	list.BuildHeader(list.Size.Width, true)
}
