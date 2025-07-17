package components

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
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// TreeItem represents a single directory tree row
type TreeItem struct {
	Item

	// The parent index of the directory.
	// Used to make expanding and collapsing a directory possible
	// using DirectoryTree.dirsListFlat
	parent      int
	children    []TreeItem
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

	// Indicates whether the directory should be pinnend
	isPinned *bool

	// Stores the rendered item row icon
	Icon string

	// Stores the rendered toggle arrow icon
	ToggleArrow string

	// Stores the visual representation of the indentation level
	Indent string
}

// Index returns the index of a Dir-Item
func (d TreeItem) Index() int {
	return d.index
}

// Path returns the path of a Dir-Item
func (d TreeItem) Path() string {
	return d.path
}

// Name returns the name of a Dir-Item
func (d TreeItem) Name() string {
	return d.name
}

// Expanded returns the name of a Dir-Item
func (d TreeItem) Expanded() bool {
	return d.expanded
}

// setIndentation sets the visual indentation for the tree item based on its level
// and whether line markers (like │ or ╰) should be shown.
func (d *TreeItem) setIndentation(indentLines bool) {
	indentStr := "  "
	if indentLines {
		if d.isLastChild {
			indentStr = "╰ "
		} else {
			indentStr = "│ "
		}
	}

	// repeat indentation once per indentation level
	indent := strings.Repeat(indentStr, d.level)

	// width of indentation per level which needs to be subtracted later rom
	// general item width to prevent row breaks
	indentWidth := lipgloss.Width(indent)
	style := d.styles.indent.Width(indentWidth)

	if d.selected {
		style = style.Background(theme.ColourBgSelected)
	}

	// store rendered indentation
	d.Indent = style.Render(indent)

	// subtract the indentation width from the item width
	d.width -= indentWidth
}

// setIcon sets the icon representing a folder state (open/closed).
// If the nerd fonts settings is disabled, nothing will be shown
func (d *TreeItem) setIcon() {
	folderClosed := ""
	folderOpen := ""

	if d.nerdFonts {
		folderClosed = theme.IconDirClosed.Nerd
		folderOpen = theme.IconDirOpen.Nerd
	}

	iconDir := folderClosed
	if d.expanded {
		iconDir = folderOpen
	}

	style := d.styles.icon.Width(d.styles.iconWidth)
	if d.selected {
		style = d.styles.selected.Width(d.styles.iconWidth)
	}

	// store rendered icon
	d.Icon = style.Render(iconDir)

	// subtract the indentation width from the item width
	d.width -= d.styles.iconWidth
}

// setToggleArrow sets the arrow icon used to expand/collapse tree items.
// Hides the arrow if the item has no children.
func (d *TreeItem) setToggleArrow() {
	iconArrow := theme.IconDirClosed.Alt
	if d.expanded {
		iconArrow = theme.IconDirOpen.Alt
	}

	if len(d.children) == 0 {
		iconArrow = ""
	}

	style := d.styles.toggle
	if d.selected {
		style = d.styles.selected.Width(d.styles.toggleWidth)
	}

	// store rendered toggle arrow
	d.ToggleArrow = style.Render(iconArrow)

	// subtract the indentation width from the item width
	d.width -= d.styles.toggleWidth
}

// prepareRow initialises all visual elements (indent, icon, arrow) for the row.
func (d *TreeItem) prepareRow() {
	d.setIndentation(false)
	d.setIcon()
	d.setToggleArrow()
}

// String renders the complete visual representation of the tree item.
// If input is true render the input view to allow renaming/creating.
func (d *TreeItem) String(input bool) string {
	if d.width <= 0 {
		return ""
	}

	baseStyle := d.styles.base.Width(d.width)
	selectedStyle := d.styles.selected.Width(d.width)
	if d.selected {
		baseStyle = selectedStyle
	}

	name := utils.TruncateText(d.Name(), d.width-1)
	name = baseStyle.Render(name)

	// replace name with the stored input view if we are creating or
	// renaming
	if input {
		name = d.input.View()
		d.Icon = ""
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		d.Indent, d.ToggleArrow, d.Icon, name,
	)
}

