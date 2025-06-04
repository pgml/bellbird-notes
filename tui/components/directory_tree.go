package components

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/directories"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/messages"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DirectoryTree represents the bubbletea model.
type DirectoryTree struct {
	List[TreeItem]

	// A flattened representation to make vertical navigation easier
	dirsListFlat []TreeItem
	// Stores currently expanded directories
	expandedDirs map[string]bool
}

// TreeItem represents a single directory tree row
type TreeItem struct {
	Item

	// The parent index of the directory.
	// Used to make expanding and collapsing a directory possible
	// using DirectoryTree.dirsListFlat
	parent   int
	children []TreeItem
	// Indicates whether a directory is expanded
	expanded bool
	// Indicates the depth of a directory
	// Used to determine the indentation of DirectoryTree.dirsFlatList
	level int
	// the amount of notes a directory contains
	NbrNotes int
	// the amount of sub directories a directory has
	NbrFolders int
}

// Index returns the index of a Dir-Item
func (d TreeItem) Index() int { return d.index }

// GetPath returns the path of a Dir-Item
func (d TreeItem) Path() string { return d.path }

// GetName returns the name of a Dir-Item
func (d TreeItem) Name() string { return d.name }

// GetIndent returns the path of a Dir-Item
func (d TreeItem) Indent(indentLines bool) string {
	if indentLines {
		return "│ "
	} else {
		return "  "
	}
}

// The string representation of a Dir
func (d *TreeItem) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	e := d.styles.enumerator.Render

	indent := strings.Repeat(
		d.Indent(false), // @todo make this a config option
		d.level,
	)
	name := utils.TruncateText(d.Name(), 22)

	// nerdfonts required
	toggle := map[string]string{
		"open":  "",
		"close": "󰉋",
	}

	if *app.NoNerdFonts {
		toggle = map[string]string{
			"open":  "▼",
			"close": "▶",
		}
	}

	baseStyle := lipgloss.NewStyle().Width(28)
	if d.selected {
		baseStyle = baseStyle.
			Background(theme.ColourBgSelected).
			Bold(true)
	}

	row := e(indent)
	if d.expanded {
		row += t(toggle["open"])
	} else {
		row += t(toggle["close"])
	}

	return baseStyle.Render(row + n(name))
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (t *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (t *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	termWidth, termHeight := theme.GetTerminalSize()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.editIndex != nil && !t.editor.Focused() {
			t.editor.Focus()
			return t, nil
		}

		if t.editor.Focused() {
			t.editor.Focus()
			t.editor, cmd = t.editor.Update(msg)
			return t, cmd
		}

	case tea.WindowSizeMsg:
		if !t.ready {
			t.viewport = viewport.New(termWidth, termHeight)
			t.viewport.SetContent(t.render())
			t.viewport.KeyMap = viewport.KeyMap{}
			t.lastVisibleLine = t.viewport.
				VisibleLineCount() - reservedLines
			t.ready = true
		} else {
			t.viewport.Width = termWidth
			t.viewport.Height = termHeight
		}
	}

	t.viewport, cmd = t.viewport.Update(msg)

	return t, cmd
}

func (t *DirectoryTree) View() string {
	if !t.ready {
		return "\n  Initializing..."
	}

	t.viewport.SetContent(t.render())
	t.UpdateViewportInfo()

	t.viewport.Style = theme.BaseColumnLayout(
		t.Size,
		t.Focused,
	)

	return t.viewport.View()
}

// NewDirectoryTree creates a new model with default settings.
func NewDirectoryTree() *DirectoryTree {
	ti := textinput.New()
	ti.Prompt = theme.IconInput + " "
	ti.CharLimit = 100

	tree := &DirectoryTree{
		List: List[TreeItem]{
			selectedIndex: 0,
			editIndex:     nil,
			EditState:     EditNone,
			editor:        ti,
			items:         make([]TreeItem, 0),
		},
		expandedDirs: make(map[string]bool),
	}

	conf := config.New()
	notesDir := conf.Value(
		config.General,
		config.UserNotesDirectory,
	)

	// append root directory
	tree.items = append(tree.items, TreeItem{
		Item: Item{
			index:  0,
			name:   app.Name(),
			path:   notesDir,
			styles: DirTreeStyle(),
		},
		expanded: true,
		level:    0,
		parent:   -1,
		children: tree.getChildren(notesDir, 0),
	})

	tree.build()
	return tree
}

// build prepares t.dirsListFlat for rendering
// checking directory states etc.
func (t *DirectoryTree) build() {
	t.refreshFlatList()

	//for _, dir := range t.dirsListFlat {
	//	dir.expanded = t.isExpanded(dir.Path)
	//}

	t.length = len(t.dirsListFlat)
	t.lastIndex = t.dirsListFlat[len(t.dirsListFlat)-1].index
}

