package components

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"
	"bellbird-notes/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DirectoryTree represents the bubbletea model.
type DirectoryTree struct {
	List[Dir]

	dirsListFlat []Dir           // A flattened representation to make vertical navigation easier
	expandedDirs map[string]bool // Stores currently expanded directories
}

//type statusMsg string

// Dir represents a single directory tree row
type Dir struct {
	Item

	// The parent index of the directory.
	// Used to make expanding and collapsing a directory possible
	// using DirectoryTree.dirsListFlat
	parent   int
	children []Dir

	expanded bool // Indicates whether a directory is expanded

	// Indicates the depth of a directory
	// Used to determine the indentation of DirectoryTree.dirsFlatList
	level      int
	NbrNotes   int // the amount of notes a directory contains
	NbrFolders int // the amount of sub directories a directory has
}

func (d Dir) GetIndex() int   { return d.index }
func (d Dir) GetPath() string { return d.Path }

// The string representation of a Dir
func (d Dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	e := d.styles.enumerator.Render
	//indent := strings.Repeat("│ ", d.level)
	indent := strings.Repeat("  ", d.level)
	name := utils.TruncateText(d.Name, 22)

	toggle := map[string]string{"open": "", "close": "󰉋"}
	if noNerdFonts {
		toggle = map[string]string{"open": "▼", "close": "▶"}
	}

	baseStyle := lipgloss.NewStyle().Width(30)

	if d.selected {
		baseStyle = baseStyle.
			Background(lipgloss.Color("#424B5D")).
			Bold(true)
	}

	row := e(indent)

	if d.expanded {
		row += t(toggle["open"])
	} else {
		row += t(toggle["close"])
	}
	return baseStyle.Render(row + n(name))
	//return baseStyle.Render(row + n(name+" "+strconv.Itoa(d.index)))
}

func (d Dir) GetName() string {
	return d.Name
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (t *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (t *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd tea.Cmd
		//cmds []tea.Cmd
	)
	termWidth, termHeight := theme.GetTerminalSize()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.editingIndex != nil && !t.editor.Focused() {
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
			t.viewport = viewport.New(termWidth, termHeight-1)
			t.viewport.SetContent(t.render())
			t.viewport.KeyMap = viewport.KeyMap{}
			t.lastVisibleLine = t.viewport.VisibleLineCount() - 3
			t.ready = true
		} else {
			t.viewport.Width = termWidth
			t.viewport.Height = termHeight - 1
		}
	}

	t.viewport, cmd = t.viewport.Update(msg)
	//cmds = append(cmds, cmd)

	return t, cmd
}

func (t *DirectoryTree) View() string {
	if !t.ready {
		return "\n  Initializing..."
	}

	t.viewport.SetContent(t.render())
	t.UpdateViewportInfo()
	t.viewport.Style = theme.BaseColumnLayout(t.Size, t.Focused)
	return t.viewport.View()
}

