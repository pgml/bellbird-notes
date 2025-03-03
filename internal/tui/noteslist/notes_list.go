package noteslist

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/notes"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/theme"
	"bellbird-notes/internal/tui/utils"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bl "github.com/winder/bubblelayout"
)

type NotesList struct {
	Id   bl.ID
	Size bl.Size

	// The current mode the directory tree is in
	// Possible modes are Normal, Insert, Command
	Mode mode.Mode

	// Indicates hether the directory tree column is focused.
	// Used to determine if the notes list should receive keyboard shortcuts
	Focused bool

	selectedIndex int // The currently selector note

	editor       textinput.Model // The text input that is used for renaming or creating notes
	editingIndex *int            // The index of the currently edited note
	editingState EditState       // States if a note is being created or renamed

	// The directory path of the currently displayed notes.
	// This path might not match the directory that is selected in the
	// directory tree since we don't automatically display a directory's
	// content on a selection change
	CurrentPath string
	notes       []Note // Stores the notes of a directory

	statusMessage string         // For displaying useful information in the status bar
	viewport      viewport.Model // The tree viewport that allows scrolling
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

type Note struct {
	index    int
	name     string
	path     string
	selected bool
	isPinned bool
	styles   styles
}

func (n Note) String() string {
	r := n.styles.note.Render
	name := utils.TruncateText(n.name, 22)

	icon := " 󰎞"
	noNerdFonts := false
	if noNerdFonts {
		icon = " "
	}

	baseStyle := lipgloss.NewStyle().Width(30)
	if n.selected {
		baseStyle = baseStyle.Background(lipgloss.Color("#424B5D")).Bold(true)
	}
	return baseStyle.Render(icon + r(name))
	//return baseStyle.Render(icon + r(name+" "+strconv.Itoa(n.index)))
}

func (n Note) GetName() string {
	return n.name
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (l *NotesList) Init() tea.Cmd {
	return nil
}

func (l *NotesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd tea.Cmd
		//cmds []tea.Cmd
	)
	_, termHeight := theme.GetTerminalSize()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if l.editingIndex != nil && !l.editor.Focused() {
			l.editor.Focus()
			return l, nil
		}

		if l.editor.Focused() {
			l.editor.Focus()
			l.editor, cmd = l.editor.Update(msg)
			return l, cmd
		}
	case tea.WindowSizeMsg:
		if !l.ready {
			l.viewport = viewport.New(30, termHeight-1)
			l.viewport.SetContent(l.buildList())
			l.viewport.KeyMap = viewport.KeyMap{}
			l.lastVisibleLine = l.viewport.VisibleLineCount() - 3
			l.ready = true
		} else {
			l.viewport.Width = 30
			l.viewport.Height = termHeight - 1
		}
	}

	// Handle keyboard and mouse events in the viewport
	l.viewport, cmd = l.viewport.Update(msg)
	//cmds = append(cmds, cmd)

	return l, cmd
}

func (l *NotesList) View() string {
	if !l.ready {
		return "\n  Initializing..."
	}

	l.viewport.SetContent(l.buildList())

	if l.visibleLineCount != l.viewport.VisibleLineCount() {
		l.visibleLineCount = l.viewport.VisibleLineCount() - 3
		l.lastVisibleLine = l.visibleLineCount
	}

	l.viewport.Style = theme.BaseColumnLayout(l.Size, l.Focused)
	return l.viewport.View()
}

func New() *NotesList {
	ti := textinput.New()
	ti.Prompt = "  "
	ti.CharLimit = 100

	conf := config.New()
	list := &NotesList{
		selectedIndex:    0,
		editingIndex:     nil,
		editingState:     EditNone,
		editor:           ti,
		notes:            make([]Note, 0, 0),
		CurrentPath:      conf.Value(config.General, config.UserNotesDirectory),
		lastVisibleLine:  0,
		firstVisibleLine: 0,
	}

	list.Refresh()
	return list
}

func (l NotesList) buildList() string {
	var list string

	for i, note := range l.notes {
		note.selected = (l.selectedIndex == i)

		//style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
		//tree += style.Render(fmt.Sprintf("%02d", dir.index)) + " "
		if l.editingIndex != nil && i == *l.editingIndex {
			list += l.editor.View() + "\n" // Show input field instead of text
		} else {
			list += fmt.Sprintf("%-*s \n", l.viewport.Width, note.String())
		}
	}

	return list
}

func (l *NotesList) Refresh() messages.StatusBarMsg {
	l.selectedIndex = 0
	notes, err := notes.List(l.CurrentPath)

	if err != nil {
		return messages.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    messages.Error,
		}
	}

	l.notes = make([]Note, 0, len(notes))

	for i, note := range notes {
		noteItem := l.createNoteItem(note)
		noteItem.index = i
		l.notes = append(l.notes, noteItem)
	}

	return messages.StatusBarMsg{}
}

