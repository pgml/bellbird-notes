package directorytree

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/directories"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components/statusbar"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// TreeItem represents a single directory tree row
type TreeItem struct {
	shared.Item

	// The parent index of the directory.
	// Used to make expanding and collapsing a directory possible
	// using DirectoryTree.dirsListFlat
	parent      int
	children    []*TreeItem
	isLastChild bool

	// Indicates whether a directory is expanded
	expanded bool

	// Indicates the depth of a directory
	// Used to determine the indentation of DirectoryTree.dirsFlatList
	level int

	// the amount of notes a directory contains
	NbrNotes int

	// the amount of sub directories a directory has
	NbrFolders int

	// Stores the rendered item row icon
	Icon string

	// Stores the rendered toggle arrow icon
	ToggleArrow string

	// Stores the visual representation of the indentation level
	Indent string
}

// Expanded returns the name of a Dir-Item
func (d TreeItem) Expanded() bool {
	return d.expanded
}

func (d *TreeItem) Expand() {
	d.expanded = true
}

func (d *TreeItem) Collapse() {
	d.expanded = false
}

func (d *TreeItem) SetExpanded(expand bool) {
	if expand {
		d.Expand()
	} else {
		d.Collapse()
	}
}

// setIndentation sets the visual indentation for the tree item based on its level
// and whether line markers (like │ or ╰) should be shown.
func (d *TreeItem) setIndentation(indentLines bool) {
	indentStr := "  "
	if indentLines {
		//if d.isLastChild {
		//	indentStr = "╰ "
		//} else {
		//	indentStr = "│ "
		//}
		indentStr = "│ "
	}

	// repeat indentation once per indentation level
	indent := strings.Repeat(indentStr, d.level)

	// width of indentation per level which needs to be subtracted later rom
	// general item width to prevent row breaks
	indentWidth := lipgloss.Width(indent)
	style := d.Styles.Indent.Width(indentWidth)

	if d.IsSelected {
		style = style.Background(theme.ColourBgSelected)
	}

	// store rendered indentation
	d.Indent = style.Render(indent)

	// subtract the indentation width from the item width
	d.SetWidth(d.Width() - indentWidth)
}

// setIcon sets the icon representing a folder state (open/closed).
// If the nerd fonts settings is disabled, nothing will be shown
func (d *TreeItem) setIcon() {
	folderClosed := ""
	folderOpen := ""

	if d.NerdFonts {
		folderClosed = theme.IconDirClosed.Nerd
		folderOpen = theme.IconDirOpen.Nerd
	}

	style := d.Styles.Icon.Width(d.Styles.IconWidth)
	if d.IsSelected {
		style = d.Styles.Selected.Width(d.Styles.IconWidth)
	}

	iconDir := folderClosed

	if d.IsPinned {
		iconDir = theme.Icon(theme.IconPin, d.NerdFonts)
		style = style.Foreground(theme.ColourBorderFocused)
	} else if d.Expanded() {
		iconDir = folderOpen
	}

	// store rendered icon
	d.Icon = style.Render(iconDir)

	// subtract the indentation width from the item width
	d.SetWidth(d.Width() - d.Styles.IconWidth)
}

// setToggleArrow sets the arrow icon used to expand/collapse tree items.
// Hides the arrow if the item has no children.
func (d *TreeItem) setToggleArrow() {
	iconArrow := theme.IconDirClosed.Alt
	if d.Expanded() {
		iconArrow = theme.IconDirOpen.Alt
	}

	if len(d.children) == 0 {
		iconArrow = ""
	}

	style := d.Styles.Toggle
	if d.IsSelected {
		style = d.Styles.Selected.Width(d.Styles.ToggleWidth)
	}

	// store rendered toggle arrow
	d.ToggleArrow = style.Render(iconArrow)

	// subtract the indentation width from the item width
	d.SetWidth(d.Width() - d.Styles.ToggleWidth)
}

// prepareRow initialises all visual elements (indent, icon, arrow) for the row.
func (d *TreeItem) prepareRow(showIndentLines bool) {
	d.setIndentation(showIndentLines)
	d.setIcon()
	d.setToggleArrow()
}

