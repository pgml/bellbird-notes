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
func (item TreeItem) Expanded() bool {
	return item.expanded
}

func (item *TreeItem) Expand() {
	item.expanded = true
}

func (item *TreeItem) Collapse() {
	item.expanded = false
}

func (item *TreeItem) SetExpanded(expand bool) {
	if expand {
		item.Expand()
	} else {
		item.Collapse()
	}
}

// setIndentation sets the visual indentation for the tree item based on its level
// and whether line markers (like │ or ╰) should be shown.
func (item *TreeItem) setIndentation(indentLines bool) {
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
	indent := strings.Repeat(indentStr, item.level)

	// width of indentation per level which needs to be subtracted later rom
	// general item width to prevent row breaks
	indentWidth := lipgloss.Width(indent)
	style := item.Styles.Indent.Width(indentWidth)

	if item.IsSelected {
		style = style.Background(theme.ColourBgSelected)
	}

	// store rendered indentation
	item.Indent = style.Render(indent)

	// subtract the indentation width from the item width
	item.SetWidth(item.Width() - indentWidth)
}

// setIcon sets the icon representing a folder state (open/closed).
// If the nerd fonts settings is disabled, nothing will be shown
func (item *TreeItem) setIcon() {
	folderClosed := ""
	folderOpen := ""

	if item.NerdFonts {
		folderClosed = theme.IconDirClosed.Nerd
		folderOpen = theme.IconDirOpen.Nerd
	}

	style := item.Styles.Icon.Width(item.Styles.IconWidth)
	if item.IsSelected {
		style = item.Styles.Selected.Width(item.Styles.IconWidth)
	}

	iconDir := folderClosed

	if item.IsPinned {
		iconDir = theme.Icon(theme.IconPin, item.NerdFonts)
		style = style.Foreground(theme.ColourBorderFocused)
	} else if item.Expanded() {
		iconDir = folderOpen
	}

	// store rendered icon
	item.Icon = style.Render(iconDir)

	// subtract the indentation width from the item width
	item.SetWidth(item.Width() - item.Styles.IconWidth)
}

// setToggleArrow sets the arrow icon used to expand/collapse tree items.
// Hides the arrow if the item has no children.
func (item *TreeItem) setToggleArrow() {
	iconArrow := theme.IconDirClosed.Alt
	if item.Expanded() {
		iconArrow = theme.IconDirOpen.Alt
	}

	if len(item.children) == 0 {
		iconArrow = ""
	}

	style := item.Styles.Toggle
	if item.IsSelected {
		style = item.Styles.Selected.Width(item.Styles.ToggleWidth)
	}

	// store rendered toggle arrow
	item.ToggleArrow = style.Render(iconArrow)

	// subtract the indentation width from the item width
	item.SetWidth(item.Width() - item.Styles.ToggleWidth)
}

// prepareRow initialises all visual elements (indent, icon, arrow) for the row.
func (item *TreeItem) prepareRow(showIndentLines bool) {
	item.setIndentation(showIndentLines)
	item.setIcon()
	item.setToggleArrow()
}