// DirectoryTree represents the bubbletea model.
type DirectoryTree struct {
	List[TreeItem]

	// A flattened representation to make vertical navigation easier
	dirsListFlat []TreeItem

	// Stores currently expanded directories
	expandedDirs map[string]bool
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
		if t.editIndex != nil && !t.input.Focused() {
			t.input.Focus()
			return t, nil
		}

		if t.input.Focused() {
			t.input, cmd = t.input.Update(msg)
			return t, cmd
		}

	case tea.WindowSizeMsg:
		t.Size.Width = msg.Width
		t.Size.Height = msg.Height

		if !t.Ready {
			t.viewport = viewport.New()
			t.viewport.SetContent(t.render())
			t.viewport.KeyMap = viewport.KeyMap{}
			t.lastVisibleLine = t.viewport.VisibleLineCount() - reservedLines
			t.Ready = true
		} else {
			t.viewport.SetWidth(t.Size.Width)
			t.viewport.SetHeight(t.Size.Height)
		}

	}

	t.viewport, cmd = t.viewport.Update(msg)

	return t, cmd
}

func (t *DirectoryTree) RefreshSize() {
	vp := t.viewport
	if vp.Width() != t.Size.Width && vp.Height() != t.Size.Height {
		t.viewport.SetWidth(t.Size.Width)
		t.viewport.SetHeight(t.Size.Height)
	}
}

func (t *DirectoryTree) View() string {
	if !t.Ready {
		return "\n  Initializing..."
	}

	t.viewport.SetContent(t.render())
	t.UpdateViewportInfo()

	t.viewport.Style = theme.BaseColumnLayout(
		t.Size,
		t.Focused(),
	)

	var view strings.Builder
	view.WriteString(t.BuildHeader(t.Size.Width, false))
	view.WriteString(t.viewport.View())
	return view.String()
}

// NewDirectoryTree creates a new model with default settings.
func NewDirectoryTree(conf *config.Config) *DirectoryTree {
	tree := &DirectoryTree{
		List: List[TreeItem]{
			selectedIndex: 0,
			editIndex:     nil,
			EditState:     EditStates.None,
			items:         make([]TreeItem, 0),
			conf:          conf,
		},
		expandedDirs: make(map[string]bool),
	}

	tree.input = tree.Input()

	// fetch notes directory
	notesDir, err := conf.Value(
		config.General,
		config.NotesDirectory,
	)

	if err != nil {
		debug.LogErr(err)
	}

	// append root directory
	tree.items = append(tree.items, TreeItem{
		Item: Item{
			index:     0,
			name:      app.Name(),
			path:      notesDir,
			styles:    DirTreeStyle(),
			nerdFonts: conf.NerdFonts(),
		},
		expanded: true,
		level:    0,
		parent:   -1,
		children: tree.getChildren(notesDir, 0),
	})

	tree.build()
	tree.SelectLastDir()
	return tree
}

// Input returns and textinput model tailored to the directory tree
func (t *DirectoryTree) Input() textinput.Model {
	ti := textinput.New()
	ti.Prompt = theme.Icon(theme.IconPen, t.conf.NerdFonts()) + " "
	ti.CharLimit = 100
	ti.VirtualCursor = true

	bgSelected := DirTreeStyle().selected
	ti.Styles.Focused = textinput.StyleState{
		Text:   bgSelected,
		Prompt: bgSelected,
	}

	return ti
}

// build prepares t.dirsListFlat for rendering
// checking directory states etc.
func (t *DirectoryTree) build() {
	t.refreshFlatList()
	t.length = len(t.dirsListFlat)
	t.lastIndex = t.dirsListFlat[len(t.dirsListFlat)-1].index
}

// BuildHeader builds title of the directory tree column
func (t *DirectoryTree) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if t.header != nil && !rebuild {
		if width == lipgloss.Width(*t.header) {
			return *t.header
		}
	}

	header := theme.Header("FOLDERS", width, t.Focused()) + "\n"
	t.header = &header
	return header
}

func (t *DirectoryTree) render() string {
	var tree strings.Builder

	for i, dir := range t.dirsListFlat {
		// Removes invalid directory items
		// index and parent 0 shouldn't be possible but
		//sometime occurs after certain user actions
		if dir.index == 0 && dir.parent == 0 {
			t.dirsListFlat = slices.Delete(t.dirsListFlat, i, i+1)
			continue
		}

		if t.lastIndex == dir.Index() {
			dir.isLastChild = true
		}

		dir.selected = (t.selectedIndex == i)

		if *app.Debug {
			// prepend tree item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			tree.WriteString(style.Render(fmt.Sprintf("%02d", dir.index)))
			tree.WriteString(" ")
		}

		if t.Size.Width > 0 {
			dir.width = t.Size.Width - 2
		}

		// Prepare all visual row elements and get the correct width.
		// This has to be called before any actuall output
		dir.prepareRow()

		// Set the correct input width so that in case the folder name is too
		// long we're not breaking to the next line
		t.input.SetWidth(dir.width - 1)
		dir.input = &t.input

		// get text input if there's an edit index which likely means
		// we're renaming or creating
		isInput := t.editIndex != nil && i == *t.editIndex

		tree.WriteString(dir.String(isInput))
		tree.WriteByte('\n')
	}

	return tree.String()
}

