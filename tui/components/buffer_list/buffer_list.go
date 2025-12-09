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
func (item BufferListItem) PathOnly() string {
	p := path.Dir(item.Path())
	notesRoot, _ := app.NotesRootDir()

	if p == notesRoot {
		return "/"
	}

	return p
}

func (item BufferListItem) render(
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

	if item.IsSelected {
		style = style.Background(theme.ColourBgSelected)
	}

	return style.Render(content)
}

// String is string representation of a Note
func (item BufferListItem) String() string {
	index := item.render(strconv.Itoa(item.Index()+1), false, false, 0, 2, 2)
	name := item.render(item.Name(), false, false, 0, 0, 2)

	icon := theme.Icon(theme.IconNote, item.NerdFonts)
	if item.buffer.Dirty {
		icon = theme.Icon(theme.IconDot, item.NerdFonts)
	}
	iconRender := item.render(icon, false, item.buffer.Dirty, 3, 0, 0)

	pathWidth := item.Width() - lipgloss.Width(index) - lipgloss.Width(iconRender) - lipgloss.Width(name)
	ellipsisWidth := 4
	path := utils.TruncateText(
		utils.RelativePath(item.PathOnly(), true),
		pathWidth-ellipsisWidth,
	)
	path = item.render(path, true, false, pathWidth, 0, 0)

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

func New(title string, conf *config.Config) *BufferList {
	termW, _ := theme.TerminalSize()

	var list shared.List[*BufferListItem]
	list.MakeEmpty()
	list.Conf = conf

	panel := &BufferList{
		List:    list,
		height:  10,
		width:   termW / 3,
		Overlay: &overlay.Overlay{},
	}

	panel.SetTitle(title)
	panel.SetTheme(theme.New(conf))
	panel.Blur()
	panel.Mode = mode.Normal

	return panel
}

func (list BufferList) Width() int {
	return list.Viewport.Width()
}

func (list BufferList) ListSize() (int, int) {
	w, _ := theme.TerminalSize()
	return w / 3, 10
}

func (list *BufferList) UpdateSize() {
	w, h := list.ListSize()
	list.Viewport.SetWidth(w)
	list.Viewport.SetHeight(h)
	list.width = w
	list.height = h
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (list *BufferList) Init() tea.Cmd {
	return nil
}

func (list *BufferList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if list.Focused() {
			// check if input is a numeric value
			num, err := strconv.Atoi(msg.String())

			// switch to buffer if existent
			if err == nil && num-1 < len(list.Items) {
				path := list.Items[num-1].Path()
				cmds = append(cmds, editor.SendSwitchBufferMsg(path, true))
			}
		}

	case tea.WindowSizeMsg:
		list.UpdateSize()

	case editor.BuffersChangedMsg:
		list.buildItems()
	}

	if list.Focused() {
		if !list.IsReady {
			list.Viewport = viewport.New()
			list.Viewport.SetContent(list.render())
			list.Viewport.KeyMap = viewport.KeyMap{}
			list.IsReady = true
		}

		list.updateOverlay()

		var cmd tea.Cmd
		list.Viewport, cmd = list.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return list, tea.Batch(cmds...)
}

func (list *BufferList) buildItems() {
	buffers := *list.Buffers
	list.Items = make([]*BufferListItem, 0, len(buffers))

	for i := range buffers {
		item := list.createListItem(&buffers[i], i)
		list.Items = append(list.Items, &item)
	}
}

func (list *BufferList) View() tea.View {
	var view tea.View
	view.SetContent(list.Content())
	return view
}

func (list *BufferList) Content() string {
	if !list.IsReady {
		return "\n  Initializing..."
	}

	list.Viewport.SetContent(list.render())
	list.UpdateViewportInfo()

	list.Viewport.Style = list.Theme().BaseColumnLayout(
		list.Size,
		list.IsReady,
	)

	var view strings.Builder
	view.WriteString(list.BuildHeader(list.width, false))
	view.WriteString(list.Viewport.View())

	if len(list.Items) > 0 {
		list.LastIndex = list.Items[len(list.Items)-1].Index()
	}

	return view.String()
}

func (list *BufferList) SetBuffers(b *editor.Buffers) {
	list.Buffers = b
}

func (list *BufferList) RefreshSize() {
	vp := list.Viewport
	if vp.Width() != list.width && vp.Height() != list.height {
		list.Viewport.SetWidth(list.width)
		list.Viewport.SetHeight(list.height)
	}
}

func (bl *BufferList) render() string {
	var list strings.Builder

	if bl.Items == nil {
		bl.SelectedIndex = 0
	}

	for i, item := range bl.Items {
		item.IsSelected = bl.SelectedIndex == i
		item.SetIndex(i)

		list.WriteString(item.String())
		list.WriteByte('\n')
	}

	bl.Length = len(bl.Items)

	return list.String()
}

func (list *BufferList) createListItem(buf *editor.Buffer, index int) BufferListItem {
	var item shared.Item
	item.SetIndex(index)
	item.SetName(buf.Name())
	item.SetPath(buf.Path(false))
	item.SetWidth(list.width)
	item.IsSelected = index == list.SelectedIndex
	item.NerdFonts = list.Conf.NerdFonts()

	listItem := BufferListItem{
		Item:   item,
		buffer: buf,
	}

	return listItem
}

func (list *BufferList) SelectedBuffer() *BufferListItem {
	var (
		items        = list.Items
		selectedItem = list.SelectedItem(items)
		lastBuffer   = 0
	)

	if len(items) <= 0 {
		return nil
	}

	lastBuffer = items[len(items)-1].Index() - 1

	if list.SelectedIndex > lastBuffer {
		list.SelectedIndex = lastBuffer
	}

	return selectedItem
}

func (list *BufferList) CancelAction(cb func()) message.StatusBarMsg {
	list.Blur()
	list.SelectedIndex = 0

	return message.StatusBarMsg{}
}

func (list *BufferList) NeedsUpdate() bool {
	if len(list.Items) != len(*list.Buffers) {
		list.buildItems()
		return true
	}

	found := 0

	for _, buf := range *list.Buffers {
		for _, item := range list.Items {
			if buf.Path(false) == item.Path() {
				found++
			}
		}
	}

	if found != len(list.Items) {
		list.buildItems()
		return true
	}

	return false
}

func (list *BufferList) RefreshStyles() {
	list.Viewport.Style = list.Theme().BaseColumnLayout(
		list.Size,
		list.Focused(),
	)
	list.BuildHeader(list.Size.Width, true)
}

func (list BufferList) updateOverlay() {
	x, y := list.overlayPosition()

	list.IsReady = true
	list.Focus()

	list.Overlay.SetPosition(x, y)
	list.Overlay.SetContent(list.Content())
}

func (list *BufferList) overlayPosition() (int, int) {
	termW, _ := theme.TerminalSize()

	x := (termW / 2) - (list.Width() / 2)
	y := 2

	return x, y
}

func (list *BufferList) ConfirmAction() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (list *BufferList) PasteSelectedItems() message.StatusBarMsg {
	return message.StatusBarMsg{}
}

func (list *BufferList) TogglePinnedItems() message.StatusBarMsg {
	return message.StatusBarMsg{}
}
