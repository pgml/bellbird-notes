package components

import (
	"path"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
)

type BufferListItem struct {
	Item

	buffer *Buffer
}

// Index returns the index of a BufferListItem
func (b BufferListItem) Index() int { return b.index }

// Name returns the path of a BufferListItem
func (b BufferListItem) Name() string { return b.name }

// Path returns the path of a BufferListItem
func (b BufferListItem) Path() string { return b.path }

// PathOnly returns the relative path of a BufferListItem wihout the filename
func (b BufferListItem) PathOnly() string {
	p := path.Dir(b.Path())

	notesRoot, _ := app.NotesRootDir()

	if p == notesRoot {
		return "/"
	}

	return p
}

// IsCut returns whether a bufferlist item is cut
func (b BufferListItem) IsCut() bool { return b.isCut }

// SetIsCut returns whether the buffer list item is cut
func (b *BufferListItem) SetIsCut(isCut bool) { b.isCut = isCut }

func (b BufferListItem) render(
	content string,
	faded bool,
	dirty bool,
	w, pl, pr int,
) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		PaddingLeft(pl).
		PaddingRight(pr).
		Width(w)

	if faded {
		style = style.Foreground(theme.ColourBorder)
	} else if dirty {
		style = style.Foreground(theme.ColourDirty)
	}

	if b.selected {
		style = style.Background(theme.ColourBgSelected)
	}

	return style.Render(content)
}

// String is string representation of a Note
func (b BufferListItem) String() string {
	index := b.render(strconv.Itoa(b.index+1), false, false, 0, 0, 2)
	name := b.render(b.name, false, false, 0, 0, 2)

	icon := theme.Icon(theme.IconNote, b.nerdFonts)
	if b.buffer.Dirty {
		icon = theme.Icon(theme.IconDot, b.nerdFonts)
	}
	iconRender := b.render(icon, false, b.buffer.Dirty, 3, 0, 0)

	pathWidth := b.width - lipgloss.Width(index) - lipgloss.Width(iconRender) - lipgloss.Width(name)
	ellipsisWidth := 4
	path := utils.TruncateText(
		utils.RelativePath(b.PathOnly(), true),
		pathWidth-ellipsisWidth,
	)
	path = b.render(path, true, false, pathWidth, 0, 0)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		index, iconRender, name, path,
	)
}

type BufferList struct {
	List[*BufferListItem]

	width  int
	height int

	// Buffers holds all the open buffers
	Buffers *Buffers
}

func NewBufferList(conf *config.Config) *BufferList {
	termW, _ := theme.TerminalSize()

	panel := &BufferList{
		List: List[*BufferListItem]{
			title: "Open Notes",
			conf:  conf,
		},
		height: 10,
		width:  termW / 3,
	}
	panel.focused = false
	panel.Mode = mode.Normal

	return panel
}

func (l BufferList) Name() string { return "BufferList" }

func (l BufferList) Width() int {
	return l.viewport.Width()
}

func (l BufferList) ListSize() (int, int) {
	w, _ := theme.TerminalSize()
	return w / 3, 10
}

func (l *BufferList) UpdateSize() {
	w, h := l.ListSize()
	l.viewport.SetWidth(w)
	l.viewport.SetHeight(h)
	l.width = w
	l.height = h
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (l *BufferList) Init() tea.Cmd {
	return nil
}

func (l *BufferList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if l.Focused() {
			// check if input is a numeric value
			num, err := strconv.Atoi(msg.String())

			// switch to buffer if existent
			if err == nil && num-1 < len(l.items) {
				path := l.items[num-1].path
				cmds = append(cmds, SendSwitchBufferMsg(path, true))
			}
		}

	case tea.WindowSizeMsg:
		l.UpdateSize()

	case BuffersChangedMsg:
		l.buildItems()
	}

	if l.focused {
		if !l.Ready {
			l.viewport = viewport.New()
			l.viewport.SetContent(l.render())
			l.viewport.KeyMap = viewport.KeyMap{}
			l.Ready = true
		}

		var cmd tea.Cmd
		l.viewport, cmd = l.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return l, tea.Batch(cmds...)
}

func (l *BufferList) buildItems() {
	buffers := *l.Buffers
	l.items = make([]*BufferListItem, 0, len(buffers))

	for i := range buffers {
		item := l.createListItem(&buffers[i], i)
		l.items = append(l.items, &item)
	}
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
	view.WriteString(l.BuildHeader(l.width, false))
	view.WriteString(l.viewport.View())

	return view.String()
}

func (l *BufferList) SetBuffers(b *Buffers) {
	l.Buffers = b
}

func (l *BufferList) RefreshSize() {
	vp := l.viewport
	if vp.Width() != l.width && vp.Height() != l.height {
		l.viewport.SetWidth(l.width)
		l.viewport.SetHeight(l.height)
	}
}

func (l *BufferList) render() string {
	var list strings.Builder

	if l.items == nil || l.items[l.selectedIndex] == nil {
		l.selectedIndex = 0
	}

	for i, item := range l.items {
		item.selected = (l.selectedIndex == i)
		item.index = i

		list.WriteString(item.String())
		list.WriteByte('\n')
	}

	l.length = len(l.items)

	return list.String()
}

func (l *BufferList) createListItem(buf *Buffer, index int) BufferListItem {
	item := BufferListItem{
		Item: Item{
			index:     index,
			name:      buf.Name(),
			path:      buf.Path(false),
			selected:  index == l.selectedIndex,
			nerdFonts: l.conf.NerdFonts(),
			width:     l.width,
		},
		buffer: buf,
	}

	return item
}

func (l *BufferList) Items() []*BufferListItem {
	return l.items
}

func (l *BufferList) CancelAction(cb func()) message.StatusBarMsg {
	l.SetFocus(false)
	l.SetSelectedIndex(0)

	return message.StatusBarMsg{}
}

func (l *BufferList) NeedsUpdate() bool {
	if len(l.items) != len(*l.Buffers) {
		l.buildItems()
		return true
	}

	found := 0

	for _, buf := range *l.Buffers {
		for _, item := range l.items {
			if buf.path == item.path {
				found++
			}
		}
	}

	if found != len(l.items) {
		l.buildItems()
		return true
	}

	return false
}

func (l *BufferList) RefreshStyles() {
	l.viewport.Style = theme.BaseColumnLayout(
		l.Size,
		l.Focused(),
	)
	l.BuildHeader(l.Size.Width, true)
}
