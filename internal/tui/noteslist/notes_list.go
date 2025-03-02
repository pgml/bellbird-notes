package noteslist

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/notes"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	bl "github.com/winder/bubblelayout"
)

type NotesList struct {
	Id        bl.ID
	Size      bl.Size
	IsFocused bool
	Mode      mode.Mode

	editor        textinput.Model
	editingIndex  *int
	editingState  EditState
	selectedIndex int

	CurrentPath string
	notes       []Note
	content     *list.List

	statusMessage string
	viewport      viewport.Model
	ready         bool

	firstVisibleLine int
	lastVisibleLine  int
	visibleLineCount int
}

type EditState int

const (
	EditNone EditState = iota
	EditCreate
	EditRename
)

type statusMsg string

type styles struct {
	base,
	enumerator,
	note,
	toggle lipgloss.Style
}

func defaultStyles() styles {
	var s styles
	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		MarginLeft(0).
		PaddingLeft(1)
	s.note = s.base.
		MarginRight(0).
		PaddingLeft(2).
		PaddingRight(2).
		Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#eee"})
	return s
}

type Note struct {
	index    int
	name     string
	path     string
	selected bool
	isPinned bool
	styles   styles
}

func (d Note) String() string {
	n := d.styles.note.Render
	name := theme.TruncateText(d.name, 22)

	icon := " 󰎞"
	noNerdFonts := false
	if noNerdFonts {
		icon = " "
	}
	baseStyle := lipgloss.NewStyle().Width(40)

	if d.selected {
		baseStyle = baseStyle.Background(lipgloss.Color("#424B5D")).Bold(true)
	}

	return baseStyle.Render(icon + n(name))
	//return baseStyle.Render(icon + n(name+" "+strconv.Itoa(d.index)))
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (n *NotesList) Init() tea.Cmd {
	//conf := config.New()
	//n.CurrentPath = conf.Value(config.General, config.UserNotesDirectory)
	return nil
}

func (n *NotesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	_, termHeight := theme.GetTerminalSize()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if n.editingIndex != nil && !n.editor.Focused() {
			n.editor.Focus()
			return n, nil
		}

		if n.editor.Focused() {
			n.editor.Focus()
			n.editor, cmd = n.editor.Update(msg)
			return n, cmd
		}
	case tea.WindowSizeMsg:
		if !n.ready {
			n.viewport = viewport.New(30, termHeight-1)
			n.viewport.SetContent(n.renderList())
			n.viewport.KeyMap = viewport.KeyMap{}
			n.lastVisibleLine = n.viewport.VisibleLineCount() - 3
			n.ready = true
		} else {
			n.viewport.Width = 30
			n.viewport.Height = termHeight
		}
	}

	//app.LogDebug(n.firstVisibleLine, n.lastVisibleLine, n.selectedIndex, n.viewport.VisibleLineCount())

	//app.LogDebug(n.viewport.View())
	// Handle keyboard and mouse events in the viewport
	n.viewport, cmd = n.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return n, cmd
}

func (n *NotesList) View() string {
	if !n.ready {
		return "\n  Initializing..."
	}

	n.viewport.SetContent(n.renderList())

	if n.visibleLineCount != n.viewport.VisibleLineCount() {
		n.visibleLineCount = n.viewport.VisibleLineCount() - 3
		n.lastVisibleLine = n.visibleLineCount
	}

	var borderColour lipgloss.TerminalColor = lipgloss.Color("#424B5D")
	if n.IsFocused {
		borderColour = lipgloss.Color("#69c8dc")
	}

	_, termHeight := theme.GetTerminalSize()

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColour).
		Foreground(lipgloss.NoColor{}).
		Height(termHeight - 1)

	n.viewport.Style = style
	return n.viewport.View()
}

func New() *NotesList {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	conf := config.New()
	list := &NotesList{
		selectedIndex:    0,
		editingIndex:     nil,
		editingState:     EditNone,
		editor:           ti,
		notes:            nil,
		CurrentPath:      conf.Value(config.General, config.UserNotesDirectory),
		lastVisibleLine:  0,
		firstVisibleLine: 0,
	}

	list.Refresh()
	return list
}

func (n *NotesList) renderList() string {
	var list string

	for i, note := range n.notes {
		//if dir.index == 0 && dir.parent == 0 {
		//	n.dirsListFlat = slices.Delete(n.dirsListFlat, i, i+1)
		//	continue
		//}

		note.selected = (n.selectedIndex == i)

		//style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
		//tree += style.Render(fmt.Sprintf("%02d", dir.index)) + " "
		if n.editingIndex != nil && i == *n.editingIndex {
			list += n.editor.View() + "\n" // Show input field instead of text
		} else {
			note.styles.base.Background(lipgloss.Color("#424B5D")).Bold(true)
			list += note.String() + "\n"
		}
	}

	return list
}

func (n *NotesList) Refresh() messages.StatusBarMsg {
	n.notes = nil
	n.selectedIndex = 0
	notes, _ := notes.List(n.CurrentPath)

	for i, note := range notes {
		noteItem := n.createNoteItem(note)
		noteItem.index = i
		//app.LogDebug(noteItem.name)
		n.notes = append(n.notes, noteItem)
	}

	return messages.StatusBarMsg{}
}

// Creates a dir
func (n *NotesList) createNoteItem(note notes.Note) Note {
	style := defaultStyles()
	childItem := Note{
		index:    0,
		name:     note.Name,
		path:     note.Path,
		isPinned: note.IsPinned,
		styles:   style,
	}
	return childItem
}

// Returns the currently selected note
// or the first if there's no selected for some reaon
func (n *NotesList) SelectedNote() Note {
	for i := range n.notes {
		dir := n.notes[i]
		if i == n.selectedIndex {
			return dir
		}
	}
	return n.notes[0]
}

// Decrements `m.selectedIndex`
func (n *NotesList) LineUp() messages.StatusBarMsg {
	if n.selectedIndex > 0 {
		n.selectedIndex--
	}

	// scroll up
	if n.selectedIndex < n.firstVisibleLine {
		n.firstVisibleLine = n.selectedIndex
		n.lastVisibleLine = n.visibleLineCount + n.firstVisibleLine
		n.viewport.LineUp(1)
	}

	return messages.StatusBarMsg{
		Content: n.SelectedNote().name,
	}
}

// Increments `m.selectedIndex`
func (n *NotesList) LineDown() messages.StatusBarMsg {
	if n.selectedIndex < len(n.notes)-1 {
		n.selectedIndex++
	}

	// scroll down
	if n.selectedIndex > n.visibleLineCount {
		n.firstVisibleLine = n.selectedIndex - n.visibleLineCount
		n.lastVisibleLine = n.selectedIndex
		n.viewport.LineDown(1)
	}

	return messages.StatusBarMsg{
		Content: n.SelectedNote().name,
	}
}

func (n *NotesList) GoToTop() messages.StatusBarMsg {
	n.selectedIndex = 0
	n.viewport.GotoTop()
	app.LogDebug(n.selectedIndex)
	return messages.StatusBarMsg{}
}

func (n *NotesList) GoToBottom() messages.StatusBarMsg {
	n.selectedIndex = n.notes[len(n.notes)-1].index
	n.viewport.GotoBottom()
	app.LogDebug(n.selectedIndex)
	return messages.StatusBarMsg{}
}