// String renders the complete visual representation of the tree item.
// If input is true render the input view to allow renaming/creating.
func (item TreeItem) String(input bool) string {
	if item.Width() <= 0 {
		return ""
	}

	baseStyle := item.Styles.Base.Width(item.Width())
	selectedStyle := item.Styles.Selected.Width(item.Width())
	if item.IsSelected {
		baseStyle = selectedStyle
	}

	name := utils.TruncateText(item.Name(), item.Width()-1)
	name = baseStyle.Render(name)

	// replace name with the stored input view if we are creating or
	// renaming
	if input {
		name = item.InputModel().View()
		item.Icon = ""
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		item.Indent, item.ToggleArrow, item.Icon, name,
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
func (tree *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (tree *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// focus the input field when renaming a list item
		if tree.EditIndex != nil && !tree.InputModel.Focused() {
			tree.InputModel.Focus()
			return tree, nil
		}

		if tree.InputModel.Focused() {
			tree.InputModel, cmd = tree.InputModel.Update(msg)
			return tree, cmd
		}

	case tea.WindowSizeMsg:
		tree.Size.Width = msg.Width
		tree.Size.Height = msg.Height

		if !tree.IsReady {
			tree.Viewport = viewport.New()
			tree.Viewport.SetContent(tree.viewportContent())
			tree.Viewport.KeyMap = viewport.KeyMap{}
			tree.LastVisibleLine = tree.Viewport.VisibleLineCount() - shared.ReservedLines
			tree.IsReady = true
		} else {
			tree.Viewport.SetWidth(tree.Size.Width)
			tree.Viewport.SetHeight(tree.Size.Height)
		}
	}

	tree.Viewport, cmd = tree.Viewport.Update(msg)

	return tree, cmd
}

func (tree *DirectoryTree) View() tea.View {
	var view tea.View
	view.SetContent(tree.Content())
	return view
}

// NewDirectoryTree creates a new model with default settings.
func New(title string, conf *config.Config) *DirectoryTree {
	var list shared.List[*TreeItem]
	list.MakeEmpty()
	list.Conf = conf

	tree := &DirectoryTree{
		List:         list,
		expandedDirs: make(map[string]bool),
	}

	tree.SetTitle(title)
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

	tree.SetTitle(title)
	tree.build()
	tree.SelectLastDir()
	return tree
}

func (tree DirectoryTree) NotesDir() string {
	// fetch notes directory
	notesDir, err := tree.Conf.NotesDir()
	if err != nil {
		debug.LogErr(err)
		return ""
	}

	return notesDir
}

// Input returns and textinput model tailored to the directory tree
func (tree *DirectoryTree) TreeInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = theme.Icon(theme.IconPen, tree.Conf.NerdFonts()) + " "
	ti.CharLimit = 100
	ti.VirtualCursor = true

	bgSelected := shared.DirTreeStyle().Selected
	ti.Styles.Focused = textinput.StyleState{
		Text:   bgSelected,
		Prompt: bgSelected,
	}

	return ti
}

func (tree *DirectoryTree) RefreshSize() {
	vp := tree.Viewport
	if vp.Width() != tree.Size.Width && vp.Height() != tree.Size.Height {
		tree.Viewport.SetWidth(tree.Size.Width)
		tree.Viewport.SetHeight(tree.Size.Height)
	}
}

func (tree *DirectoryTree) Content() string {
	if !tree.IsReady {
		return "\n  Initializing..."
	}

	if !tree.Visible() {
		return ""
	}

	tree.Viewport.SetContent(tree.viewportContent())
	tree.UpdateViewportInfo()
	tree.Viewport.EnsureVisible(tree.SelectedIndex, 0, 0)

	tree.Viewport.Style = tree.Theme().BaseColumnLayout(
		tree.Size,
		tree.Focused(),
	)

	var view strings.Builder
	view.WriteString(tree.BuildHeader(tree.Size.Width, false))
	view.WriteString(tree.Viewport.View())
	return view.String()
}

func (tree *DirectoryTree) viewportContent() string {
	var s strings.Builder

	for i, dir := range tree.dirsListFlat {
		// Removes invalid directory items
		// index and parent 0 shouldn't be possible but
		//sometime occurs after certain user actions
		if dir.Index() == 0 && dir.parent == 0 {
			tree.dirsListFlat = slices.Delete(tree.dirsListFlat, i, i+1)
			continue
		}

		if tree.LastIndex == dir.Index() {
			dir.isLastChild = true
		}

		dir.IsSelected = (tree.SelectedIndex == i)

		if *app.Debug {
			// prepend tree item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			s.WriteString(style.Render(fmt.Sprintf("%02d", dir.Index())))
			s.WriteString(" ")
		}

		if tree.Size.Width > 0 {
			dir.SetWidth(tree.Size.Width - 2)
		}

		// Prepare all visual row elements and get the correct width.
		// This has to be called before any actuall output
		dir.prepareRow(tree.indentLines)

		// Set the correct input width so that in case the folder name is too
		// long we're not breaking to the next line
		tree.InputModel.SetWidth(dir.Width() - 1)
		dir.SetInputModel(tree.InputModel)

		// get text input if there's an edit index which likely means
		// we're renaming or creating
		isInput := tree.EditIndex != nil && i == *tree.EditIndex

		s.WriteString(dir.String(isInput))
		s.WriteByte('\n')
	}

	return s.String()
}

// build prepares t.dirsListFlat for rendering
// checking directory states etc.
func (tree *DirectoryTree) build() {
	tree.refreshFlatList()
	tree.Length = len(tree.dirsListFlat)
	tree.LastIndex = tree.dirsListFlat[len(tree.dirsListFlat)-1].Index()
}

// getChildren reads a directory and returns a slice of a directory Dir
func (tree *DirectoryTree) getChildren(path string, level int) []*TreeItem {
	var dirs []*TreeItem
	childDir, _ := directories.List(path)

	// pinned stuff
	if !tree.PinnedItems.IsLoaded {
		// reset pinned and refetch pinned notes when we entered a new directory
		tree.PinnedItems.Items = make([]*TreeItem, 0, len(childDir))
		for _, dir := range childDir {
			if dir.IsPinned {
				item := tree.createDirectoryItem(dir, -1, true)
				tree.PinnedItems.Add(&item)
			}
		}
	}

	pinnedMap := make(map[string]struct{}, len(tree.PinnedItems.Items))
	for _, n := range tree.PinnedItems.Items {
		pinnedMap[n.Path()] = struct{}{}
	}

	var (
		pinnedItems   []*TreeItem
		unpinnedItems []*TreeItem
	)

	for _, dir := range childDir {
		_, isPinned := pinnedMap[dir.Path]
		dirItem := tree.createDirectoryItem(dir, level, isPinned)

		if dir.IsExpanded {
			tree.expandedDirs[dir.Path] = dir.IsExpanded
		}

		dirItem.SetExpanded(dir.IsExpanded)

		if isPinned {
			pinnedItems = append(pinnedItems, &dirItem)
		} else {
			unpinnedItems = append(unpinnedItems, &dirItem)
		}
	}

	dirs = append(pinnedItems, unpinnedItems...)
	tree.PinnedItems.IsLoaded = true

	return dirs
}

// createDirectoryItem creates a directory item
func (tree *DirectoryTree) createDirectoryItem(
	dir directories.Directory,
	level int,
	isPinned bool,
) TreeItem {
	style := shared.DirTreeStyle()

	var item shared.Item
	item.SetIndex(0)
	item.SetName(dir.Name())
	item.SetPath(dir.Path)
	item.NerdFonts = tree.Conf.NerdFonts()
	item.IsPinned = isPinned
	item.Styles = style

	dirItem := TreeItem{
		Item:       item,
		expanded:   dir.IsExpanded,
		parent:     0,
		children:   tree.getChildren(dir.Path, level+1),
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
func (tree *DirectoryTree) createVirtualDir() TreeItem {
	selectedDir := tree.SelectedDir()
	tempFolderName := "New Folder"
	tempFolderPath := filepath.Join(
		selectedDir.Path(),
		tempFolderName,
	)
	indent := selectedDir.level + 1

	var item shared.Item
	item.SetIndex(len(tree.dirsListFlat))
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
func (tree *DirectoryTree) Refresh(
	resetIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if tree.EditState == shared.EditStates.Create {
		resetIndex = true
	}

	if sel := tree.SelectedDir(); sel != nil {
		selectAfter := tree.SelectedIndex
		if resetIndex {
			selectAfter = -1
		}

		tree.RefreshBranch(
			tree.SelectedDir().parent,
			selectAfter,
		)
	} else {
		// If for some reason there is no selected directory and
		// refresh is triggered just select root
		if tree.SelectedIndex != 0 {
			tree.SelectedIndex = 0
			tree.Refresh(false, false)
		}
	}

	tree.checkVisibility()
	tree.checkIndentLines()

	return statusMsg
}

// RefreshBranch refreshes a tree branch by its branch index
//
// Use `selectAfter` to change the selected tree item after the branch got refreshed.
// If `selectAfter` is -1 the branch's parent is selected
func (tree *DirectoryTree) RefreshBranch(index int, selectAfter int) {
	tree.SelectedIndex = index

	if sel := tree.SelectedDir(); sel != nil {
		if dir := findDirInTree(tree.Items, sel.Path()); dir != nil {
			dir.children = tree.getChildren(dir.Path(), dir.level+1)
		}
	}

	if selectAfter == -1 {
		selectAfter = index
	}

	tree.SelectedIndex = selectAfter
	tree.build()
}

// SelectedDir returns the currently selected directory in the directory tree
func (tree *DirectoryTree) SelectedDir() *TreeItem {
	return tree.SelectedItem(tree.dirsListFlat)
}

// refreshFlatList reorganises the one-dimensional directory tree
func (tree *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	tree.dirsListFlat = tree.flatten(tree.Items, 0, -1, &nextIndex)
}

// flatten converts a slice of Dir and its sub slices into
// a one dimensional slice that we use to render the directory tree
func (tree *DirectoryTree) flatten(
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

		if _, contains := tree.expandedDirs[dir.Path()]; contains {
			dir.Expand()
		}

		result = append(result, dir)

		if !dir.Expanded() {
			continue
		}

		children := dirs[i].children
		result = append(
			result,
			tree.flatten(children, level+1, dir.Index(), nextIndex)...,
		)
	}
	return result
}

// lastChildOfSelection returns the corresponding last child
// of the selected directory.
func (tree *DirectoryTree) lastChildOfSelection() *TreeItem {
	selectedDir := tree.SelectedDir()
	lastChild := tree.getLastChild(selectedDir.Index())

	if lastChild.Expanded() && len(lastChild.children) > 0 {
		lastChild = tree.getLastChild(lastChild.Index())
	}
	return lastChild
}

// getLastChild returns the last child of the item with the given index
//
// if `createEmpty` is set to true, we attempt to create an empty
// Dir{}
func (tree *DirectoryTree) getLastChild(index int) *TreeItem {
	lastChild := tree.dirsListFlat[len(tree.dirsListFlat)-1]
	dir := tree.dirsListFlat[index]

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
		for _, dir := range tree.dirsListFlat {
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
func (tree *DirectoryTree) insertDirAfter(afterIndex int, directory TreeItem) {
	for i, dir := range tree.dirsListFlat {
		if dir.Index() == afterIndex {
			tree.dirsListFlat = append(
				tree.dirsListFlat[:i+1],
				append([]*TreeItem{&directory}, tree.dirsListFlat[i+1:]...)...,
			)
			break
		}
	}
}

func (tree *DirectoryTree) dirExists(dirPath string) bool {
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
func (tree *DirectoryTree) SelectLastDir() string {
	dirPath, err := tree.Conf.MetaValue("", config.LastDirectory)
	if err == nil && dirPath != "" {
		for i := range tree.dirsListFlat {
			if tree.dirsListFlat[i].Path() != dirPath {
				continue
			}

			index := tree.dirsListFlat[i].Index()
			tree.SelectedIndex = index
			return dirPath
		}
	}
	return ""
}

///
/// keyboard shortcut commands
///

// Collapse collapses the currently selected directory
func (tree *DirectoryTree) Collapse() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}
	if tree.SelectedIndex >= len(tree.dirsListFlat) {
		return statusMsg
	}

	items := tree.Items
	path := tree.SelectedDir().Path()

	if dir := findDirInTree(items, path); dir != nil {
		if dir.Expanded() {
			// remove expanded state from cached map
			delete(tree.expandedDirs, dir.Path())
			dir.Collapse()

			// rebuild directory tree
			tree.build()
		}

		// save state to meta config file
		tree.Conf.SetMetaValue(dir.Path(), config.Expanded, "false")
	}

	return statusMsg
}

// Expand expands the currently selected directory
func (tree *DirectoryTree) Expand() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}
	if tree.SelectedIndex >= len(tree.dirsListFlat) ||
		len(tree.SelectedDir().children) == 0 {

		return statusMsg
	}

	items := tree.Items
	path := tree.SelectedDir().Path()

	if dir := findDirInTree(items, path); dir != nil {
		if !dir.Expanded() {
			// add expanded state to cached map
			tree.expandedDirs[dir.Path()] = true
			dir.children = tree.getChildren(dir.Path(), dir.level+1)
			dir.Expand()

			// rebuild directory tree
			tree.build()

			// save state to meta config file
			tree.Conf.SetMetaValue(dir.Path(), config.Expanded, "true")
		}
	}

	return statusMsg
}

// Create creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (tree *DirectoryTree) Create(
	mi *mode.ModeInstance,
	statusBar *statusbar.StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if tree.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		tree.EditState = shared.EditStates.Create
		// get a fresh version of the tree to work with
		tree.refreshFlatList()
		// expand the selected directory for a live preview
		// of the new directory
		tree.Expand()

		vrtDir := tree.createVirtualDir()
		selDir := tree.SelectedDir()

		// if the selected directory has no children yet
		// we append and empty Dir so that we get a correct result
		if selDir.Index() != 0 && len(selDir.children) == 0 {
			selDir.children = append(selDir.children, &TreeItem{})
		}

		lastChild := tree.lastChildOfSelection()
		tree.insertDirAfter(lastChild.Index(), vrtDir)

		// update the selected index to the virtual directory
		// so that we input the name at the correct position
		tree.SelectedIndex = lastChild.Index() + 1

		if tree.EditIndex == nil {
			index := tree.SelectedIndex
			tree.EditIndex = &index
			tree.InputModel.SetValue(vrtDir.Name())
			tree.InputModel.CursorEnd()
		}
	}

	return statusMsg
}