// String renders the complete visual representation of the tree item.
// If input is true render the input view to allow renaming/creating.
func (d TreeItem) String(input bool) string {
	if d.Width() <= 0 {
		return ""
	}

	baseStyle := d.Styles.Base.Width(d.Width())
	selectedStyle := d.Styles.Selected.Width(d.Width())
	if d.IsSelected {
		baseStyle = selectedStyle
	}

	name := utils.TruncateText(d.Name(), d.Width()-1)
	name = baseStyle.Render(name)

	// replace name with the stored input view if we are creating or
	// renaming
	if input {
		name = d.InputModel().View()
		d.Icon = ""
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		d.Indent, d.ToggleArrow, d.Icon, name,
	)
}

// DirectoryTree represents the bubbletea model.
type DirectoryTree struct {
	shared.List[*TreeItem]

	// A flattened representation to make vertical navigation easier
	dirsListFlat []*TreeItem

	// Stores currently expanded directories
	expandedDirs map[string]bool

	indentLines bool
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (t *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (t *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// focus the input field when renaming a list item
		if t.EditIndex != nil && !t.InputModel.Focused() {
			t.InputModel.Focus()
			return t, nil
		}

		if t.InputModel.Focused() {
			t.InputModel, cmd = t.InputModel.Update(msg)
			return t, cmd
		}

	case tea.WindowSizeMsg:
		t.Size.Width = msg.Width
		t.Size.Height = msg.Height

		if !t.IsReady {
			t.Viewport = viewport.New()
			t.Viewport.SetContent(t.viewportContent())
			t.Viewport.KeyMap = viewport.KeyMap{}
			t.LastVisibleLine = t.Viewport.VisibleLineCount() - shared.ReservedLines
			t.IsReady = true
		} else {
			t.Viewport.SetWidth(t.Size.Width)
			t.Viewport.SetHeight(t.Size.Height)
		}
	}

	t.Viewport, cmd = t.Viewport.Update(msg)

	return t, cmd
}

func (t *DirectoryTree) View() tea.View {
	var view tea.View
	view.SetContent(t.Content())
	return view
}

// NewDirectoryTree creates a new model with default settings.
func New(conf *config.Config) *DirectoryTree {
	var list shared.List[*TreeItem]
	list.MakeEmpty()
	list.Conf = conf
	list.Title = "FOLDERS"

	tree := &DirectoryTree{
		List:         list,
		expandedDirs: make(map[string]bool),
	}

	tree.SetTheme(theme.New(conf))
	tree.checkVisibility()
	tree.checkIndentLines()
	tree.InputModel = tree.TreeInput()

	var item shared.Item
	item.NerdFonts = conf.NerdFonts()
	item.Styles = shared.DirTreeStyle()
	item.SetName(app.Name())
	item.SetPath(tree.NotesDir())

	// append root directory
	tree.Items = append(tree.Items, &TreeItem{
		Item:     item,
		expanded: true,
		level:    0,
		parent:   -1,
		children: tree.getChildren(tree.NotesDir(), 0),
	})

	tree.build()
	tree.SelectLastDir()
	return tree
}

func (t DirectoryTree) Name() string {
	return "Folders"
}

func (t DirectoryTree) NotesDir() string {
	// fetch notes directory
	notesDir, err := t.Conf.NotesDir()
	if err != nil {
		debug.LogErr(err)
		return ""
	}

	return notesDir
}

// Input returns and textinput model tailored to the directory tree
func (t *DirectoryTree) TreeInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = theme.Icon(theme.IconPen, t.Conf.NerdFonts()) + " "
	ti.CharLimit = 100
	ti.VirtualCursor = true

	bgSelected := shared.DirTreeStyle().Selected
	ti.Styles.Focused = textinput.StyleState{
		Text:   bgSelected,
		Prompt: bgSelected,
	}

	return ti
}

