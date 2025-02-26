package tui

import (
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	bl "github.com/winder/bubblelayout"
)

type treeModel struct {
	id            bl.ID
	size          bl.Size
	isFocused     bool
	selectedIndex int
	// the original directory hierarchy
	dirsList []dir
	// a flattened representation to make vertical navigation easier
	dirsListFlat []dir
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
	name := truncateText(d.name, 22)
	if d.expanded {
		return e(indent) + t("") + n(name)
	}
	return e(indent) + t("󰉋") + n(name)
}

func newDirectoryTree() *treeModel {
	tree := &treeModel{}
	conf := config.New()
	notesDir := conf.Value(config.General, config.DefaultNotesDirectory)

	for _, child := range directories.List(notesDir) {
		childItem := tree.makeChild(child, 0)
		tree.dirsList = append(tree.dirsList, childItem)
	}

	tree.selectedIndex = 0
	tree.renderTree()
	return tree
}

func (m *treeModel) renderTree() {
	m.refreshFlatList()

	dirTree := list.New().
		Enumerator(func(items list.Items, index int) string { return "" })

	for _, dir := range m.dirsListFlat {
		dirTree.Item(dir)
	}

	m.content = dirTree
	m.refreshTreeStyle()
}

// Reads a directory of `path` and return dir slice
func (m *treeModel) getChildren(path string, level int) []dir {
	var dirs []dir
	childDir := directories.List(path)
	for _, item := range childDir {
		dirs = append(dirs, m.makeChild(item, level))
	}
	return dirs
}

// Creates a dir
func (m *treeModel) makeChild(child directories.Directory, level int) dir {
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

func (m *treeModel) refreshFlatList() {
	m.dirsListFlat = m.flatten(m.dirsList, 0)
}

func (m *treeModel) refreshTreeStyle() {
	style := defaultStyles()
	m.content = m.content.EnumeratorStyle(style.enumerator).
		ItemStyleFunc(func(c list.Items, i int) lipgloss.Style {
			style := style.base.Width(30).MaxWidth(m.size.Width)
			if m.selectedIndex == i {
				return style.Background(lipgloss.Color("#424B5D")).Bold(true)
			}
			return style
		})
}

func (m *treeModel) flatten(items []dir, level int) []dir {
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

func (m *treeModel) expand() {
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

func (m *treeModel) collapse() {
	if m.selectedIndex >= len(m.dirsListFlat) {
		return
	}

	dir := findDirectoryInTree(&m.dirsList, m.dirsListFlat[m.selectedIndex].path)
	if dir != nil {
		dir.expanded = false
	}
	m.renderTree()
}

func (m *treeModel) moveUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
		m.refreshTreeStyle()
	}
}

func (m *treeModel) moveDown() {
	if m.selectedIndex < len(m.dirsListFlat)-1 {
		m.selectedIndex++
		m.refreshTreeStyle()
	}
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
