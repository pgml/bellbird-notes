package components

import (
	"bellbird-notes/internal/config"
	"bellbird-notes/internal/notes"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"
	"bellbird-notes/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotesList struct {
	List[Note]

	// The directory path of the currently displayed notes.
	// This path might not match the directory that is selected in the
	// directory tree since we don't automatically display a directory's
	// content on a selection change
	CurrentPath string
}

type Note struct {
	Item
	isPinned bool
}

func (n Note) String() string {
	r := n.styles.note.Render
	name := utils.TruncateText(n.Name, 22)

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
	return n.Name
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
			l.viewport.SetContent(l.build())
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

	l.viewport.SetContent(l.build())
	l.UpdateViewportInfo()

	l.viewport.Style = theme.BaseColumnLayout(l.Size, l.Focused)
	return l.viewport.View()
}

func NewNotesList() *NotesList {
	ti := textinput.New()
	ti.Prompt = "  "
	ti.CharLimit = 100

	conf := config.New()
	list := &NotesList{
		List: List[Note]{
			selectedIndex:    0,
			editingIndex:     nil,
			editingState:     EditNone,
			editor:           ti,
			lastVisibleLine:  0,
			firstVisibleLine: 0,
			items:            make([]Note, 0, 0),
		},
		//notes:       make([]Note, 0, 0),
		CurrentPath: conf.Value(config.General, config.UserNotesDirectory),
	}

	list.Refresh()
	return list
}

func (l NotesList) build() string {
	var list string

	for i, note := range l.items {
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

	l.items = make([]Note, 0, len(notes))

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

func (l *NotesList) createNoteItem(note notes.Note) Note {
	style := NotesListStyle()
	childItem := Note{
		Item: Item{
			index:  0,
			Name:   note.Name,
			Path:   note.Path,
			styles: style,
		},
		isPinned: note.IsPinned,
	}
	return childItem
}

// createVirtualNote creates a temporary, virtual note `Note`
//
// This note is mainly used as a placeholder when creating a note
func (l *NotesList) createVirtualNote() Note {
	selectedNote := l.SelectedItem(nil)
	tempNoteName := "New Note"
	tempNotePath := filepath.Join(filepath.Dir(selectedNote.Path), tempNoteName)

	return Note{
		Item: Item{
			index: len(l.items),
			Name:  tempNoteName,
			Path:  tempNotePath,
		},
	}
}

func (l NotesList) getLastChild() Note {
	if len(l.items) <= 0 {
		return Note{}
	}
	return l.items[len(l.items)-1]
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// of the directories.
// To make it persistent write it to the file system
func (l *NotesList) insertDirAfter(afterIndex int, note Note) {
	for i, dir := range l.items {
		if dir.index == afterIndex {
			l.items = append(
				l.items[:i+1],
				append([]Note{note}, l.items[i+1:]...)...,
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
func (l *NotesList) Create() messages.StatusBarMsg {
	l.editingState = EditCreate
	tmpNote := l.createVirtualNote()
	lastChild := l.getLastChild()
	if lastChild.Name == "" {
		l.items = append(l.items, tmpNote)
	} else {
		l.insertDirAfter(lastChild.index, tmpNote)
		l.selectedIndex = lastChild.index + 1
	}

	if l.editingIndex == nil {
		l.editingIndex = &l.selectedIndex
		l.editor.SetValue(l.SelectedItem(nil).Name)
	}
	return messages.StatusBarMsg{}
}

func (l *NotesList) ConfirmRemove() messages.StatusBarMsg {
	selectedNote := l.SelectedItem(nil)
	msgType := messages.PromptError
	resultMsg := fmt.Sprintf(messages.RemovePrompt, selectedNote.Path)

	return messages.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  messages.SenderNotesList,
	}
}

// Renames the currently selected directory
// Returns a message to be displayed in the status bar
func (l *NotesList) Remove() messages.StatusBarMsg {
	note := l.SelectedItem(nil)
	index := l.selectedIndex
	resultMsg := fmt.Sprintf(messages.SuccessRemove, note.Path)
	msgType := messages.Success

	if err := notes.Delete(note.Path); err == nil {
		l.items = slices.Delete(l.items, index, index+1)
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
		selectedNote := l.SelectedItem(nil)
		newPath := filepath.Join(l.CurrentPath, l.editor.Value())

		switch l.editingState {
		case EditRename:
			oldPath := selectedNote.Path
			// rename if path exists
			if _, err := os.Stat(oldPath); err == nil {
				notes.Rename(oldPath, newPath)
				selectedNote.Name = filepath.Base(newPath)
				selectedNote.Path = newPath
			}

		case EditCreate:
			if !l.noteExists(newPath) {
				notes.Create(newPath)
			}
		}

		l.CancelAction(func() { l.Refresh() })
		return messages.StatusBarMsg{Content: "yep", Sender: messages.SenderNotesList}
	}

	l.Refresh()

	return messages.StatusBarMsg{Sender: messages.SenderNotesList}
}
