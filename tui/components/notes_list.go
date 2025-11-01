package components

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type NoteItem struct {
	Item
	IsDirty bool
}

// Path returns the index of a Note-Item
func (i NoteItem) Path() string { return i.path }

// String is string representation of a Note
func (n NoteItem) String() string {
	baseStyle := n.styles.base
	iconStyle := n.styles.icon
	name := utils.TruncateText(n.Name(), 24)

	if n.selected {
		baseStyle = n.styles.selected
		iconStyle = n.styles.iconSelected
	}

	var icon strings.Builder
	icon.WriteByte(' ')

	if n.IsDirty {
		iconStyle = iconStyle.Foreground(theme.ColourDirty)
		icon.WriteString(theme.Icon(theme.IconDot, n.nerdFonts))
	} else if n.isCut {
		baseStyle = baseStyle.Foreground(theme.ColourBorder)
		iconStyle = iconStyle.Foreground(theme.ColourBorder)
		icon.WriteString(theme.Icon(theme.IconNote, n.nerdFonts))
	} else if n.isPinned {
		icon.WriteString(theme.Icon(theme.IconPin, n.nerdFonts))
		iconStyle = iconStyle.Foreground(theme.ColourBorderFocused)
	} else {
		icon.WriteString(theme.Icon(theme.IconNote, n.nerdFonts))
	}

	return iconStyle.Render(icon.String()) + baseStyle.Render(name)
}

type NotesList struct {
	List[*NoteItem]

	// The directory path of the currently displayed notes.
	// This path might not match the directory that is selected in the
	// directory tree since we don't automatically display a directory's
	// content on a selection change
	CurrentPath string

	// Contains dirty buffers of the current notes list
	DirtyBuffers []Buffer

	// Buffers holds all the open buffers
	Buffers *Buffers
}

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (l *NotesList) Init() tea.Cmd {
	return nil
}

func (l *NotesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// focus the input field when renaming a list item
		if l.editIndex != nil && !l.input.Focused() {
			l.input.Focus()
			return l, nil
		}

		if l.input.Focused() {
			l.input.Focus()
			l.input, cmd = l.input.Update(msg)
			return l, cmd
		}

	case tea.WindowSizeMsg:
		l.Size.Width = msg.Width
		l.Size.Height = msg.Height

		if !l.Ready {
			l.viewport = viewport.New()
			l.viewport.SetContent(l.build())
			l.viewport.KeyMap = viewport.KeyMap{}
			l.lastVisibleLine = l.viewport.VisibleLineCount() - reservedLines
			l.Ready = true
		} else {
			l.viewport.SetWidth(l.Size.Width)
			l.viewport.SetHeight(l.Size.Height)
		}
	}

	// Handle keyboard and mouse events in the viewport
	l.viewport, cmd = l.viewport.Update(msg)

	return l, cmd
}

func (l *NotesList) RefreshSize() {
	vp := l.viewport
	if vp.Width() != l.Size.Width && vp.Height() != l.Size.Height {
		l.viewport.SetWidth(l.Size.Width)
		l.viewport.SetHeight(l.Size.Height)
	}
}

func (l *NotesList) View() string {
	if !l.Ready {
		return "\n  Initializing..."
	}

	if !l.visible {
		return ""
	}

	l.viewport.SetContent(l.build())
	l.UpdateViewportInfo()

	l.viewport.Style = l.theme.BaseColumnLayout(
		l.Size,
		l.Focused(),
	)

	var view strings.Builder
	view.WriteString(l.BuildHeader(l.Size.Width, false))
	view.WriteString(l.viewport.View())
	return view.String()
}

// NewNotesList creates a new model with default settings.
func NewNotesList(conf *config.Config) *NotesList {
	ti := textinput.New()
	ti.Prompt = " " + theme.Icon(theme.IconPen, conf.NerdFonts()) + " "
	ti.VirtualCursor = true
	ti.CharLimit = 100

	notesDir, err := conf.MetaValue("", config.LastDirectory)
	if err != nil {
		debug.LogErr(err)
	}

	list := &NotesList{
		List: List[*NoteItem]{
			title:            "NOTES",
			selectedIndex:    0,
			editIndex:        nil,
			EditState:        EditStates.None,
			input:            ti,
			lastVisibleLine:  0,
			firstVisibleLine: 0,
			items:            make([]*NoteItem, 0),
			conf:             conf,
			PinnedItems:      PinnedItems[*NoteItem]{},
		},
		CurrentPath: notesDir,
	}

	list.theme = theme.New(conf)
	list.visible = list.Visible()
	list.Refresh(false, true)

	return list
}

func (l NotesList) Name() string { return "Notes" }

