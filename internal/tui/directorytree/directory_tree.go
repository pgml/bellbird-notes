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
	selectedIndex int

	dirsList     []Dir // the original directory hierarchy
	dirsListFlat []Dir // a flattened representation to make vertical navigation easier
	content      *list.List

	statusMessage string
}

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

func (d Dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	e := d.styles.enumerator.Render
	//indent := strings.Repeat(" │", d.level)
	indent := strings.Repeat("  ", d.level)
	name := theme.TruncateText(d.name, 22)

	toggle := map[string]string{"open": "", "close": "󰉋"}
	noNerdFonts := true
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
			return t, cmd
		}
	}

	return t, cmd
}

func (t *DirectoryTree) View() string {
	var tree string

	for i, dir := range t.dirsListFlat {
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
	childDir := directories.List(path)
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

// Creates a temporary, virtual dir
func (m *DirectoryTree) createTempDir() Dir {
	selectedDir := m.selectedDir()
	parent := selectedDir.parent
	tempFolderName := "New Folder"
	tempFolderPath := filepath.Join(selectedDir.path, tempFolderName)

	return Dir{
		index:    len(m.dirsListFlat),
		name:     tempFolderName,
		path:     tempFolderPath,
		expanded: false,
		children: nil,
		parent:   parent,
		level:    selectedDir.level + 1,
	}
}

func (m *DirectoryTree) selectedDir() *Dir {
	return &m.dirsListFlat[m.selectedIndex]
}

func (m *DirectoryTree) refreshFlatList() {
	nextIndex := 0
	m.dirsListFlat = m.flatten(m.dirsList, 0, -1, &nextIndex)
}

// Rebuilds `m.dirsList` from `m.dirsListFlat`
func (m *DirectoryTree) rebuildDirsList() {
	dirMap := make(map[int]*Dir)
	var dirsList []Dir

	for i := range m.dirsListFlat {
		dirMap[m.dirsListFlat[i].index] = &m.dirsListFlat[i]
	}

	for i := range m.dirsListFlat {
		dir := &m.dirsListFlat[i]
		if dir.parent == -1 {
			dirsList = append(dirsList, *dir)
		} else {
			if parent, exists := dirMap[dir.parent]; exists {
				parent.children = append(parent.children, *dir)
			}
		}
	}

	m.dirsList = dirsList
	m.refreshFlatList()
}

// Converts a slice of Dir and its sub slices into a one dimensional slice
func (m *DirectoryTree) flatten(dirs []Dir, level int, parent int, nextIndex *int) []Dir {
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
				m.flatten(dirs[i].children, level+1, dir.index, nextIndex)...,
			)
		}
	}
	return result
}

// Expands the currently selected directory
func (m *DirectoryTree) Expand() messages.StatusBarMsg {
	if m.selectedIndex >= len(m.dirsListFlat) {
		return messages.StatusBarMsg{}
	}

	dir := findDirInTree(&m.dirsList, m.dirsListFlat[m.selectedIndex].path)
	if dir != nil {
		if !dir.expanded {
			dir.children = m.getChildren(dir.path, dir.level+1)
		}
		dir.expanded = true
	}
	m.renderTree()
	return messages.StatusBarMsg{}
}

// Collapses the currently selected directory
func (m *DirectoryTree) Collapse() messages.StatusBarMsg {
	if m.selectedIndex >= len(m.dirsListFlat) {
		return messages.StatusBarMsg{}
	}

	dir := findDirInTree(&m.dirsList, m.dirsListFlat[m.selectedIndex].path)
	if dir != nil {
		dir.expanded = false
	}
	m.renderTree()
	return messages.StatusBarMsg{}
}

// Decrements `m.selectedIndex`
func (m *DirectoryTree) MoveUp() messages.StatusBarMsg {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
	return messages.StatusBarMsg{
		Content: strconv.Itoa(m.selectedDir().nbrFolders) + " folders",
	}
}

// Increments `m.selectedIndex`
func (m *DirectoryTree) MoveDown() messages.StatusBarMsg {
	if m.selectedIndex < len(m.dirsListFlat)-1 {
		m.selectedIndex++
	}
	return messages.StatusBarMsg{
		Content: strconv.Itoa(m.selectedDir().nbrFolders) + " folders",
	}
}

// Creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (m *DirectoryTree) Create() messages.StatusBarMsg {
	m.Expand()
	lastChild := m.lastChildOfSelection()
	tmpdir := m.createTempDir()
	m.insertDirAfter(lastChild.index, tmpdir)
	m.selectedIndex = lastChild.index + 1
	//m.rebuildDirsList()
	return m.Rename()
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (m *DirectoryTree) Rename() messages.StatusBarMsg {
	if m.editingIndex == nil {
		m.editingIndex = &m.selectedIndex
		m.editor.SetValue(m.dirsListFlat[m.selectedIndex].name)
	}
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
		dirs := t.dirsListFlat
		t.dirsListFlat = slices.Delete(dirs, index, index+1)
		t.rebuildDirsList()
		// the next four lines are a very, very dirty hack to update the tree
		// but I don't know any better right now...so deal with it
		t.selectedIndex = dir.parent
		t.Collapse()
		t.Expand()
		t.selectedIndex = index
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	return messages.StatusBarMsg{Content: resultMsg, Type: msgType}
}

// Confirms a user action
func (m *DirectoryTree) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if m.editingIndex != nil {
		oldPath := m.selectedDir().path
		newPath := filepath.Join(filepath.Dir(oldPath), m.editor.Value())

		// rename if path exists
		if _, err := os.Stat(oldPath); err == nil {
			directories.Rename(oldPath, newPath)
		} else {
			directories.Create(newPath)
			// since the new directory was only created in the flat copy of the
			// directories we need to update the original list to make the creation
			// visually persistent for this session
			m.rebuildDirsList()
		}

		if dir := findDirInTree(&m.dirsList, oldPath); dir != nil {
			dir.name = filepath.Base(newPath)
			dir.path = newPath
		}

		m.CancelAction()
		return messages.StatusBarMsg{Content: "yep"}
	}

	return messages.StatusBarMsg{}
}

// Cancel the current action and blurs the editor
func (m *DirectoryTree) CancelAction() messages.StatusBarMsg {
	m.editingIndex = nil
	m.editor.Blur()
	m.renderTree()
	return messages.StatusBarMsg{}
}

// Gets the respectively last child of the selected directory
func (m *DirectoryTree) lastChildOfSelection() Dir {
	selectedDir := m.selectedDir()
	lastChild := m.dirsListFlat[len(m.dirsListFlat)-1]
	if selectedDir.index > 0 {
		for _, item := range m.dirsListFlat {
			if item.parent == selectedDir.index {
				lastChild = item
			}
		}
	}
	return lastChild
}

// Inserts an item after `afterIndex`
//
// Note: this is only a temporary insertion into to the flat copy
// of the directories.
// To make it persistent use `m.rebuildDirsList` afterwards
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
func findDirInTree(directories *[]Dir, path string) *Dir {
	for i := range *directories {
		if (*directories)[i].path == path {
			return &(*directories)[i]
		}
		if (*directories)[i].expanded {
			if ok := findDirInTree(&(*directories)[i].children, path); ok != nil {
				return ok
			}
		}
	}
	return nil
}
