package directorytree

import (
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/theme"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	bl "github.com/winder/bubblelayout"
)

type DirectoryTree struct {
	Id        bl.ID
	Size      bl.Size
	IsFocused bool
	Mode      mode.Mode

	editor        textinput.Model
	editingIndex  *int
	editingState  EditState
	selectedIndex int

	dirsList     []Dir // the original directory hierarchy
	dirsListFlat []Dir // a flattened representation to make vertical navigation easier
	content      *list.List

	statusMessage string
}

type EditState int

const (
	EditNone EditState = iota
	EditCreate
	EditRename
)

type statusMsg string

type styles struct {
	base,
	enumerator,
	dir,
	toggle lipgloss.Style
}

func defaultStyles() styles {
	var s styles
	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		MarginLeft(0).
		PaddingLeft(1)
	s.dir = s.base.
		MarginRight(0).
		PaddingRight(0).
		Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#eee"})
	return s
}

type Dir struct {
	index      int
	name       string
	path       string
	expanded   bool
	selected   bool
	level      int
	parent     int
	nbrNotes   int
	nbrFolders int
	children   []Dir
	styles     styles
}
type comparable interface{ Dir }

func (d Dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	e := d.styles.enumerator.Render
	//indent := strings.Repeat("│ ", d.level)
	indent := strings.Repeat("  ", d.level)
	name := theme.TruncateText(d.name, 22)

	toggle := map[string]string{"open": "", "close": "󰉋"}
	noNerdFonts := false
	if noNerdFonts {
		toggle = map[string]string{"open": "▼", "close": "▶"}
	}
	baseStyle := lipgloss.NewStyle().Width(30)

	if d.selected {
		baseStyle = baseStyle.Background(lipgloss.Color("#424B5D")).Bold(true)
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

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (t *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (t *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.editingIndex != nil && !t.editor.Focused() {
			t.editor.Focus()
			return t, nil
		}

		if t.editor.Focused() {
			t.editor.Focus()
			t.editor, cmd = t.editor.Update(msg)
			//t.dirsListFlat[len(t.dirsListFlat)-1].name = t.editor.Value()
			return t, cmd
		}
	}

	return t, cmd
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

func (t *DirectoryTree) View() string {
	var tree string

	for i, dir := range t.dirsListFlat {
		if dir.index == 0 && dir.parent == 0 {
			t.dirsListFlat = slices.Delete(t.dirsListFlat, i, i+1)
			continue
		}

		indent := strings.Repeat("  ", dir.level)
		dir.selected = (t.selectedIndex == i)

		//style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
		//tree += style.Render(fmt.Sprintf("%02d", dir.index)) + " "
		if t.editingIndex != nil && i == *t.editingIndex {
			tree += indent + t.editor.View() + "\n" // Show input field instead of text
		} else {
			dir.styles.base.Background(lipgloss.Color("#424B5D")).Bold(true)
			tree += dir.String() + "\n"
		}
	}

	return theme.BaseColumnLayout(t.Size, t.IsFocused).
		Align(lipgloss.Left).
		Render(tree)
}

func New() *DirectoryTree {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	tree := &DirectoryTree{
		selectedIndex: 0,
		editingIndex:  nil,
		editor:        ti,
	}
	conf := config.New()
	notesDir := conf.Value(config.General, config.DefaultNotesDirectory)

	// append root directory
	tree.dirsList = append(tree.dirsList, Dir{
		index:    0,
		name:     "Bellbird Notes",
		expanded: true,
		level:    0,
		path:     notesDir,
		parent:   -1,
		children: tree.getChildren(notesDir, 0),
		styles:   defaultStyles(),
	})

	tree.renderTree()
	return tree
}

func (m *DirectoryTree) renderTree() {
	dirTree := list.New().
		Enumerator(func(items list.Items, index int) string { return "" })

	m.refreshFlatList()
	for _, dir := range m.dirsListFlat {
		dirTree.Item(dir)
	}

	m.content = dirTree
}

// Reads a directory of `path` and return dir slice
func (m *DirectoryTree) getChildren(path string, level int) []Dir {
	var dirs []Dir
	childDir, _ := directories.List(path)
	for _, item := range childDir {
		dirs = append(dirs, m.createDir(item, level))
	}
	return dirs
}

// Creates a dir
func (m *DirectoryTree) createDir(dir directories.Directory, level int) Dir {
	style := defaultStyles()
	childItem := Dir{
		index:      0,
		name:       dir.Name,
		path:       dir.Path,
		expanded:   dir.IsExpanded,
		parent:     0,
		children:   nil,
		nbrFolders: dir.NbrFolders,
		level:      level,
		styles:     style,
	}
	return childItem
}

// Creates a temporary, virtual directory `Dir`
func (m *DirectoryTree) createVirtualDir() Dir {
	selectedDir := m.selectedDir()
	tempFolderName := "New Folder"
	tempFolderPath := filepath.Join(selectedDir.path, tempFolderName)

	return Dir{
		index:    len(m.dirsListFlat),
		name:     tempFolderName,
		path:     tempFolderPath,
		expanded: false,
		children: nil,
		parent:   selectedDir.index,
		level:    selectedDir.level + 1,
	}
}

// Returns the currently selected directory in the directory tree
// or the first if there's no selected for some reaon
func (t *DirectoryTree) selectedDir() Dir {
	for i := range t.dirsListFlat {
		dir := t.dirsListFlat[i]
		if i == t.selectedIndex {
			return dir
		}
	}
	return t.dirsListFlat[0]
}

func (t *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	t.dirsListFlat = t.flatten(t.dirsList, 0, -1, &nextIndex)
}

// Converts a slice of Dir and its sub slices into a one dimensional slice
func (t *DirectoryTree) flatten(dirs []Dir, level int, parent int, nextIndex *int) []Dir {
	var result []Dir
	for i, dir := range dirs {
		dir.index = *nextIndex
		dir.parent = parent
		dir.level = level

		*nextIndex++

		result = append(result, dir)
		if dirs[i].expanded {
			result = append(
				result,
				t.flatten(dirs[i].children, level+1, dir.index, nextIndex)...,
			)
		}
	}
	return result
}

// Expands the currently selected directory
func (t *DirectoryTree) Expand() messages.StatusBarMsg {
	if t.selectedIndex >= len(t.dirsListFlat) {
		return messages.StatusBarMsg{}
	}

	if dir := findDirInTree(t.dirsList, t.selectedDir().path); dir != nil {
		if !dir.expanded {
			dir.children = t.getChildren(dir.path, dir.level+1)
		}
		dir.expanded = true
	}
	t.renderTree()
	return messages.StatusBarMsg{}
}

// Collapses the currently selected directory
func (t *DirectoryTree) Collapse() messages.StatusBarMsg {
	if t.selectedIndex >= len(t.dirsListFlat) {
		return messages.StatusBarMsg{}
	}

	if dir := findDirInTree(t.dirsList, t.selectedDir().path); dir != nil {
		dir.expanded = false
	}
	t.renderTree()
	return messages.StatusBarMsg{}
}

// Refreshes a tree branch by its branch index
// by collapsing and expanding it right after...
//
// Use `selectAfter` to change the selection after the branch got refreshed
// If  `selectAfter` is -1 the branch root is selected
//
// A bit hacky but it works
func (t *DirectoryTree) RefreshTreeBranch(index int, selectAfter int) {
	t.selectedIndex = index
	t.Collapse()
	t.Expand()
	t.refreshFlatList()

	if selectAfter == -1 {
		selectAfter = index
	}

	t.selectedIndex = selectAfter
}

// Decrements `m.selectedIndex`
func (t *DirectoryTree) MoveUp() messages.StatusBarMsg {
	if t.selectedIndex > 0 {
		t.selectedIndex--
	}
	return messages.StatusBarMsg{
		Content: strconv.Itoa(t.selectedDir().nbrFolders) + " folders",
	}
}

// Increments `m.selectedIndex`
func (t *DirectoryTree) MoveDown() messages.StatusBarMsg {
	if t.selectedIndex < len(t.dirsListFlat)-1 {
		t.selectedIndex++
	}
	return messages.StatusBarMsg{
		Content: strconv.Itoa(t.selectedDir().nbrFolders) + " folders",
	}
}

// Creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (t *DirectoryTree) Create() messages.StatusBarMsg {
	t.Collapse()
	t.Expand()
	t.editingState = EditCreate
	t.refreshFlatList()
	lastChild := t.lastChildOfSelection()
	//app.LogDebug(lastChild)
	tmpdir := t.createVirtualDir()
	t.insertDirAfter(lastChild.index, tmpdir)
	t.selectedIndex = lastChild.index + 1

	if t.editingIndex == nil {
		t.editingIndex = &t.selectedIndex
		t.editor.SetValue(t.selectedDir().name)
	}
	return messages.StatusBarMsg{}
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (t *DirectoryTree) Rename() messages.StatusBarMsg {
	if t.editingIndex == nil {
		t.editingState = EditRename
		t.editingIndex = &t.selectedIndex
		t.editor.SetValue(t.selectedDir().name)
		// set cursor to last position
		t.editor.SetCursor(100)
	}
	return messages.StatusBarMsg{}
}

func (t *DirectoryTree) GoToTop() messages.StatusBarMsg {
	t.selectedIndex = 0
	return messages.StatusBarMsg{}
}

func (t *DirectoryTree) GoToBottom() messages.StatusBarMsg {
	t.selectedIndex = t.dirsListFlat[len(t.dirsListFlat)-1].index
	return messages.StatusBarMsg{}
}

func (t *DirectoryTree) ConfirmRemove() messages.StatusBarMsg {
	selectedDir := t.selectedDir()
	msgType := messages.PromptError
	resultMsg := fmt.Sprintf(messages.RemovePromptContent, selectedDir.path)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  messages.SenderDirTree,
	}
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (t *DirectoryTree) Remove() messages.StatusBarMsg {
	dir := t.selectedDir()
	index := t.selectedIndex
	resultMsg := fmt.Sprintf(messages.SuccessRemove, dir.path)
	msgType := messages.Success

	if err := directories.Delete(dir.path, false); err == nil {
		t.dirsListFlat = slices.Delete(t.dirsListFlat, index, index+1)
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	t.RefreshTreeBranch(dir.parent, index)
	return messages.StatusBarMsg{Content: resultMsg, Type: msgType}
}

// Confirms a user action
func (t *DirectoryTree) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if t.editingIndex != nil {
		oldPath := t.selectedDir().path
		newPath := filepath.Join(filepath.Dir(oldPath), t.editor.Value())

		switch t.editingState {
		case EditRename:
			// rename if path exists
			if _, err := os.Stat(oldPath); err == nil {
				directories.Rename(oldPath, newPath)
				if dir := findDirInTree(t.dirsList, oldPath); dir != nil {
					dir.name = filepath.Base(newPath)
					dir.path = newPath
					t.RefreshTreeBranch(dir.parent, dir.index)
				}
			}

		case EditCreate:
			if !t.dirExists(newPath) {
				directories.Create(newPath)
			}
		}

		t.CancelAction()
		return messages.StatusBarMsg{Content: "yep"}
	}

	return messages.StatusBarMsg{}
}

// Cancel the current action and blurs the editor
func (t *DirectoryTree) CancelAction() messages.StatusBarMsg {
	t.editingIndex = nil
	t.editingState = EditNone
	t.editor.Blur()
	t.RefreshTreeBranch(t.selectedDir().parent, t.selectedIndex)
	return messages.StatusBarMsg{}
}

// Gets the respectively last child of the selected directory.
func (m *DirectoryTree) lastChildOfSelection() Dir {
	selectedDir := m.selectedDir()
	lastChild := m.dirsListFlat[len(m.dirsListFlat)-1]

	if selectedDir.index > 0 {
		if len(selectedDir.children) == 0 {
			selectedDir.children = append(selectedDir.children, Dir{})
		}
		if len(selectedDir.children) > 0 {
			lastChild = selectedDir.children[len(selectedDir.children)-1]
			if lastChild.name == "" {
				lastChild = selectedDir
			}
			for _, dir := range m.dirsListFlat {
				if lastChild.name == dir.name {
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

// Recursively search for a directory by path
func findDirInTree(directories []Dir, path string) *Dir {
	for i := range directories {
		if directories[i].path == path {
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