func (t *DirectoryTree) RefreshSize() {
	vp := t.Viewport
	if vp.Width() != t.Size.Width && vp.Height() != t.Size.Height {
		t.Viewport.SetWidth(t.Size.Width)
		t.Viewport.SetHeight(t.Size.Height)
	}
}

func (t *DirectoryTree) Content() string {
	if !t.IsReady {
		return "\n  Initializing..."
	}

	if !t.Visible() {
		return ""
	}

	t.Viewport.SetContent(t.viewportContent())
	t.UpdateViewportInfo()
	t.Viewport.EnsureVisible(t.SelectedIndex, 0, 0)

	t.Viewport.Style = t.Theme().BaseColumnLayout(
		t.Size,
		t.Focused(),
	)

	var view strings.Builder
	view.WriteString(t.BuildHeader(t.Size.Width, false))
	view.WriteString(t.Viewport.View())
	return view.String()
}

func (t *DirectoryTree) viewportContent() string {
	var tree strings.Builder

	for i, dir := range t.dirsListFlat {
		// Removes invalid directory items
		// index and parent 0 shouldn't be possible but
		//sometime occurs after certain user actions
		if dir.Index() == 0 && dir.parent == 0 {
			t.dirsListFlat = slices.Delete(t.dirsListFlat, i, i+1)
			continue
		}

		if t.LastIndex == dir.Index() {
			dir.isLastChild = true
		}

		dir.IsSelected = (t.SelectedIndex == i)

		if *app.Debug {
			// prepend tree item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			tree.WriteString(style.Render(fmt.Sprintf("%02d", dir.Index())))
			tree.WriteString(" ")
		}

		if t.Size.Width > 0 {
			dir.SetWidth(t.Size.Width - 2)
		}

		// Prepare all visual row elements and get the correct width.
		// This has to be called before any actuall output
		dir.prepareRow(t.indentLines)

		// Set the correct input width so that in case the folder name is too
		// long we're not breaking to the next line
		t.InputModel.SetWidth(dir.Width() - 1)
		dir.SetInputModel(t.InputModel)

		// get text input if there's an edit index which likely means
		// we're renaming or creating
		isInput := t.EditIndex != nil && i == *t.EditIndex

		tree.WriteString(dir.String(isInput))
		tree.WriteByte('\n')
	}

	return tree.String()
}

// build prepares t.dirsListFlat for rendering
// checking directory states etc.
func (t *DirectoryTree) build() {
	t.refreshFlatList()
	t.Length = len(t.dirsListFlat)
	t.LastIndex = t.dirsListFlat[len(t.dirsListFlat)-1].Index()
}

// getChildren reads a directory and returns a slice of a directory Dir
func (t *DirectoryTree) getChildren(path string, level int) []*TreeItem {
	var dirs []*TreeItem
	childDir, _ := directories.List(path)

	// pinned stuff
	if !t.PinnedItems.IsLoaded {
		// reset pinned and refetch pinned notes when we entered a new directory
		t.PinnedItems.Items = make([]*TreeItem, 0, len(childDir))
		for _, dir := range childDir {
			if dir.IsPinned {
				item := t.createDirectoryItem(dir, -1, true)
				t.PinnedItems.Add(&item)
			}
		}
	}

	pinnedMap := make(map[string]struct{}, len(t.PinnedItems.Items))
	for _, n := range t.PinnedItems.Items {
		pinnedMap[n.Path()] = struct{}{}
	}

	var (
		pinnedItems   []*TreeItem
		unpinnedItems []*TreeItem
	)

	for _, dir := range childDir {
		_, isPinned := pinnedMap[dir.Path]
		dirItem := t.createDirectoryItem(dir, level, isPinned)

		if dir.IsExpanded {
			t.expandedDirs[dir.Path] = dir.IsExpanded
		}

		dirItem.SetExpanded(dir.IsExpanded)

		if isPinned {
			pinnedItems = append(pinnedItems, &dirItem)
		} else {
			unpinnedItems = append(unpinnedItems, &dirItem)
		}
	}

	dirs = append(pinnedItems, unpinnedItems...)
	t.PinnedItems.IsLoaded = true

	return dirs
}