// build prepares the notes list as a string
func (l NotesList) build() string {
	var list strings.Builder

	dirtyMap := make(map[string]struct{}, len(l.DirtyBuffers))
	for _, buf := range l.DirtyBuffers {
		dirtyMap[buf.path] = struct{}{}
	}

	for i, note := range l.items {
		note.selected = (l.selectedIndex == i)
		note.index = i

		_, isDirty := dirtyMap[note.path]
		note.IsDirty = isDirty

		if *app.Debug {
			// prepend list item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			list.WriteString(style.Render(fmt.Sprintf("%02d", note.index)))
			list.WriteString(" ")
		}

		if l.editIndex != nil && i == *l.editIndex {
			// Show input field instead of text
			list.WriteString(l.input.View())
			list.WriteByte('\n')
		} else {
			list.WriteString(note.String())
			list.WriteByte('\n')
		}
	}

	return list.String()
}

func (l *NotesList) SetBuffers(b *Buffers) {
	l.Buffers = b
}

// Refresh updates the notes list
//
// If `resetIndex` is set to true, 'l.selectedIndex' will be set to 0
// which representns the first note
func (l *NotesList) Refresh(
	resetSelectedIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	notesList, err := notes.List(l.CurrentPath)

	if resetSelectedIndex {
		l.selectedIndex = 0
	}

	if resetPinned {
		l.PinnedItems.loaded = false
	}

	if err != nil {
		return message.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    message.Error,
		}
	}

	if cap(l.items) >= len(notesList) {
		l.items = l.items[:0]
	} else {
		l.items = make([]*NoteItem, 0, len(notesList))
	}

	if !l.PinnedItems.loaded {
		// reset pinned and refetch pinned notes when we entered a new directory
		l.PinnedItems.items = make([]*NoteItem, 0, len(notesList))
		for _, note := range notesList {
			if note.IsPinned {
				item := l.createNoteItem(note, -1, true)
				l.PinnedItems.add(&item)
			}
		}
	}

	pinnedMap := make(map[string]struct{}, len(l.PinnedItems.items))
	for _, n := range l.PinnedItems.items {
		pinnedMap[n.Path()] = struct{}{}
	}

	var (
		pinnedItems   []*NoteItem
		unpinnedItems []*NoteItem
	)

	for i, note := range notesList {
		_, isPinned := pinnedMap[note.Path]
		noteItem := l.createNoteItem(note, i, isPinned)

		if buf, ok := l.YankedItemsContain(note.Path); ok {
			noteItem.isCut = buf.isCut
		}

		if isPinned {
			pinnedItems = append(pinnedItems, &noteItem)
		} else {
			unpinnedItems = append(unpinnedItems, &noteItem)
		}
	}

	l.items = append(pinnedItems, unpinnedItems...)
	l.PinnedItems.loaded = true

	l.length = len(l.items)
	l.lastIndex = 0

	if l.length > 0 {
		l.lastIndex = l.items[len(l.items)-1].index
	}

	l.visible = l.Visible()

	return message.StatusBarMsg{}
}

// createNoteItem creates a NoteItem from a note, applying styles and pinning logic.
// If the note is pinned and not yet loaded, it is added to the pinned notes list.
func (l *NotesList) createNoteItem(note notes.Note, index int, isPinned bool) NoteItem {
	style := NotesListStyle()
	iconWidth := style.iconWidth

	noteItem := NoteItem{
		Item: Item{
			index:     index,
			name:      note.Name(),
			path:      note.Path,
			styles:    style,
			nerdFonts: l.conf.NerdFonts(),
			isPinned:  isPinned,
		},
	}

	noteItem.styles.icon = style.icon.Width(iconWidth)
	noteItem.styles.iconSelected = style.selected.Width(iconWidth)

	return noteItem
}

// createVirtualNote creates a virtual note `Note`
// with dummy data
//
// This note is mainly used as a placeholder when creating a note
// and is not actually written to the file system.
func (l *NotesList) createVirtualNote() NoteItem {
	name := "New Note"

	path := filepath.Join(
		filepath.Dir(l.CurrentPath),
		name,
	)

	item := notes.NewNote(path, false)
	noteItem := l.createNoteItem(item, -1, false)
	noteItem.index = len(l.items)

	return noteItem
}

// getLastChild returns the last NoteItem in the current directory
func (l NotesList) getLastChild() *NoteItem {
	if len(l.items) <= 0 {
		return nil
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
				append([]*NoteItem{&note}, l.items[i+1:]...)...,
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

	if l.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		l.EditState = EditStates.Create
		vrtNote := l.createVirtualNote()
		lastChild := l.getLastChild()

		if lastChild == nil {
			l.items = append(l.items, &vrtNote)
		} else {
			l.insertNoteAfter(lastChild.index, vrtNote)
			l.selectedIndex = lastChild.index + 1
		}

		if l.editIndex == nil {
			l.editIndex = &l.selectedIndex
			l.input.SetValue(vrtNote.name)
			l.input.CursorEnd()
		}
	}

	return statusMsg
}

func (l *NotesList) ConfirmRemove() message.StatusBarMsg {
	selectedNote := *l.SelectedItem(nil)
	msgType := message.PromptError

	rootDir, _ := app.NotesRootDir()
	path := strings.TrimPrefix(selectedNote.Path(), rootDir+"/")
	resultMsg := fmt.Sprintf(message.StatusBar.RemovePrompt, path)

	l.EditState = EditStates.Delete

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  message.SenderNotesList,
		Column:  sbc.General,
	}
}

