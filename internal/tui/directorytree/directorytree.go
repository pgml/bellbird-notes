package directorytree

import (
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/theme"
	"path"
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

	dirsList     []dir // the original directory hierarchy
	dirsListFlat []dir // a flattened representation to make vertical navigation easier
	content      *list.List
}

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

type dir struct {
	name     string
	path     string
	expanded bool
	selected bool
	level    int
	children []dir
	styles   styles
}

func (d dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	e := d.styles.enumerator.Render
	//indent := strings.Repeat(" │", d.level)
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

}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (m *DirectoryTree) Init() tea.Cmd {
	return nil
}

func (m *DirectoryTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editingIndex != nil && !m.editor.Focused() {
			m.editor.Focus()
			return m, nil
		}

		if m.editor.Focused() {
			m.editor.Focus()
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
	}

	return m, cmd
}

func (t *DirectoryTree) View() string {
	var tree string

	for i, dir := range t.dirsListFlat {
		indent := strings.Repeat("  ", dir.level)
		dir.selected = (t.selectedIndex == i)

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

	tree.dirsList = append(tree.dirsList, dir{
		name:     "Bellbird Notes",
		expanded: true,
		level:    0,
		path:     notesDir,
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
func (m *DirectoryTree) getChildren(path string, level int) []dir {
	var dirs []dir
	childDir := directories.List(path)
	for _, item := range childDir {
		dirs = append(dirs, m.makeChild(item, level))
	}
	return dirs
}

// Creates a dir
func (m *DirectoryTree) makeChild(child directories.Directory, level int) dir {
	style := defaultStyles()
	childItem := dir{
		name:     child.Name,
		path:     child.Path,
		expanded: child.IsExpanded,
		children: nil,
		level:    level,
		styles:   style,
	}
	return childItem
}

func (m *DirectoryTree) selectedDir() *dir {
	return &m.dirsListFlat[m.selectedIndex]
}

func (m *DirectoryTree) refreshFlatList() {
	m.dirsListFlat = m.flatten(m.dirsList, 0)
}

func (m *DirectoryTree) flatten(items []dir, level int) []dir {
	var result []dir
	for i := range items {
		items[i].level = level
		result = append(result, items[i])
		if items[i].expanded {
			result = append(result, m.flatten(items[i].children, level+1)...)
		}
	}
	return result
}

func (m *DirectoryTree) Expand() {
	if m.selectedIndex >= len(m.dirsListFlat) {
		return
	}

	dir := findDirectoryInTree(&m.dirsList, m.dirsListFlat[m.selectedIndex].path)
	if dir != nil {
		if !dir.expanded {
			dir.children = m.getChildren(dir.path, dir.level+1)
		}
		dir.expanded = true
	}
	m.renderTree()
}

func (m *DirectoryTree) Collapse() {
	if m.selectedIndex >= len(m.dirsListFlat) {
		return
	}

	dir := findDirectoryInTree(&m.dirsList, m.dirsListFlat[m.selectedIndex].path)
	if dir != nil {
		dir.expanded = false
	}
	m.renderTree()
}

func (m *DirectoryTree) MoveUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

func (m *DirectoryTree) MoveDown() {
	if m.selectedIndex < len(m.dirsListFlat)-1 {
		m.selectedIndex++
	}
}

func (m *DirectoryTree) Rename() {
	if m.editingIndex == nil {
		m.editingIndex = &m.selectedIndex
		m.editor.SetValue(m.dirsListFlat[m.selectedIndex].name)
	}
}

func (m *DirectoryTree) ConfirmAction() {
	if m.editingIndex != nil {
		oldPath := m.selectedDir().path
		newPath := path.Join(path.Dir(oldPath), m.editor.Value())
		directories.Rename(oldPath, newPath)

		dir := findDirectoryInTree(&m.dirsList, oldPath)
		dir.name = path.Base(newPath)
		dir.path = newPath

		m.CancelAction()
	}
}

func (m *DirectoryTree) CancelAction() {
	m.editingIndex = nil
	m.editor.Blur()

	m.renderTree()
}

// Recursively search for an item by path
func findDirectoryInTree(directories *[]dir, path string) *dir {
	for i := range *directories {
		if (*directories)[i].path == path {
			return &(*directories)[i]
		}
		if (*directories)[i].expanded {
			if ok := findDirectoryInTree(&(*directories)[i].children, path); ok != nil {
				return ok
			}
		}
	}
	return nil
}