// ConfirmRemove returns a status bar prompt
// to confirm or cancel the removal of a directory
func (tree *DirectoryTree) ConfirmRemove() message.StatusBarMsg {
	selectedDir := tree.SelectedDir()

	// prevent deleting root directory
	if selectedDir == nil || tree.SelectedIndex == 0 {
		return message.StatusBarMsg{Type: message.None}
	}

	tree.EditState = shared.EditStates.Delete

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
func (tree *DirectoryTree) Remove() message.StatusBarMsg {
	dir := tree.SelectedDir()
	index := tree.SelectedIndex
	parent := dir.parent
	resultMsg := ""
	msgType := message.Success

	if err := directories.Delete(dir.Path(), true); err == nil {
		// delete the directory from the flat list
		tree.dirsListFlat = slices.Delete(
			tree.dirsListFlat,
			index,
			index+1,
		)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	tree.RefreshBranch(parent, index)

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  sbc.General,
	}
}

// ConfirmAction confirms a user action
func (tree *DirectoryTree) ConfirmAction() message.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if tree.EditIndex != nil {
		selDir := tree.SelectedDir()
		oldPath := selDir.Path()

		// build the new path with the new name
		newPath := filepath.Join(
			filepath.Dir(oldPath),
			tree.InputModel.Value(),
		)

		switch tree.EditState {
		case shared.EditStates.Rename:
			if err := directories.Rename(oldPath, newPath); err != nil {
				debug.LogErr(err)
			}

		case shared.EditStates.Create:
			if !tree.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

		// selected the newly renamed or created directory
		tree.Refresh(false, false)
		tree.SelectedIndex = tree.IndexByPath(newPath, &tree.dirsListFlat)

		tree.CancelAction(func() {
			tree.Refresh(false, false)
		})

		return message.StatusBarMsg{Content: "yep"}
	}

	return message.StatusBarMsg{}
}

// ContentInfo returns the info about the currently selected directory
// in the status bar
func (tree *DirectoryTree) ContentInfo() message.StatusBarMsg {
	sel := tree.SelectedDir()
	iconDir := theme.Icon(theme.IconDirClosed, tree.Conf.NerdFonts())
	iconNotes := theme.Icon(theme.IconNote, tree.Conf.NerdFonts())

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

func (tree *DirectoryTree) TogglePinnedItems() message.StatusBarMsg {
	dir := tree.SelectedDir()

	tree.TogglePinned(dir)
	tree.Refresh(false, false)

	// get the new index and select the newly pinned or unpinned note
	// since the pinned notes are always at the top and the notes order
	// is changed
	for i, it := range tree.dirsListFlat {
		if it.Path() == dir.Path() {
			tree.SelectedIndex = i
		}
	}

	return message.StatusBarMsg{}
}

// YankSelection clears the yankedItems list and adds the currently selected item
// from the NotesList to it. This simulates copying an item for later pasting.
func (tree *DirectoryTree) YankSelection(markCut bool) {
	sel := tree.SelectedDir()
	sel.SetIsCut(markCut)

	tree.YankedItems = []*TreeItem{}
	tree.YankedItems = append(tree.YankedItems, sel)
}

// PasteSelection duplicates all yanked notes into the specified directory path.
// It handles name conflicts by appending " Copy" to the note name until a unique
// path is found. Returns an error if any note cannot be created.
func (tree *DirectoryTree) PasteSelectedItems() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	sel := tree.SelectedDir()

	for _, dir := range tree.YankedItems {
		tree.PasteSelection(dir, sel.Path(), func(newPath string) {
			err := directories.Copy(dir.Path(), newPath)

			if err != nil {
				debug.LogErr(err)
				return
			}

			//t.Refresh(false, false)
			tree.RefreshBranch(sel.Index(), -1)
			tree.Expand()

			// select the currently pasted item
			if dir, ok := tree.ItemsContain(newPath); ok {
				tree.SelectedIndex = dir.Index()
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
func (tree *DirectoryTree) ItemsContain(path string) (*TreeItem, bool) {
	for _, item := range tree.dirsListFlat {
		if item.Path() == path {
			return item, true
		}
	}

	return nil, false
}

func (tree *DirectoryTree) ToggleIndentLines() message.StatusBarMsg {
	tree.indentLines = !tree.indentLines

	tree.Conf.SetValue(
		config.Folders,
		config.IndentLines,
		strconv.FormatBool(tree.indentLines),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (tree *DirectoryTree) Toggle() message.StatusBarMsg {
	tree.ToggleVisibility()

	tree.Conf.SetValue(
		config.Folders,
		config.Visible,
		strconv.FormatBool(tree.Visible()),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (tree *DirectoryTree) checkIndentLines() {
	lines, err := tree.Conf.Value(config.Folders, config.IndentLines)

	if err != nil {
		debug.LogErr(err)
	}

	tree.indentLines = lines.GetBool()
}

func (tree *DirectoryTree) checkVisibility() {
	vis, err := tree.Conf.Value(config.Folders, config.Visible)

	if err != nil {
		debug.LogErr(err)
	}

	tree.SetVisibility(vis.GetBool())
}