// Remove deletes the selected note from the file system
func (l *NotesList) Remove() message.StatusBarMsg {
	note := *l.SelectedItem(nil)
	index := l.selectedIndex
	resultMsg := "213"
	msgType := message.Success

	if err := notes.Delete(note.path); err == nil {
		l.items = slices.Delete(l.items, index, index+1)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	l.Refresh(false, false)

	// if we deleted the last item in the list select the note
	// that is the last after the deletion
	if l.selectedIndex >= len(l.items) {
		l.selectedIndex = len(l.items) - 1
	}

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  sbc.General,
	}
}

// ConfirmAction confirms a user action
func (l *NotesList) ConfirmAction() message.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if l.editIndex != nil {
		selectedNote := l.SelectedItem(nil)
		ext := notes.Ext

		if selectedNote != nil {
			ext = filepath.Ext(selectedNote.path)
		}

		oldPath := ""
		newPath := filepath.Join(l.CurrentPath, l.input.Value()+ext)
		resultMsg := ""
		var cmd tea.Cmd

		switch l.EditState {
		case EditStates.Rename:
			oldPath = selectedNote.path

			if err := notes.Rename(oldPath, newPath); err == nil {
				selectedNote.name = filepath.Base(newPath)
				selectedNote.path = newPath

				if oldPath != newPath {
					// update the meta file so we don't lose meta data
					if err := l.conf.RenameMetaSection(oldPath, newPath); err != nil {
						debug.LogErr(err)
					}

					// Update Buffers so that all other components know
					// what's going on
					if buf, ok, _ := l.Buffers.Contain(oldPath); ok {
						buf.path = newPath
						cmd = SendRefreshBufferMsg(buf.Path(false))
					}

					l.Refresh(false, true)
				}
			}

		case EditStates.Create:
			if note, err := notes.Create(newPath); err == nil {
				l.Refresh(true, true)

				if note, ok := l.ItemsContain(note.Path); ok {
					l.selectedIndex = note.index
				} else {
					debug.LogErr(ok)
				}

				autoOpenNewNote, _ := l.conf.Value(config.General, config.AutoOpenNewNote)
				if autoOpenNewNote.GetBool() {
					cmd = SendSwitchBufferMsg(note.Path, true)
				}

				resultMsg = note.Path
			} else {
				debug.LogErr(err)
			}
		}

		l.CancelAction(func() {
			l.Refresh(false, false)
		})

		return message.StatusBarMsg{
			Content: resultMsg,
			Sender:  message.SenderNotesList,
			Column:  sbc.General,
			Cmd:     cmd,
		}
	}

	return message.StatusBarMsg{Sender: message.SenderNotesList}
}

// TogglePinned pins or unpins the current selection
func (l *NotesList) TogglePinned() message.StatusBarMsg {
	note := l.SelectedItem(nil)

	l.togglePinned(note)
	l.Refresh(false, false)

	// get the new index and select the newly pinned or unpinned note
	// since the pinned notes are always at the top and the notes order
	// is changed
	for i, it := range l.items {
		if it.path == note.path {
			l.selectedIndex = i
		}
	}

	return message.StatusBarMsg{}
}

// YankSelection clears the yankedItems list and adds the currently selected item
// from the NotesList to it. This simulates copying an item for later pasting.
func (l *NotesList) YankSelection(markCut bool) {
	sel := l.SelectedItem(nil)
	sel.isCut = markCut

	l.yankedItems = []*NoteItem{}
	l.yankedItems = append(l.yankedItems, sel)
}

// PasteSelection duplicates all yanked notes into the specified directory path.
// It handles name conflicts by appending " Copy" to the note name until a unique
// path is found. Returns an error if any note cannot be created.
func (l *NotesList) PasteSelection() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	dirPath := l.CurrentPath

	for _, note := range l.yankedItems {
		l.pasteSelection(note, dirPath, func(newPath string) {
			err := notes.Copy(note.Path(), newPath)

			if err != nil {
				debug.LogErr(err)
			}

			l.Refresh(true, true)

			// select the currently pasted item
			if note, ok := l.ItemsContain(newPath); ok {
				l.selectedIndex = note.index
			}

			// Remove the original note if it's marked for moving (cut)
			if note.isCut {
				if err := notes.Delete(note.path); err != nil {
					debug.LogErr(err)
				}
			}
		})
	}

	return statusMsg
}

func (l *NotesList) Visible() bool {
	vis, err := l.conf.Value(config.Notes, config.Visible)

	if err != nil {
		debug.LogErr(err)
		return false
	}

	return vis.GetBool()
}

func (l *NotesList) Toggle() message.StatusBarMsg {
	l.visible = !l.visible
	return message.StatusBarMsg{
		Cmd: SendRefreshUiMsg(),
	}
}
