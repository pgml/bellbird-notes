package components

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/messages"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"

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

// GetIndex returns the index of a Note-Item
func (n NoteItem) GetIndex() int { return n.index }

// GetPath() returns the index of a Note-Item
func (n NoteItem) GetPath() string { return n.path }

// GetName() returns the index of a Note-Item
func (n NoteItem) GetName() string { return n.name }

// The string representation of a Dir
func (n NoteItem) String() string {
	r := n.styles.note.Render
	name := utils.TruncateText(n.GetName(), 22)
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
			EditState:        EditNone,
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
func (l *NotesList) Refresh(resetSelectedIndex bool) messages.StatusBarMsg {
	if resetSelectedIndex {
		l.selectedIndex = 0
	}

	notes, err := notes.List(l.CurrentPath)

	if err != nil {
		return messages.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    messages.Error,
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

	return messages.StatusBarMsg{}
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
// of the directories.
// To make it persistent write it to the file system
func (l *NotesList) insertDirAfter(afterIndex int, note NoteItem) {
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

func (l *NotesList) noteExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

// Create creates a directory after the last child of the currently selected directory
// If root is selected, directory will be created at the end
//
// @todo: reindex directories immediately on creating temp dir
func (l *NotesList) Create(
	mi *mode.ModeInstance,
	statusBar *StatusBar,
) messages.StatusBarMsg {
	statusMsg := messages.StatusBarMsg{}

	if l.Focused {
		mi.Current = mode.Insert
		statusBar.Focused = false

		l.EditState = EditCreate

		vrtNote := l.createVirtualNote()
		lastChild := l.getLastChild()

		if lastChild.name == "" {
			l.items = append(l.items, vrtNote)
		} else {
			l.insertDirAfter(lastChild.index, vrtNote)
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

func (l *NotesList) ConfirmRemove() messages.StatusBarMsg {
	selectedNote := l.SelectedItem(nil)
	msgType := messages.PromptError
	resultMsg := fmt.Sprintf(messages.RemovePrompt, selectedNote.path)
	l.EditState = EditDelete

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  messages.SenderNotesList,
		Column:  1,
	}
}

// Remove deletes the selected note from the file system
func (l *NotesList) Remove() messages.StatusBarMsg {
	note := l.SelectedItem(nil)
	index := l.selectedIndex
	resultMsg := ""
	msgType := messages.Success

	if err := notes.Delete(note.path); err == nil {
		l.items = slices.Delete(l.items, index, index+1)
	} else {
		msgType = messages.Error
		resultMsg = err.Error()
	}

	l.Refresh(false)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  1,
	}
}

// Confirms a user action
func (l *NotesList) ConfirmAction() messages.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if l.editIndex != nil {
		selectedNote := l.SelectedItem(nil)
		newPath := filepath.Join(l.CurrentPath, l.editor.Value())

		switch l.EditState {
		case EditRename:
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

		case EditCreate:
			if !l.noteExists(newPath) {
				notes.Create(newPath)
			}
		}

		l.CancelAction(func() {
			l.Refresh(false)
		})

		return messages.StatusBarMsg{
			Content: "yep",
			Sender:  messages.SenderNotesList,
		}
	}

	return messages.StatusBarMsg{Sender: messages.SenderNotesList}
}
