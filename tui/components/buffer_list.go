package components

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"bellbird-notes/app/config"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components/textarea"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
)

type BufferListItem struct {
	Item
	cursorPos textarea.CursorPos
}

// Index returns the index of a Note-Item
func (b BufferListItem) Index() int { return b.index }

// Path returns the index of a Note-Item
func (b BufferListItem) Path() string { return b.path }

// String is string representation of a Note
func (b BufferListItem) String() string {
	baseStyle := b.styles.base.Width(b.width)
	selectedStyle := b.styles.selected.Width(b.width)

	if b.selected {
		baseStyle = selectedStyle
	}

	var list strings.Builder

	list.WriteString(theme.Icon(theme.IconNote, b.nerdFonts))
	list.WriteString("  ")

	list.WriteString(utils.RelativePath(b.Path(), true))
	list.WriteByte(':')
	list.WriteString(strconv.Itoa(b.cursorPos.Row))

	return baseStyle.Render(list.String())
}

type BufferList struct {
	List[BufferListItem]

	Width  int
	Height int
}

func NewBufferList(conf *config.Config) *BufferList {
	termW, _ := theme.TerminalSize()

	panel := &BufferList{
		List:   List[BufferListItem]{conf: conf},
		Height: 10,
		Width:  termW / 3,
	}
	panel.focused = false
	panel.Mode = mode.Normal

	return panel
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (l *BufferList) Init() tea.Cmd {
	return nil
}

func (l *BufferList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case BuffersChangedMsg:
		buffers := *msg.Buffers
		l.items = make([]BufferListItem, 0, len(buffers))

		for i, buf := range buffers {
			l.items = append(l.items, l.createListItem(buf, i))
		}
	}

	if l.focused {
		if !l.Ready {
			l.viewport = viewport.New()
			l.viewport.SetContent(l.render())
			l.viewport.KeyMap = viewport.KeyMap{}
			l.Ready = true
		}

		l.viewport, cmd = l.viewport.Update(msg)
	}

	return l, cmd
}

func (l *BufferList) View() string {
	if !l.Ready {
		return "\n  Initializing..."
	}

	l.viewport.SetContent(l.render())

	l.viewport.Style = theme.BaseColumnLayout(
		l.Size,
		l.Focused(),
	)

	var view strings.Builder
	view.WriteString(l.BuildHeader(l.Width, false))
	view.WriteString(l.viewport.View())

	return view.String()
}

func (l *BufferList) RefreshSize() {
	vp := l.viewport
	if vp.Width() != l.Width && vp.Height() != l.Height {
		l.viewport.SetWidth(l.Width)
		l.viewport.SetHeight(l.Height)
	}
}

// BuildHeader builds title of the directory tree column
func (l *BufferList) BuildHeader(width int, rebuild bool) string {
	// return cached header
	if l.header != nil && !rebuild {
		if width == lipgloss.Width(*l.header) {
			return *l.header
		}
	}

	header := theme.Header("Buffers", width, l.Focused()) + "\n"
	l.header = &header
	return header
}

func (l *BufferList) render() string {
	var list strings.Builder

	for i, item := range l.items {
		item.selected = (l.selectedIndex == i)
		item.index = i

		list.WriteString(item.String())
		list.WriteByte('\n')
	}

	l.length = len(l.items)

	return list.String()
}

func (l *BufferList) createListItem(buf Buffer, index int) BufferListItem {
	item := BufferListItem{
		Item: Item{
			index:     index,
			name:      buf.Name(),
			path:      buf.Path,
			selected:  index == l.selectedIndex,
			styles:    NotesListStyle(),
			nerdFonts: l.conf.NerdFonts(),
			width:     l.Width,
		},
		cursorPos: buf.CursorPos,
	}

	return item
}

func (l *BufferList) Items() []BufferListItem {
	return l.items
}

func (l *BufferList) CancelAction(cb func()) message.StatusBarMsg {
	l.SetFocus(false)
	l.SetSelectedIndex(0)

	return message.StatusBarMsg{}
}