// createDirectoryItem creates a directory item
func (m *DirectoryTree) createDirectoryItem(
	dir directories.Directory,
	level int,
	isPinned bool,
) TreeItem {
	style := shared.DirTreeStyle()

	var item shared.Item
	item.SetIndex(0)
	item.SetName(dir.Name())
	item.SetPath(dir.Path)
	item.NerdFonts = m.Conf.NerdFonts()
	item.IsPinned = isPinned
	item.Styles = style

	dirItem := TreeItem{
		Item:       item,
		expanded:   dir.IsExpanded,
		parent:     0,
		children:   m.getChildren(dir.Path, level+1),
		NbrFolders: dir.NbrFolders,
		level:      level,
		NbrNotes:   dir.NbrNotes,
	}

	return dirItem
}

// createVirtualDir creates a temporary, virtual directory `Dir`
//
// This directory is mainly used as a placeholder when creating a directory
// and is not actually written to the file system.
func (t *DirectoryTree) createVirtualDir() TreeItem {
	selectedDir := t.SelectedDir()
	tempFolderName := "New Folder"
	tempFolderPath := filepath.Join(
		selectedDir.Path(),
		tempFolderName,
	)
	indent := selectedDir.level + 1

	var item shared.Item
	item.SetIndex(len(t.dirsListFlat))
	item.SetName(tempFolderName)
	item.SetPath(tempFolderPath)

	return TreeItem{
		Item:     item,
		expanded: false,
		children: nil,
		parent:   selectedDir.Index(),
		level:    indent,
	}
}

// Refresh updates the currently selected tree branch
//
// If `resetIndex` is set to true, 't.selectedIndex' will be set to -1
// which means the selected directory's parent
func (t *DirectoryTree) Refresh(
	resetIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if t.EditState == shared.EditStates.Create {
		resetIndex = true
	}

	if sel := t.SelectedDir(); sel != nil {
		selectAfter := t.SelectedIndex
		if resetIndex {
			selectAfter = -1
		}

		t.RefreshBranch(
			t.SelectedDir().parent,
			selectAfter,
		)
	} else {
		// If for some reason there is no selected directory and
		// refresh is triggered just select root
		if t.SelectedIndex != 0 {
			t.SelectedIndex = 0
			t.Refresh(false, false)
		}
	}

	t.checkVisibility()
	t.checkIndentLines()

	return statusMsg
}

// RefreshBranch refreshes a tree branch by its branch index
//
// Use `selectAfter` to change the selected tree item after the branch got refreshed.
// If `selectAfter` is -1 the branch's parent is selected
func (t *DirectoryTree) RefreshBranch(index int, selectAfter int) {
	t.SelectedIndex = index

	if sel := t.SelectedDir(); sel != nil {
		if dir := findDirInTree(t.Items, sel.Path()); dir != nil {
			dir.children = t.getChildren(dir.Path(), dir.level+1)
		}
	}

	if selectAfter == -1 {
		selectAfter = index
	}

	t.SelectedIndex = selectAfter
	t.build()
}

// SelectedDir returns the currently selected directory in the directory tree
func (t *DirectoryTree) SelectedDir() *TreeItem {
	return t.SelectedItem(t.dirsListFlat)
}

// refreshFlatList reorganises the one-dimensional directory tree
func (t *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	t.dirsListFlat = t.flatten(t.Items, 0, -1, &nextIndex)
}

// flatten converts a slice of Dir and its sub slices into
// a one dimensional slice that we use to render the directory tree
func (t *DirectoryTree) flatten(
	dirs []*TreeItem,
	level int,
	parent int,
	nextIndex *int,
) []*TreeItem {
	var result []*TreeItem
	for i, dir := range dirs {
		dir.SetIndex(*nextIndex)
		dir.parent = parent
		dir.level = level

		*nextIndex++

		if _, contains := t.expandedDirs[dir.Path()]; contains {
			dir.Expand()
		}

		result = append(result, dir)

		if !dir.Expanded() {
			continue
		}

		children := dirs[i].children
		result = append(
			result,
			t.flatten(children, level+1, dir.Index(), nextIndex)...,
		)
	}
	return result
}

