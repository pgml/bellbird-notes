package tui

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/directories"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	bl "github.com/winder/bubblelayout"
)

type directoryTree struct {
	id            bl.ID
	size          bl.Size
	isFocused     bool
	selectedIndex int
	content       *tree.Tree
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
	name     string
	path     string
	open     bool
	styles   styles
	tree.Node
}

func (d dir) String() string {
	//t := d.styles.toggle.Render
	n := d.styles.dir.Render
	name := truncateText(d.name, 22)
	if d.open {
		//return t(" ") + n(name)
		return n(name)
	}
	//return t("󰉋 ") + n(name)
	return n(name)
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

	tree.buildDirectoryTree()
	return tree
}

func (t *directoryTree) buildDirectoryTree() {
	style := defaultStyles()
	conf := config.New()
	notesDir := conf.Value(config.General, config.DefaultNotesDirectory)

	tree := tree.Root(dir{"Folders", notesDir, true, style, nil}).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(style.enumerator)

	app.LogDebug(notesDir)

	for _, child := range directories.List(notesDir) {
		tree.Child(dir{child.Name, child.Path, child.IsExpanded, style, nil})
	}

	t.selectedIndex = 2
	t.content = tree
	t.refreshTreeStyle()
}

func (t *directoryTree) collapseChild(childIndex int) {

}

func (t *directoryTree) expandChild(childIndex int) {
	child := t.content.Children().At(childIndex).(dir)

	app.LogDebug(directories.List(child.path))
}

func (t *directoryTree) refreshTreeStyle() {
	style := defaultStyles()

	t.content = t.content.EnumeratorStyle(style.enumerator).
		ItemStyleFunc(func(c tree.Children, i int) lipgloss.Style {
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
	if t.selectedIndex < t.content.Children().Length() - 1 {
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