func (t *DirectoryTree) render() string {
	var tree string

	for i, dir := range t.dirsListFlat {
		// Removes invalid directory items
		// index and parent 0 shouldn't be possible but
		//sometime occurs after certain user actions
		if dir.index == 0 && dir.parent == 0 {
			t.dirsListFlat = slices.Delete(t.dirsListFlat, i, i+1)
			continue
		}

		indent := strings.Repeat(
			dir.Indent(false),
			dir.level,
		)

		dir.selected = (t.selectedIndex == i)

		if *app.Debug {
			// prepend tree item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			tree += style.Render(fmt.Sprintf("%02d", dir.index)) + " "
		}

		if t.editIndex != nil && i == *t.editIndex {
			// Show input field instead of text
			tree += indent + t.editor.View() + "\n"
		} else {
			tree += fmt.Sprintf(
				"%-*s \n",
				t.viewport.Width,
				dir.String(),
			)
		}
	}

	return tree
}

// getChildren reads a directory and returns a slice of a directory Dir
func (t *DirectoryTree) getChildren(path string, level int) []TreeItem {
	var dirs []TreeItem
	childDir, _ := directories.List(path)

	for _, dir := range childDir {
		dirItem := t.createDirectoryItem(dir, level)
		dirItem.expanded = t.isExpanded(dir.Path)
		dirs = append(dirs, dirItem)
	}

	return dirs
}

// isExpanded returns whether a Dir-Item is expanded
func (t DirectoryTree) isExpanded(dirPath string) bool {
	_, contains := t.expandedDirs[dirPath]
	return contains
}

func (m *DirectoryTree) createDirectoryItem(
	dir directories.Directory,
	level int,
) TreeItem {
	style := DirTreeStyle()

	dirItem := TreeItem{
		Item: Item{
			index:  0,
			name:   dir.Name,
			path:   dir.Path,
			styles: style,
		},
		expanded:   dir.IsExpanded,
		parent:     0,
		children:   m.getChildren(dir.Path, level+1),
		NbrFolders: dir.NbrFolders,
		level:      level,
		NbrNotes:   0,
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
func (t *DirectoryTree) Refresh(resetIndex bool) messages.StatusBarMsg {
	statusMsg := messages.StatusBarMsg{}

	if t.EditState == EditCreate {
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
			t.Refresh(false)
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
	path := t.SelectedDir().path

	if dir := findDirInTree(t.items, path); dir != nil {
		dir.children = t.getChildren(dir.path, dir.level+1)
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
		//statusMsg = messages.StatusBarMsg{
		//	Content: "Directory already exists, please choose another name.",
		//	Type:    messages.Error,
		//	Sender:  messages.SenderDirTree,
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

///
/// keyboard shortcut commands
///

// Collapses the currently selected directory
func (t *DirectoryTree) Collapse() messages.StatusBarMsg {
	statusMsg := messages.StatusBarMsg{}
	if t.selectedIndex >= len(t.dirsListFlat) || !t.Focused {
		return statusMsg
	}

	items := t.items
	path := t.SelectedDir().path

	if dir := findDirInTree(items, path); dir != nil {
		if dir.expanded {
			delete(t.expandedDirs, dir.path)
			dir.expanded = false
			t.build()
		}
	}

	return statusMsg
}

// Expands the currently selected directory
func (t *DirectoryTree) Expand() messages.StatusBarMsg {
	statusMsg := messages.StatusBarMsg{}
	if t.selectedIndex >= len(t.dirsListFlat) || !t.Focused {
		return statusMsg
	}

	items := t.items
	path := t.SelectedDir().path

	if dir := findDirInTree(items, path); dir != nil {
		if !dir.expanded {
			t.expandedDirs[dir.path] = true
			dir.children = t.getChildren(dir.path, dir.level+1)
			dir.expanded = true
			t.build()
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
) messages.StatusBarMsg {
	statusMsg := messages.StatusBarMsg{}

	if t.Focused {
		mi.Current = mode.Insert
		statusBar.Focused = false

		t.EditState = EditCreate
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
			t.editor.SetValue(vrtDir.name)
			t.editor.CursorEnd()
		}
	}

	return statusMsg
}

func (t *DirectoryTree) ConfirmRemove() messages.StatusBarMsg {
	selectedDir := t.SelectedDir()
	msgType := messages.PromptError
	t.EditState = EditDelete

	resultMsg := fmt.Sprintf(
		messages.RemovePromptContent,
		selectedDir.path,
	)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  messages.SenderDirTree,
		Column:  1,
	}
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (t *DirectoryTree) Remove() messages.StatusBarMsg {
	dir := t.SelectedDir()
	index := t.selectedIndex
	resultMsg := ""
	msgType := messages.Success

	if err := directories.Delete(dir.path, false); err == nil {
		t.dirsListFlat = slices.Delete(
			t.dirsListFlat,
			index,
			index+1,
		)
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	t.RefreshBranch(dir.parent, index)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  1,
	}
}

// Confirms a user action
func (t *DirectoryTree) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if t.editIndex != nil {
		selDir := t.SelectedDir()
		oldPath := selDir.path

		newPath := filepath.Join(
			filepath.Dir(oldPath),
			t.editor.Value(),
		)

		switch t.EditState {
		case EditRename:
			if err := directories.Rename(oldPath, newPath); err == nil {
				t.Refresh(false)
				t.selectedIndex = t.indexByPath(
					newPath,
					&t.dirsListFlat,
				)
			}

		case EditCreate:
			if !t.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

		t.CancelAction(func() {
			t.Refresh(false)
		})

		return messages.StatusBarMsg{Content: "yep"}
	}

	return messages.StatusBarMsg{}
}