// New creates a new model with default settings.
func NewDirectoryTree() *DirectoryTree {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	tree := &DirectoryTree{
		List: List[Dir]{
			selectedIndex: 0,
			editingIndex:  nil,
			editingState:  EditNone,
			editor:        ti,
			items:         make([]Dir, 0),
		},
		expandedDirs: make(map[string]bool),
	}
	conf := config.New()
	notesDir := conf.Value(config.General, config.UserNotesDirectory)

	// append root directory
	tree.items = append(tree.items, Dir{
		Item: Item{
			index:  0,
			Name:   "Bellbird Notes",
			Path:   notesDir,
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
	//dirTree := list.New().
	//	Enumerator(func(items list.Items, index int) string { return "" })

	t.refreshFlatList()
	for _, dir := range t.dirsListFlat {
		dir.expanded = t.isExpanded(dir.Path)
		//dirTree.Item(dir)
	}

	t.length = len(t.dirsListFlat)
	t.lastIndex = t.dirsListFlat[len(t.dirsListFlat)-1].index
	//t.tree = dirTree
}

func (t DirectoryTree) render() string {
	var tree string

	for i, dir := range t.dirsListFlat {
		// Removes invalid directory items
		// index and parent 0 shouldn't be possible but sometime occurs after
		// certain user actions
		if dir.index == 0 && dir.parent == 0 {
			t.dirsListFlat = slices.Delete(t.dirsListFlat, i, i+1)
			continue
		}

		indent := strings.Repeat("  ", dir.level)
		dir.selected = (t.selectedIndex == i)

		//style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
		//tree += style.Render(fmt.Sprintf("%02d", dir.index)) + " "
		if t.editingIndex != nil && i == *t.editingIndex {
			// Show input field instead of text
			tree += indent + t.editor.View() + "\n"
		} else {
			tree += fmt.Sprintf("%-*s \n", t.viewport.Width, dir.String())
		}
	}

	return tree
}

// getChildren reads a directory and returns a slice of a directory Dir
func (t *DirectoryTree) getChildren(path string, level int) []Dir {
	var dirs []Dir
	childDir, _ := directories.List(path)
	for _, dir := range childDir {
		dirItem := t.createDirectoryItem(dir, level)
		dirItem.expanded = t.isExpanded(dir.Path)
		dirs = append(dirs, dirItem)
	}
	return dirs
}

func (t DirectoryTree) isExpanded(dirPath string) bool {
	_, contains := t.expandedDirs[dirPath]
	return contains
}

func (m *DirectoryTree) createDirectoryItem(dir directories.Directory, level int) Dir {
	style := DirTreeStyle()
	dirItem := Dir{
		Item: Item{
			index:  0,
			Name:   dir.Name,
			Path:   dir.Path,
			styles: style,
		},
		expanded:   dir.IsExpanded,
		parent:     0,
		children:   m.getChildren(dir.Path, level+1),
		NbrFolders: dir.NbrFolders,
		level:      level,
	}
	return dirItem
}

// createVirtualDir creates a temporary, virtual directory `Dir`
//
// This directory is mainly used as a placeholder when creating a directory
func (t *DirectoryTree) createVirtualDir() Dir {
	selectedDir := t.SelectedDir()
	tempFolderName := "New Folder"
	tempFolderPath := filepath.Join(selectedDir.Path, tempFolderName)

	//indent := 1
	//if selectedDir.expanded {
	indent := selectedDir.level + 1
	//}

	return Dir{
		Item: Item{
			index: len(t.dirsListFlat),
			Name:  tempFolderName,
			Path:  tempFolderPath,
		},
		expanded: false,
		children: nil,
		parent:   selectedDir.index,
		level:    indent,
	}
}

// Refresh updates the currently selected tree branch
func (t *DirectoryTree) Refresh() {
	t.RefreshBranch(t.SelectedDir().parent, t.selectedIndex)
}

// RefreshBranch refreshes a tree branch by its branch index
//
// Use `selectAfter` to change the selection after the branch got refreshed
// If  `selectAfter` is -1 the branch root is selected
func (t *DirectoryTree) RefreshBranch(index int, selectAfter int) {
	t.selectedIndex = index
	if dir := findDirInTree(t.items, t.SelectedDir().Path); dir != nil {
		dir.children = t.getChildren(dir.Path, dir.level+1)
	}

	if selectAfter == -1 {
		selectAfter = index
	}

	t.selectedIndex = selectAfter
	t.build()
}

// SelectedDir returns the currently selected directory in the directory tree
func (t *DirectoryTree) SelectedDir() Dir {
	return t.SelectedItem(t.dirsListFlat)
}

func (t *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	t.dirsListFlat = t.flatten(t.items, 0, -1, &nextIndex)
}

// flatten converts a slice of Dir and its sub slices into a one dimensional slice
// that we use to render the directory tree
func (t *DirectoryTree) flatten(dirs []Dir, level int, parent int, nextIndex *int) []Dir {
	var result []Dir
	for i, dir := range dirs {
		dir.index = *nextIndex
		dir.parent = parent
		dir.level = level

		*nextIndex++

		if _, contains := t.expandedDirs[dir.Path]; contains {
			dir.expanded = true
		}

		result = append(result, dir)
		if dir.expanded {
			result = append(
				result,
				t.flatten(dirs[i].children, level+1, dir.index, nextIndex)...,
			)
		}
	}
	return result
}

// Gets the respectively last child of the selected directory.
func (t DirectoryTree) lastChildOfSelection() Dir {
	selectedDir := t.SelectedDir()
	lastChild := t.getLastChild(selectedDir.index)

	if lastChild.expanded && len(lastChild.children) > 0 {
		lastChild = t.getLastChild(lastChild.index)
	}
	return lastChild
}

// getLastChild returns the last child of the item with the given index
func (t DirectoryTree) getLastChild(index int) Dir {
	lastChild := t.dirsListFlat[len(t.dirsListFlat)-1]
	dir := t.dirsListFlat[index]

	//if selectedDir.index > 0 && selectedDir.expanded {
	if dir.index > 0 {
		if len(dir.children) == 0 {
			dir.children = append(dir.children, Dir{})
		}
		if len(dir.children) > 0 {
			lastChild = dir.children[len(dir.children)-1]
			if lastChild.Name == "" {
				lastChild = dir
			}
			for _, dir := range t.dirsListFlat {
				if lastChild.Name == dir.Name {
					lastChild = dir
				}
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
func (m *DirectoryTree) insertDirAfter(afterIndex int, directory Dir) {
	for i, dir := range m.dirsListFlat {
		if dir.index == afterIndex {
			m.dirsListFlat = append(
				m.dirsListFlat[:i+1],
				append([]Dir{directory}, m.dirsListFlat[i+1:]...)...,
			)
			break
		}
	}
}

func (t *DirectoryTree) dirExists(dirPath string) bool {
	//dirName := t.editor.Value()
	//selectedDir := t.selectedDir()
	parentPath := filepath.Dir(dirPath)
	dirName := filepath.Base(dirPath)
	//statusMsg := messages.StatusBarMsg{}
	if _, contains := directories.ContainsDir(parentPath, dirName); contains {
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
func findDirInTree(directories []Dir, path string) *Dir {
	for i := range directories {
		if directories[i].Path == path {
			return &directories[i]
		}
		if directories[i].expanded {
			if ok := findDirInTree(directories[i].children, path); ok != nil {
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
	if t.selectedIndex >= len(t.dirsListFlat) || !t.Focused {
		return messages.StatusBarMsg{}
	}

	if dir := findDirInTree(t.items, t.SelectedDir().Path); dir != nil {
		if dir.expanded {
			delete(t.expandedDirs, dir.Path)
			dir.expanded = false
			t.build()
		}
	}
	return messages.StatusBarMsg{}
}

// Expands the currently selected directory
func (t *DirectoryTree) Expand() messages.StatusBarMsg {
	if t.selectedIndex >= len(t.dirsListFlat) || !t.Focused {
		return messages.StatusBarMsg{}
	}

	if dir := findDirInTree(t.items, t.SelectedDir().Path); dir != nil {
		if !dir.expanded {
			t.expandedDirs[dir.Path] = true
			dir.children = t.getChildren(dir.Path, dir.level+1)
			dir.expanded = true
			t.build()
		}
	}
	return messages.StatusBarMsg{}
}

// Create creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (t *DirectoryTree) Create() messages.StatusBarMsg {
	t.editingState = EditCreate
	t.refreshFlatList()
	t.Expand()

	lastChild := t.lastChildOfSelection()
	tmpdir := t.createVirtualDir()

	t.insertDirAfter(lastChild.index, tmpdir)
	t.selectedIndex = lastChild.index + 1

	if t.editingIndex == nil {
		t.editingIndex = &t.selectedIndex
		t.editor.SetValue(t.SelectedDir().Name)
	}
	return messages.StatusBarMsg{}
}

func (t *DirectoryTree) ConfirmRemove() messages.StatusBarMsg {
	selectedDir := t.SelectedDir()
	msgType := messages.PromptError
	resultMsg := fmt.Sprintf(messages.RemovePromptContent, selectedDir.Path)

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
	resultMsg := fmt.Sprintln(
		messages.SuccessRemove,
		dir.Path,
	)
	msgType := messages.Success

	if err := directories.Delete(dir.Path, false); err == nil {
		t.dirsListFlat = slices.Delete(t.dirsListFlat, index, index+1)
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	t.RefreshBranch(dir.parent, index)
	return messages.StatusBarMsg{Content: resultMsg, Type: msgType}
}

// Confirms a user action
func (t *DirectoryTree) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if t.editingIndex != nil {
		selectedDir := t.SelectedDir()
		oldPath := selectedDir.Path
		newPath := filepath.Join(filepath.Dir(oldPath), t.editor.Value())

		switch t.editingState {
		case EditRename:
			if err := directories.Rename(oldPath, newPath); err == nil {
				t.Refresh()
				t.selectedIndex = t.indexByPath(newPath, t.dirsListFlat)
			}

		case EditCreate:
			if !t.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

		t.CancelAction(func() { t.Refresh() })
		return messages.StatusBarMsg{Content: "yep"}
	}

	return messages.StatusBarMsg{}
}
