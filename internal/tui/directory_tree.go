package tui

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/config"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

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
		//Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("225"))
	s.block = s.base.
		Padding(1, 3).
		Margin(1, 3).
		Width(40)
	s.enumerator = s.base.
		Foreground(lipgloss.Color("212")).
		PaddingRight(1)
	s.dir = s.base.
		Inline(true)
	s.toggle = s.base.
		Foreground(lipgloss.Color("207")).
		PaddingRight(1)
	s.file = s.base
	return s
}

type dir struct {
	name   string
	open   bool
	styles styles
}

func (d dir) String() string {
	t := d.styles.toggle.Render
	n := d.styles.dir.Render
	if d.open {
		return t("") + n(d.name)
	}
	return t("󰉋") + n(d.name)
}

type file struct {
	name   string
	styles styles
}

func (s file) String() string {
	return s.styles.file.Render(s.name)
}

func getDirectoryTree() *tree.Tree {
	s := defaultStyles()
	conf := config.New()

	tree := tree.Root(dir{"Folders", true, s}).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(s.enumerator)
		//Child(
		//	dir{"ayman", false, s},
		//	tree.Root(dir{"bash", true, s}).
		//		Child(
		//			tree.Root(dir{"tools", true, s}).
		//				Child(
		//					file{"zsh", s},
		//					file{"doom-emacs", s},
		//				),
		//		),
		//	tree.Root(dir{"carlos", true, s}).
		//		Child(
		//			tree.Root(dir{"emotes", true, s}).
		//				Child(
		//					file{"chefkiss.png", s},
		//					file{"kekw.png", s},
		//				),
		//		),
		//	dir{"maas", false, s},
		//)

	notesDir := conf.Value(config.General, config.DefaultNotesDirectory)
	dirs, err := os.ReadDir(notesDir)
	if err != nil {
		app.LogErr(err)
	}

	for _, child := range dirs {
		tree.Child(dir{child.Name(), false, s})
	}

	return tree
}
