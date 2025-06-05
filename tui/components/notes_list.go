package components

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	statusbarcolumn "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotesList struct {
	List[NoteItem]

	// The directory path of the currently displayed notes.
	// This path might not match the directory that is selected in the
	// directory tree since we don't automatically display a directory's
	// content on a selection change
	CurrentPath string
}

type NoteItem struct {
	Item
	isPinned bool
}

// Index returns the index of a Note-Item
func (n NoteItem) Index() int { return n.index }

// Path() returns the index of a Note-Item
func (n NoteItem) Path() string { return n.path }

// Name() returns the index of a Note-Item
func (n NoteItem) Name() string { return n.name }

// The string representation of a Dir
func (n NoteItem) String() string {
	r := n.styles.note.Render
	name := utils.TruncateText(n.Name(), 22)
	name = strings.TrimSuffix(
		name,
		filepath.Ext(name),
	)

	// nerdfonts required
	icon := " 󰎞"
	if *app.NoNerdFonts {
		icon = " "
	}

	baseStyle := lipgloss.NewStyle().Width(28)
	if n.selected {
		baseStyle = baseStyle.
			Background(theme.ColourBgSelected).
			Bold(true)
	}

	return baseStyle.Render(icon + r(name))
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (l *NotesList) Init() tea.Cmd {
	return nil
}

func (l *NotesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	termWidth, termHeight := theme.GetTerminalSize()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if l.editIndex != nil && !l.editor.Focused() {
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
			l.viewport = viewport.New(
				termWidth,
				termHeight,
			)
			l.viewport.SetContent(l.build())
			l.viewport.KeyMap = viewport.KeyMap{}
			l.lastVisibleLine = l.viewport.VisibleLineCount() - reservedLines
			l.ready = true
		} else {
			l.viewport.Width = termWidth
			l.viewport.Height = termHeight
		}
	}

	// Handle keyboard and mouse events in the viewport
	l.viewport, cmd = l.viewport.Update(msg)

	return l, cmd
}

func (l *NotesList) View() string {
	if !l.ready {
		return "\n  Initializing..."
	}

	l.viewport.SetContent(l.build())
	l.UpdateViewportInfo()

	l.viewport.Style = theme.BaseColumnLayout(
		l.Size,
		l.Focused,
	)

	return l.viewport.View()
}

// NewNotesList creates a new model with default settings.
func NewNotesList() *NotesList {
	ti := textinput.New()
	ti.Prompt = " " + theme.IconInput + " "
	ti.CharLimit = 100

	conf := config.New()
	list := &NotesList{
		List: List[NoteItem]{
			selectedIndex:    0,
			editIndex:        nil,
			EditState:        EditStates.None,
			editor:           ti,
			lastVisibleLine:  0,
			firstVisibleLine: 0,
			items:            make([]NoteItem, 0),
		},
		CurrentPath: conf.Value(config.General, config.UserNotesDirectory),
	}

	list.Refresh(false)
	return list
}

// build prepares the notes list as a string
func (l NotesList) build() string {
	var list string

	for i, note := range l.items {
		note.selected = (l.selectedIndex == i)

		if *app.Debug {
			// prepend list item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			list += style.Render(fmt.Sprintf("%02d", note.index)) + " "
		}

		if l.editIndex != nil && i == *l.editIndex {
			// Show input field instead of text
			list += l.editor.View() + "\n"
		} else {
			list += fmt.Sprintf("%-*s \n", l.viewport.Width, note.String())
		}
	}

	return list
}

// Refresh updates the notes list
//
// If `resetIndex` is set to true, 'l.selectedIndex' will be set to 0
// which representns the first note
func (l *NotesList) Refresh(resetSelectedIndex bool) message.StatusBarMsg {
	if resetSelectedIndex {
		l.selectedIndex = 0
	}

	notes, err := notes.List(l.CurrentPath)

	if err != nil {
		return message.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    message.Error,
		}
	}

	l.items = make([]NoteItem, 0, len(notes))

	for i, note := range notes {
		noteItem := l.createNoteItem(note)
		noteItem.index = i
		l.items = append(l.items, noteItem)
	}

	l.length = len(l.items)
	l.lastIndex = 0

	if l.length > 0 {
		l.lastIndex = l.items[len(l.items)-1].index
	}

	return message.StatusBarMsg{}
}

