package tui

import (
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"
	"sort"

	"github.com/charmbracelet/lipgloss"
	lgtree "github.com/charmbracelet/lipgloss/tree"
	bl "github.com/winder/bubblelayout"
)

type directoryTree struct {
	id            bl.ID
	size          bl.Size
	isFocused     bool
	selectedIndex int
	rowsInfo      map[int]*dir
	content       *lgtree.Tree
}

type styles struct {
	base,
	block,
	enumerator,
	dir,
	toggle,
	file lipgloss.Style
}

func defaultStyles() styles {
	var s styles
	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		PaddingLeft(1)
	s.block = s.base.
		Padding(1, 3).
		Margin(1, 3).
		Width(20)
	s.enumerator = s.base.
		Foreground(lipgloss.NoColor{})
		//PaddingRight(1)
	//s.dir = s.base.
	//	Inline(true)
	//s.toggle = s.base.
	//	Foreground(lipgloss.Color("207")).
	//	PaddingRight(1)
	s.file = s.base
	return s
}

type dir struct {
	name       string
	path       string
	open       bool
	nbrNotes   int
	nbrFolders int
	children   []dir
	styles     styles
}

func (d dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	name := truncateText(d.name, 22)
	if d.open {
		return t(" ") + n(name)
	}
	return t("󰉋 ") + n(name)
}

type file struct {
	name   string
	styles styles
}

func (s file) String() string {
	return s.styles.file.Render(s.name)
}

func newDirectoryTree() *directoryTree {
	tree := &directoryTree{}
	tree.rowsInfo = make(map[int]*dir)
	tree.buildDirectoryTree()

	return tree
}

func (t *directoryTree) buildDirectoryTree() {
	style := defaultStyles()
	conf := config.New()
	notesDir := conf.Value(config.General, config.DefaultNotesDirectory)

	// add root directory
	t.rowsInfo[-1] = &dir{"Folders", notesDir, true, 0, 0, nil, style}
	// append all directory children
	for index, child := range directories.List(notesDir) {
		childItem := t.makeChild(child)
		t.rowsInfo[index] = &childItem
	}

	t.selectedIndex = 2
	t.renderTree()
	t.refreshTreeStyle()
}

func (t *directoryTree) renderTree() {
	style := defaultStyles()
	rootDir := t.rowsInfo[-1]
	dirTree := lgtree.Root(rootDir).
		Enumerator(lgtree.RoundedEnumerator).
		EnumeratorStyle(style.enumerator)

	for _, key := range getSortedKeys(t.rowsInfo) {
		child := t.rowsInfo[key]

		if len(child.children) > 0 && child.open {
			ch := lgtree.Root(child)
			for _, c := range child.children {
				ch.Child(c)
			}
			dirTree.Child(ch)
		} else {
			dirTree.Child(child)
		}
	}

	t.content = dirTree
}

func (t *directoryTree) collapseChild(childIndex int) {
	t.rowsInfo[childIndex].open = false
	t.renderTree()
	t.refreshTreeStyle()
}

func (t *directoryTree) expandChild(childIndex int) {
	t.getChildren(childIndex)
	t.rowsInfo[childIndex].open = true
	t.renderTree()
	t.refreshTreeStyle()
}

func (t *directoryTree) getChildren(childIndex int) {
	child := t.rowsInfo[childIndex]
	childDir := directories.List(child.path)
	// only get child directories if not present already
	if len(childDir) > 0 && len(child.children) <= 0 {
		for _, item := range childDir {
			//app.LogDebug(i)
			child.children = append(child.children, t.makeChild(item))
		}
	}
}

func (t *directoryTree) makeChild(child directories.Directory) dir {
	style := defaultStyles()
	childItem := dir{
		child.Name,
		child.Path,
		child.IsExpanded,
		child.NbrNotes,
		child.NbrFolders,
		nil,
		style,
	}
	return childItem
}

func (t *directoryTree) refreshTreeStyle() {
	style := defaultStyles()

	t.content = t.content.EnumeratorStyle(style.enumerator).
		ItemStyleFunc(func(c lgtree.Children, i int) lipgloss.Style {
			style := style.base.Width(25).MaxWidth(t.size.Width)
			if t.selectedIndex == i {
				return style.Background(lipgloss.Color("#424B5D")).Bold(true)
			}
			return style
		})
}

func (t *directoryTree) moveUp() {
	if t.selectedIndex > 0 {
		t.selectedIndex--
		t.refreshTreeStyle()
	}
}

func (t *directoryTree) moveDown() {
	if t.selectedIndex < t.content.Children().Length()-1 {
		t.selectedIndex++
		t.refreshTreeStyle()
	}
}

func truncateText(text string, maxWidth int) string {
	if lipgloss.Width(text) > maxWidth {
		if maxWidth > 3 {
			return text[:maxWidth-3] + "..."
		}
		return text[:maxWidth] // No space for "..."
	}
	return text
}

func getSortedKeys[T any](mapToSort map[int]T) []int {
	var keys []int
	for key := range mapToSort {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	return keys
}