func (l *NotesList) createNoteItem(note notes.Note) Note {
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

// createVirtualDir creates a temporary, virtual directory `Dir`
//
// This directory is mainly used as a placeholder when creating a directory
func (l *NotesList) createVirtualDir() Note {
	selectedNote := l.SelectedNote()
	tempNoteName := "New Note"
	tempNotePath := filepath.Join(filepath.Dir(selectedNote.path), tempNoteName)

	return Note{
		index: len(l.notes),
		name:  tempNoteName,
		path:  tempNotePath,
	}
}

// Returns the currently selected note
// or an empty note if there's nothing to select
func (l *NotesList) SelectedNote() Note {
	if len(l.notes) == 0 {
		return Note{}
	}
	if l.selectedIndex >= 0 && l.selectedIndex < len(l.notes) {
		return l.notes[l.selectedIndex]
	}
	return Note{}
}

func (l NotesList) getLastChild() Note {
	if len(l.notes) <= 0 {
		return Note{}
	}
	return l.notes[len(l.notes)-1]
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// of the directories.
// To make it persistent write it to the file system
func (l *NotesList) insertDirAfter(afterIndex int, note Note) {
	for i, dir := range l.notes {
		if dir.index == afterIndex {
			l.notes = append(
				l.notes[:i+1],
				append([]Note{note}, l.notes[i+1:]...)...,
			)
			break
		}
	}
}

func (l *NotesList) noteExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		app.LogErr(err)
		return false
	}
	return true
}

///
/// keyboard shortcut commands
///

// Decrements `m.selectedIndex`
func (l *NotesList) LineUp() messages.StatusBarMsg {
	if l.selectedIndex > 0 {
		l.selectedIndex--
	}

	// scroll up
	if l.selectedIndex < l.firstVisibleLine {
		l.firstVisibleLine = l.selectedIndex
		l.lastVisibleLine = l.visibleLineCount + l.firstVisibleLine
		l.viewport.LineUp(1)
	}

	return messages.StatusBarMsg{
		Content: l.SelectedNote().name,
	}
}

// Increments `m.selectedIndex`
func (l *NotesList) LineDown() messages.StatusBarMsg {
	if l.selectedIndex < len(l.notes)-1 {
		l.selectedIndex++
	}

	// scroll down
	if l.selectedIndex > l.visibleLineCount {
		l.firstVisibleLine = l.selectedIndex - l.visibleLineCount
		l.lastVisibleLine = l.selectedIndex
		l.viewport.LineDown(1)
	}

	return messages.StatusBarMsg{
		Content: l.SelectedNote().name,
	}
}

// Create creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (l *NotesList) Create() messages.StatusBarMsg {
	l.editingState = EditCreate
	tmpdir := l.createVirtualDir()
	lastChild := l.getLastChild()
	l.insertDirAfter(lastChild.index, tmpdir)
	l.selectedIndex = lastChild.index + 1

	if l.editingIndex == nil {
		l.editingIndex = &l.selectedIndex
		l.editor.SetValue(l.SelectedNote().name)
	}
	return messages.StatusBarMsg{}
}

// Rename renames the currently selected directory and
// returns a message that is displayed in the status bar
func (l *NotesList) Rename() messages.StatusBarMsg {
	if l.editingIndex == nil {
		l.editingState = EditRename
		l.editingIndex = &l.selectedIndex
		l.editor.SetValue(l.SelectedNote().name)
		// set cursor to last position
		l.editor.CursorEnd()
	}
	return messages.StatusBarMsg{}
}

func (l *NotesList) GoToTop() messages.StatusBarMsg {
	l.selectedIndex = 0
	l.viewport.GotoTop()
	return messages.StatusBarMsg{}
}

func (l *NotesList) GoToBottom() messages.StatusBarMsg {
	l.selectedIndex = l.notes[len(l.notes)-1].index
	l.viewport.GotoBottom()
	return messages.StatusBarMsg{}
}

func (l *NotesList) ConfirmRemove() messages.StatusBarMsg {
	selectedNote := l.SelectedNote()
	msgType := messages.PromptError
	resultMsg := fmt.Sprintf(messages.RemovePrompt, selectedNote.path)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  messages.SenderNotesList,
	}
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (l *NotesList) Remove() messages.StatusBarMsg {
	note := l.SelectedNote()
	index := l.selectedIndex
	resultMsg := fmt.Sprintf(messages.SuccessRemove, note.path)
	msgType := messages.Success

	if err := notes.Delete(note.path); err == nil {
		l.notes = slices.Delete(l.notes, index, index+1)
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	l.Refresh()
	//l.buildList()
	return messages.StatusBarMsg{Content: resultMsg, Type: msgType}
}

// Confirms a user action
func (l *NotesList) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if l.editingIndex != nil {
		selectedNote := l.SelectedNote()
		oldPath := selectedNote.path
		newPath := filepath.Join(filepath.Dir(oldPath), l.editor.Value())

		switch l.editingState {
		case EditRename:
			// rename if path exists
			if _, err := os.Stat(oldPath); err == nil {
				notes.Rename(oldPath, newPath)
				selectedNote.name = filepath.Base(newPath)
				selectedNote.path = newPath
			}

		case EditCreate:
			if !l.noteExists(newPath) {
				notes.Create(newPath)
			}
		}

		l.CancelAction()
		return messages.StatusBarMsg{Content: "yep", Sender: messages.SenderNotesList}
	}

	l.Refresh()

	return messages.StatusBarMsg{Sender: messages.SenderNotesList}
}

// Cancel the current action and blurs the editor
func (l *NotesList) CancelAction() messages.StatusBarMsg {
	if l.editingState != EditNone {
		l.editingIndex = nil
		l.editingState = EditNone
		l.editor.Blur()
	}
	l.Refresh()
	return messages.StatusBarMsg{}
}