// createNoteItem creates populated NoteItem
func (l *NotesList) createNoteItem(note notes.Note) NoteItem {
	style := NotesListStyle()
	childItem := NoteItem{
		Item: Item{
			index:  0,
			name:   note.Name,
			path:   note.Path,
			styles: style,
		},
		isPinned: note.IsPinned,
	}
	return childItem
}

// createVirtualNote creates a virtual note `Note`
// with dummy data
//
// This note is mainly used as a placeholder when creating a note
// and is not actually written to the file system.
func (l *NotesList) createVirtualNote() NoteItem {
	selectedNote := l.SelectedItem(nil)
	name := "New Note"
	path := filepath.Join(
		filepath.Dir(selectedNote.path),
		name,
	)

	item := notes.Note{
		Name: name,
		Path: path,
	}

	noteItem := l.createNoteItem(item)
	noteItem.index = len(l.items)

	return noteItem
}

func (l NotesList) getLastChild() NoteItem {
	if len(l.items) <= 0 {
		return NoteItem{}
	}
	return l.items[len(l.items)-1]
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// l.items. To make it persistent write it to the file system
func (l *NotesList) insertNoteAfter(afterIndex int, note NoteItem) {
	for i, dir := range l.items {
		if dir.index == afterIndex {
			l.items = append(
				l.items[:i+1],
				append([]NoteItem{note}, l.items[i+1:]...)...,
			)
			break
		}
	}
}

// Create creates a note after the last child
func (l *NotesList) Create(
	mi *mode.ModeInstance,
	statusBar *StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if l.Focused {
		mi.Current = mode.Insert
		statusBar.Focused = false

		l.EditState = EditStates.Create

		vrtNote := l.createVirtualNote()
		lastChild := l.getLastChild()

		if lastChild.name == "" {
			l.items = append(l.items, vrtNote)
		} else {
			l.insertNoteAfter(lastChild.index, vrtNote)
			l.selectedIndex = lastChild.index + 1
		}

		if l.editIndex == nil {
			selItem := l.SelectedItem(nil)
			l.editIndex = &l.selectedIndex
			l.editor.SetValue(selItem.name)
			l.editor.CursorEnd()
		}
	}

	return statusMsg
}

func (l *NotesList) ConfirmRemove() message.StatusBarMsg {
	selectedNote := l.SelectedItem(nil)
	msgType := message.PromptError

	rootDir, _ := app.NotesRootDir()
	path := strings.ReplaceAll(selectedNote.path, rootDir, ".")
	resultMsg := fmt.Sprintf(message.StatusBar.RemovePrompt, path)

	l.EditState = EditStates.Delete

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  message.SenderNotesList,
		Column:  statusbarcolumn.General,
	}
}

// Remove deletes the selected note from the file system
func (l *NotesList) Remove() message.StatusBarMsg {
	note := l.SelectedItem(nil)
	index := l.selectedIndex
	resultMsg := ""
	msgType := message.Success

	if err := notes.Delete(note.path); err == nil {
		l.items = slices.Delete(l.items, index, index+1)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	l.Refresh(false)

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  statusbarcolumn.General,
	}
}

// Confirms a user action
func (l *NotesList) ConfirmAction() message.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if l.editIndex != nil {
		selectedNote := l.SelectedItem(nil)
		newPath := filepath.Join(l.CurrentPath, l.editor.Value())
		resultMsg := ""

		switch l.EditState {
		case EditStates.Rename:
			oldPath := selectedNote.path
			if err := notes.Rename(oldPath, newPath); err == nil {
				selectedNote.name = filepath.Base(newPath)
				selectedNote.path = newPath

				// These next three lines are a bit ugly but
				// that's what they know me for
				// @todo Refresh() and build() shouldn't be necessary.
				// Find a way without those two
				l.Refresh(false)
				l.selectedIndex = l.indexByPath(newPath, nil)
				l.build()
			}

		case EditStates.Create:
			if err := notes.Create(newPath); err != nil {
				resultMsg = err.Error()
				l.Refresh(true)
			}
		}

		l.CancelAction(func() {
			l.Refresh(false)
		})

		return message.StatusBarMsg{
			Content: resultMsg,
			Sender:  message.SenderNotesList,
			Column:  statusbarcolumn.General,
		}
	}

	return message.StatusBarMsg{Sender: message.SenderNotesList}
}