// lastChildOfSelection returns the corresponding last child
// of the selected directory.
func (t *DirectoryTree) lastChildOfSelection() *TreeItem {
	selectedDir := t.SelectedDir()
	lastChild := t.getLastChild(selectedDir.Index())

	if lastChild.Expanded() && len(lastChild.children) > 0 {
		lastChild = t.getLastChild(lastChild.Index())
	}
	return lastChild
}

// getLastChild returns the last child of the item with the given index
//
// if `createEmpty` is set to true, we attempt to create an empty
// Dir{}
func (t *DirectoryTree) getLastChild(index int) *TreeItem {
	lastChild := t.dirsListFlat[len(t.dirsListFlat)-1]
	dir := t.dirsListFlat[index]

	// If the selected directory is root, bail out early
	// with the last item of the flattend directory tree
	if dir.Index() == 0 {
		return lastChild
	}

	if len(dir.children) > 0 {
		lastChild = dir.children[len(dir.children)-1]
		if lastChild.Name() == "" {
			lastChild = dir
		}
		for _, dir := range t.dirsListFlat {
			if lastChild.Name() == dir.Name() {
				lastChild = dir
			}
		}
	}

	return lastChild
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// of the directories.
// To make it persistent write it to the file system
func (m *DirectoryTree) insertDirAfter(afterIndex int, directory TreeItem) {
	for i, dir := range m.dirsListFlat {
		if dir.Index() == afterIndex {
			m.dirsListFlat = append(
				m.dirsListFlat[:i+1],
				append([]*TreeItem{&directory}, m.dirsListFlat[i+1:]...)...,
			)
			break
		}
	}
}

func (t *DirectoryTree) dirExists(dirPath string) bool {
	parentPath := filepath.Dir(dirPath)
	dirName := filepath.Base(dirPath)

	if _, contains := directories.ContainsDir(
		parentPath,
		dirName,
	); contains {
		//statusMsg = message.StatusBarMsg{
		//	Content: "Directory already exists, please choose another name.",
		//	Type:    message.Error,
		//	Sender:  message.SenderDirTree,
		//}
		return true
	}
	return false
}

// findDirInTree recursively searches for a directory by its path
func findDirInTree(directories []*TreeItem, path string) *TreeItem {
	for i := range directories {
		if directories[i].Path() == path {
			return directories[i]
		}

		if directories[i].Expanded() {
			if ok := findDirInTree(
				directories[i].children,
				path,
			); ok != nil {
				return ok
			}
		}
	}
	return nil
}

// SelectLastDir selects the last directory
func (t *DirectoryTree) SelectLastDir() string {
	dirPath, err := t.Conf.MetaValue("", config.LastDirectory)
	if err == nil && dirPath != "" {
		for i := range t.dirsListFlat {
			if t.dirsListFlat[i].Path() != dirPath {
				continue
			}

			index := t.dirsListFlat[i].Index()
			t.SelectedIndex = index
			return dirPath
		}
	}
	return ""
}

///
/// keyboard shortcut commands
///

// Collapse collapses the currently selected directory
func (t *DirectoryTree) Collapse() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}
	if t.SelectedIndex >= len(t.dirsListFlat) {
		return statusMsg
	}

	items := t.Items
	path := t.SelectedDir().Path()

	if dir := findDirInTree(items, path); dir != nil {
		if dir.Expanded() {
			// remove expanded state from cached map
			delete(t.expandedDirs, dir.Path())
			dir.Collapse()

			// rebuild directory tree
			t.build()
		}

		// save state to meta config file
		t.Conf.SetMetaValue(dir.Path(), config.Expanded, "false")
	}

	return statusMsg
}