// getChildren reads a directory and returns a slice of a directory Dir
func (t *DirectoryTree) getChildren(path string, level int) []TreeItem {
	var dirs []TreeItem
	childDir, _ := directories.List(path)

	for _, dir := range childDir {
		dirItem := t.createDirectoryItem(dir, level)

		if dir.IsExpanded {
			t.expandedDirs[dir.Path] = dir.IsExpanded
		}

		dirItem.expanded = dir.IsExpanded
		dirs = append(dirs, dirItem)
	}

	return dirs
}

// createDirectoryItem creates a directory item
func (m *DirectoryTree) createDirectoryItem(
	dir directories.Directory,
	level int,
) TreeItem {
	style := DirTreeStyle()

	dirItem := TreeItem{
		Item: Item{
			index:     0,
			name:      dir.Name(),
			path:      dir.Path,
			styles:    style,
			nerdFonts: m.conf.NerdFonts(),
		},
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

	return TreeItem{
		Item: Item{
			index: len(t.dirsListFlat),
			name:  tempFolderName,
			path:  tempFolderPath,
		},
		expanded: false,
		children: nil,
		parent:   selectedDir.index,
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

	if t.EditState == EditStates.Create {
		resetIndex = true
	}

	if sel := t.SelectedDir(); sel != nil {
		selectAfter := t.selectedIndex
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
		if t.selectedIndex != 0 {
			t.selectedIndex = 0
			t.Refresh(false, false)
		}
	}

	return statusMsg
}

// RefreshBranch refreshes a tree branch by its branch index
//
// Use `selectAfter` to change the selected tree item after the branch got refreshed.
// If `selectAfter` is -1 the branch's parent is selected
func (t *DirectoryTree) RefreshBranch(index int, selectAfter int) {
	t.selectedIndex = index

	if sel := t.SelectedDir(); sel != nil {
		if dir := findDirInTree(t.items, sel.path); dir != nil {
			dir.children = t.getChildren(dir.path, dir.level+1)
		}
	}

	if selectAfter == -1 {
		selectAfter = index
	}

	t.selectedIndex = selectAfter
	t.build()
}

// SelectedDir returns the currently selected directory in the directory tree
func (t *DirectoryTree) SelectedDir() *TreeItem {
	return t.SelectedItem(t.dirsListFlat)
}

// refreshFlatList reorganises the one-dimensional directory tree
func (t *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	t.dirsListFlat = t.flatten(t.items, 0, -1, &nextIndex)
}

// flatten converts a slice of Dir and its sub slices into
// a one dimensional slice that we use to render the directory tree
func (t *DirectoryTree) flatten(
	dirs []TreeItem,
	level int,
	parent int,
	nextIndex *int,
) []TreeItem {
	var result []TreeItem
	for i, dir := range dirs {
		dir.index = *nextIndex
		dir.parent = parent
		dir.level = level

		*nextIndex++

		if _, contains := t.expandedDirs[dir.path]; contains {
			dir.expanded = true
		}

		result = append(result, dir)

		if !dir.expanded {
			continue
		}

		children := dirs[i].children
		result = append(
			result,
			t.flatten(children, level+1, dir.index, nextIndex)...,
		)
	}
	return result
}

// lastChildOfSelection returns the corresponding last child
// of the selected directory.
func (t *DirectoryTree) lastChildOfSelection() *TreeItem {
	selectedDir := t.SelectedDir()
	lastChild := t.getLastChild(selectedDir.index)

	if lastChild.expanded && len(lastChild.children) > 0 {
		lastChild = t.getLastChild(lastChild.index)
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
	if dir.index == 0 {
		return &lastChild
	}

	if len(dir.children) > 0 {
		lastChild = dir.children[len(dir.children)-1]
		if lastChild.name == "" {
			lastChild = dir
		}
		for _, dir := range t.dirsListFlat {
			if lastChild.name == dir.name {
				lastChild = dir
			}
		}
	}

	return &lastChild
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// of the directories.
// To make it persistent write it to the file system
func (m *DirectoryTree) insertDirAfter(afterIndex int, directory TreeItem) {
	for i, dir := range m.dirsListFlat {
		if dir.index == afterIndex {
			m.dirsListFlat = append(
				m.dirsListFlat[:i+1],
				append([]TreeItem{directory}, m.dirsListFlat[i+1:]...)...,
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
func findDirInTree(directories []TreeItem, path string) *TreeItem {
	for i := range directories {
		if directories[i].path == path {
			return &directories[i]
		}

		if directories[i].expanded {
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
	dirPath, err := t.conf.MetaValue("", config.CurrentDirectory)
	if err == nil && dirPath != "" {
		for i := range t.dirsListFlat {
			if t.dirsListFlat[i].path != dirPath {
				continue
			}

			index := t.dirsListFlat[i].index
			t.selectedIndex = index
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
	if t.selectedIndex >= len(t.dirsListFlat) {
		return statusMsg
	}

	items := t.items
	path := t.SelectedDir().path

	if dir := findDirInTree(items, path); dir != nil {
		if dir.expanded {
			// remove expanded state from cached map
			delete(t.expandedDirs, dir.path)
			dir.expanded = false

			// rebuild directory tree
			t.build()
		}

		// save state to meta config file
		t.conf.SetMetaValue(dir.path, config.Expanded, "false")
	}

	return statusMsg
}

// Expand expands the currently selected directory
func (t *DirectoryTree) Expand() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}
	if t.selectedIndex >= len(t.dirsListFlat) ||
		len(t.SelectedDir().children) == 0 {

		return statusMsg
	}

	items := t.items
	path := t.SelectedDir().path

	if dir := findDirInTree(items, path); dir != nil {
		if !dir.expanded {
			// add expanded state to cached map
			t.expandedDirs[dir.path] = true
			dir.children = t.getChildren(dir.path, dir.level+1)
			dir.expanded = true

			// rebuild directory tree
			t.build()

			// save state to meta config file
			t.conf.SetMetaValue(dir.path, config.Expanded, "true")
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
	statusBar *StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if t.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		t.EditState = EditStates.Create
		// get a fresh version of the tree to work with
		t.refreshFlatList()
		// expand the selected directory for a live preview
		// of the new directory
		t.Expand()

		vrtDir := t.createVirtualDir()
		selDir := t.SelectedDir()

		// if the selected directory has no children yet
		// we append and empty Dir so that we get a correct result
		if selDir.index != 0 && len(selDir.children) == 0 {
			selDir.children = append(selDir.children, TreeItem{})
		}

		lastChild := t.lastChildOfSelection()
		t.insertDirAfter(lastChild.index, vrtDir)

		// update the selected index to the virtual directory
		// so that we input the name at the correct position
		t.selectedIndex = lastChild.index + 1

		if t.editIndex == nil {
			t.editIndex = &t.selectedIndex
			t.input.SetValue(vrtDir.name)
			t.input.CursorEnd()
		}
	}

	return statusMsg
}

// ConfirmRemove Confirms returns a status bar prompt
// to confirm or cancel the removal of a directory
func (t *DirectoryTree) ConfirmRemove() message.StatusBarMsg {
	selectedDir := t.SelectedDir()

	// prevent deleting root directory
	if selectedDir == nil || t.SelectedIndex() == 0 {
		return message.StatusBarMsg{Type: message.None}
	}

	t.EditState = EditStates.Delete

	rootDir, _ := app.NotesRootDir()
	path := strings.ReplaceAll(selectedDir.path, rootDir, ".")
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
	index := t.selectedIndex
	resultMsg := ""
	msgType := message.Success

	if err := directories.Delete(dir.path, true); err == nil {
		// delte the directory from the flat list
		t.dirsListFlat = slices.Delete(
			t.dirsListFlat,
			index,
			index+1,
		)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	t.RefreshBranch(dir.parent, index)

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
	if t.editIndex != nil {
		selDir := t.SelectedDir()
		oldPath := selDir.path

		// build the new path with the new name
		newPath := filepath.Join(
			filepath.Dir(oldPath),
			t.input.Value(),
		)

		switch t.EditState {
		case EditStates.Rename:
			if err := directories.Rename(oldPath, newPath); err == nil {
				t.Refresh(false, false)
				t.selectedIndex = t.indexByPath(
					newPath,
					&t.dirsListFlat,
				)
			}

		case EditStates.Create:
			if !t.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

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
	iconDir := theme.Icon(theme.IconDirClosed, t.conf.NerdFonts())
	iconNotes := theme.Icon(theme.IconNote, t.conf.NerdFonts())

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
	msg.WriteString(nbrFolders.String())

	return message.StatusBarMsg{
		Column:  sbc.FileInfo,
		Content: msg.String(),
	}
}

func (t *DirectoryTree) TogglePinned() message.StatusBarMsg {
	return message.StatusBarMsg{}
}
