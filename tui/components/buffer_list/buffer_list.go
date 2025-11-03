package bufferlist

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
	"bellbird-notes/tui/components/editor"
	"bellbird-notes/tui/components/overlay"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
)

type BufferListItem struct {
	shared.Item

	buffer *editor.Buffer
}

// PathOnly returns the relative path of a BufferListItem wihout the filename
func (b BufferListItem) PathOnly() string {
	p := path.Dir(b.Path())

	notesRoot, _ := app.NotesRootDir()

	if p == notesRoot {
		return "/"
	}

	return p
}

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

	if b.IsSelected {
		style = style.Background(theme.ColourBgSelected)
	}

	return style.Render(content)
}

// String is string representation of a Note
func (b BufferListItem) String() string {
	index := b.render(strconv.Itoa(b.Index()+1), false, false, 0, 2, 2)
	name := b.render(b.Name(), false, false, 0, 0, 2)

	icon := theme.Icon(theme.IconNote, b.NerdFonts)
	if b.buffer.Dirty {
		icon = theme.Icon(theme.IconDot, b.NerdFonts)
	}
	iconRender := b.render(icon, false, b.buffer.Dirty, 3, 0, 0)

	pathWidth := b.Width() - lipgloss.Width(index) - lipgloss.Width(iconRender) - lipgloss.Width(name)
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
	shared.List[*BufferListItem]

	width  int
	height int

	// Buffers holds all the open buffers
	Buffers *editor.Buffers

	Overlay *overlay.Overlay
}

func New(conf *config.Config) *BufferList {
	termW, _ := theme.TerminalSize()

	var list shared.List[*BufferListItem]
	list.MakeEmpty()
	list.Title = "OPEN NOTES"
	list.Conf = conf

	panel := &BufferList{
		List:    list,
		height:  10,
		width:   termW / 3,
		Overlay: &overlay.Overlay{},
	}
	panel.SetTheme(theme.New(conf))
	panel.Blur()
	panel.Mode = mode.Normal

	return panel
}

func (l BufferList) Name() string { return "BufferList" }

func (l BufferList) Width() int {
	return l.Viewport.Width()
}

func (l BufferList) ListSize() (int, int) {
	w, _ := theme.TerminalSize()
	return w / 3, 10
}

func (l *BufferList) UpdateSize() {
	w, h := l.ListSize()
	l.Viewport.SetWidth(w)
	l.Viewport.SetHeight(h)
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
			if err == nil && num-1 < len(l.Items) {
				path := l.Items[num-1].Path()
				cmds = append(cmds, editor.SendSwitchBufferMsg(path, true))
			}
		}

	case tea.WindowSizeMsg:
		l.UpdateSize()

	case editor.BuffersChangedMsg:
		l.buildItems()
	}

	if l.Focused() {
		if !l.IsReady {
			l.Viewport = viewport.New()
			l.Viewport.SetContent(l.render())
			l.Viewport.KeyMap = viewport.KeyMap{}
			l.IsReady = true
		}

		l.updateOverlay()

		var cmd tea.Cmd
		l.Viewport, cmd = l.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return l, tea.Batch(cmds...)
}

func (l *BufferList) buildItems() {
	buffers := *l.Buffers
	l.Items = make([]*BufferListItem, 0, len(buffers))

	for i := range buffers {
		item := l.createListItem(&buffers[i], i)
		l.Items = append(l.Items, &item)
	}
}

func (l *BufferList) View() tea.View {
	var view tea.View
	view.SetContent(l.Content())
	return view
}

func (l *BufferList) Content() string {
	if !l.IsReady {
		return "\n  Initializing..."
	}

	l.Viewport.SetContent(l.render())
	l.UpdateViewportInfo()

	l.Viewport.Style = l.Theme().BaseColumnLayout(
		l.Size,
		l.IsReady,
	)

	var view strings.Builder
	view.WriteString(l.BuildHeader(l.width, false))
	view.WriteString(l.Viewport.View())

	if len(l.Items) > 0 {
		l.LastIndex = l.Items[len(l.Items)-1].Index()
	}

	return view.String()
}

func (l *BufferList) SetBuffers(b *editor.Buffers) {
	l.Buffers = b
}

func (l *BufferList) RefreshSize() {
	vp := l.Viewport
	if vp.Width() != l.width && vp.Height() != l.height {
		l.Viewport.SetWidth(l.width)
		l.Viewport.SetHeight(l.height)
	}
}

func (l *BufferList) render() string {
	var list strings.Builder

	if l.Items == nil {
		l.SelectedIndex = 0
	}

	for i, item := range l.Items {
		item.IsSelected = l.SelectedIndex == i
		item.SetIndex(i)

		list.WriteString(item.String())
		list.WriteByte('\n')
	}

	l.Length = len(l.Items)

	return list.String()
}

func (l *BufferList) createListItem(buf *editor.Buffer, index int) BufferListItem {
	var item shared.Item
	item.SetIndex(index)
	item.SetName(buf.Name())
	item.SetPath(buf.Path(false))
	item.SetWidth(l.width)
	item.IsSelected = index == l.SelectedIndex
	item.NerdFonts = l.Conf.NerdFonts()

	listItem := BufferListItem{
		Item:   item,
		buffer: buf,
	}

	return listItem
}

func (l *BufferList) SelectedBuffer() *BufferListItem {
	var (
		items        = l.Items
		selectedItem = l.SelectedItem(items)
		lastBuffer   = 0
	)

	if len(items) <= 0 {
		return nil
	}

	lastBuffer = items[len(items)-1].Index() - 1

	if l.SelectedIndex > lastBuffer {
		l.SelectedIndex = lastBuffer
	}

	return selectedItem
}

func (l *BufferList) CancelAction(cb func()) message.StatusBarMsg {
	l.Blur()
	l.SelectedIndex = 0

	return message.StatusBarMsg{}
}

func (l *BufferList) NeedsUpdate() bool {
	if len(l.Items) != len(*l.Buffers) {
		l.buildItems()
		return true
	}

	found := 0

	for _, buf := range *l.Buffers {
		for _, item := range l.Items {
			if buf.Path(false) == item.Path() {
				found++
			}
		}
	}

	if found != len(l.Items) {
		l.buildItems()
		return true
	}

	return false
}

func (l *BufferList) RefreshStyles() {
	l.Viewport.Style = l.Theme().BaseColumnLayout(
		l.Size,
		l.Focused(),
	)
	l.BuildHeader(l.Size.Width, true)
}

func (l BufferList) updateOverlay() {
	x, y := l.overlayPosition()

	l.IsReady = true
	l.Focus()

	l.Overlay.SetPosition(x, y)
	l.Overlay.SetContent(l.Content())
}

func (l *BufferList) overlayPosition() (int, int) {
	termW, _ := theme.TerminalSize()

	x := (termW / 2) - (l.Width() / 2)
	y := 2

	return x, y
}

func (l *BufferList) ConfirmAction() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (l *BufferList) PasteSelectedItems() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (l *BufferList) TogglePinnedItems() message.StatusBarMsg {
	return message.StatusBarMsg{}
}