// Expand expands the currently selected directory
func (t *DirectoryTree) Expand() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}
	if t.SelectedIndex >= len(t.dirsListFlat) ||
		len(t.SelectedDir().children) == 0 {

		return statusMsg
	}

	items := t.Items
	path := t.SelectedDir().Path()

	if dir := findDirInTree(items, path); dir != nil {
		if !dir.Expanded() {
			// add expanded state to cached map
			t.expandedDirs[dir.Path()] = true
			dir.children = t.getChildren(dir.Path(), dir.level+1)
			dir.Expand()

			// rebuild directory tree
			t.build()

			// save state to meta config file
			t.Conf.SetMetaValue(dir.Path(), config.Expanded, "true")
		}
	}

	return statusMsg
}

// Create creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (t *DirectoryTree) Create(
	mi *mode.ModeInstance,
	statusBar *statusbar.StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if t.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		t.EditState = shared.EditStates.Create
		// get a fresh version of the tree to work with
		t.refreshFlatList()
		// expand the selected directory for a live preview
		// of the new directory
		t.Expand()

		vrtDir := t.createVirtualDir()
		selDir := t.SelectedDir()

		// if the selected directory has no children yet
		// we append and empty Dir so that we get a correct result
		if selDir.Index() != 0 && len(selDir.children) == 0 {
			selDir.children = append(selDir.children, &TreeItem{})
		}

		lastChild := t.lastChildOfSelection()
		t.insertDirAfter(lastChild.Index(), vrtDir)

		// update the selected index to the virtual directory
		// so that we input the name at the correct position
		t.SelectedIndex = lastChild.Index() + 1

		if t.EditIndex == nil {
			index := t.SelectedIndex
			t.EditIndex = &index
			t.InputModel.SetValue(vrtDir.Name())
			t.InputModel.CursorEnd()
		}
	}

	return statusMsg
}

// ConfirmRemove returns a status bar prompt
// to confirm or cancel the removal of a directory
func (t *DirectoryTree) ConfirmRemove() message.StatusBarMsg {
	selectedDir := t.SelectedDir()

	// prevent deleting root directory
	if selectedDir == nil || t.SelectedIndex == 0 {
		return message.StatusBarMsg{Type: message.None}
	}

	t.EditState = shared.EditStates.Delete

	rootDir, _ := app.NotesRootDir()
	path := strings.ReplaceAll(selectedDir.Path(), rootDir, ".")
	resultMsg := fmt.Sprintf(message.StatusBar.RemovePromptDirContent, path)

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    message.PromptError,
		Sender:  message.SenderDirTree,
		Column:  sbc.General,
	}
}

// Remove removes the currently selected directory
func (t *DirectoryTree) Remove() message.StatusBarMsg {
	dir := t.SelectedDir()
	index := t.SelectedIndex
	parent := dir.parent
	resultMsg := ""
	msgType := message.Success

	if err := directories.Delete(dir.Path(), true); err == nil {
		// delete the directory from the flat list
		t.dirsListFlat = slices.Delete(
			t.dirsListFlat,
			index,
			index+1,
		)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	t.RefreshBranch(parent, index)

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  sbc.General,
	}
}

// ConfirmAction confirms a user action
func (t *DirectoryTree) ConfirmAction() message.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if t.EditIndex != nil {
		selDir := t.SelectedDir()
		oldPath := selDir.Path()

		// build the new path with the new name
		newPath := filepath.Join(
			filepath.Dir(oldPath),
			t.InputModel.Value(),
		)

		switch t.EditState {
		case shared.EditStates.Rename:
			if err := directories.Rename(oldPath, newPath); err != nil {
				debug.LogErr(err)
			}

		case shared.EditStates.Create:
			if !t.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

		// selected the newly renamed or created directory
		t.Refresh(false, false)
		t.SelectedIndex = t.IndexByPath(newPath, &t.dirsListFlat)

		t.CancelAction(func() {
			t.Refresh(false, false)
		})

		return message.StatusBarMsg{Content: "yep"}
	}

	return message.StatusBarMsg{}
}

// ContentInfo returns the info about the currently selected directory
// in the status bar
func (t *DirectoryTree) ContentInfo() message.StatusBarMsg {
	sel := t.SelectedDir()
	iconDir := theme.Icon(theme.IconDirClosed, t.Conf.NerdFonts())
	iconNotes := theme.Icon(theme.IconNote, t.Conf.NerdFonts())

	var nbrFolders strings.Builder
	nbrFolders.WriteString(iconDir)
	nbrFolders.WriteByte(' ')
	nbrFolders.WriteString(strconv.Itoa(sel.NbrFolders))
	nbrFolders.WriteString(" Folders")

	var nbrNotes strings.Builder
	nbrNotes.WriteString(iconNotes)
	nbrNotes.WriteByte(' ')
	nbrNotes.WriteString(strconv.Itoa(sel.NbrNotes))
	nbrNotes.WriteString(" Notes")

	var msg strings.Builder
	msg.WriteString(nbrFolders.String())
	msg.WriteString(", ")
	msg.WriteString(nbrNotes.String())

	return message.StatusBarMsg{
		Column:  sbc.FileInfo,
		Content: msg.String(),
	}
}

func (t *DirectoryTree) TogglePinnedItems() message.StatusBarMsg {
	dir := t.SelectedDir()

	t.TogglePinned(dir)
	t.Refresh(false, false)

	// get the new index and select the newly pinned or unpinned note
	// since the pinned notes are always at the top and the notes order
	// is changed
	for i, it := range t.dirsListFlat {
		if it.Path() == dir.Path() {
			t.SelectedIndex = i
		}
	}

	return message.StatusBarMsg{}
}

// YankSelection clears the yankedItems list and adds the currently selected item
// from the NotesList to it. This simulates copying an item for later pasting.
func (t *DirectoryTree) YankSelection(markCut bool) {
	sel := t.SelectedDir()
	sel.SetIsCut(markCut)

	t.YankedItems = []*TreeItem{}
	t.YankedItems = append(t.YankedItems, sel)
}

// PasteSelection duplicates all yanked notes into the specified directory path.
// It handles name conflicts by appending " Copy" to the note name until a unique
// path is found. Returns an error if any note cannot be created.
func (t *DirectoryTree) PasteSelectedItems() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	sel := t.SelectedDir()

	for _, dir := range t.YankedItems {
		t.PasteSelection(dir, sel.Path(), func(newPath string) {
			err := directories.Copy(dir.Path(), newPath)

			if err != nil {
				debug.LogErr(err)
				return
			}

			//t.Refresh(false, false)
			t.RefreshBranch(sel.Index(), -1)
			t.Expand()

			// select the currently pasted item
			if dir, ok := t.ItemsContain(newPath); ok {
				t.SelectedIndex = dir.Index()
			}

			// Remove the original note if it's marked for moving (cut)
			if dir.IsCut() {
				if err := directories.Delete(dir.Path(), true); err != nil {
					debug.LogErr(err)
				}
			}
		})
	}

	return statusMsg
}

// ItemsContain returns the ListItem with the given path from the List.
// If no such item exists, it returns a nil and an error.
func (t *DirectoryTree) ItemsContain(path string) (*TreeItem, bool) {
	for _, item := range t.dirsListFlat {
		if item.Path() == path {
			return item, true
		}
	}

	return nil, false
}

func (t *DirectoryTree) ToggleIndentLines() message.StatusBarMsg {
	t.indentLines = !t.indentLines

	t.Conf.SetValue(
		config.Folders,
		config.IndentLines,
		strconv.FormatBool(t.indentLines),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (t *DirectoryTree) Toggle() message.StatusBarMsg {
	t.ToggleVisibility()

	t.Conf.SetValue(
		config.Folders,
		config.Visible,
		strconv.FormatBool(t.Visible()),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (t *DirectoryTree) checkIndentLines() {
	lines, err := t.Conf.Value(config.Folders, config.IndentLines)

	if err != nil {
		debug.LogErr(err)
	}

	t.indentLines = lines.GetBool()
}

func (t *DirectoryTree) checkVisibility() {
	vis, err := t.Conf.Value(config.Folders, config.Visible)

	if err != nil {
		debug.LogErr(err)
	}

	t.SetVisibility(vis.GetBool())
}
