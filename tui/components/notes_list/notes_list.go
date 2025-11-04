package noteslist

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/notes"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components/editor"
	sb "bellbird-notes/tui/components/statusbar"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type NoteItem struct {
	shared.Item
	IsDirty bool
}

// String is string representation of a Note
func (n NoteItem) String() string {
	baseStyle := n.Styles.Base
	iconStyle := n.Styles.Icon
	name := utils.TruncateText(n.Name(), 24)

	if n.IsSelected {
		baseStyle = n.Styles.Selected
		iconStyle = n.Styles.IconSelected
	}

	var icon strings.Builder
	icon.WriteByte(' ')

	if n.IsDirty {
		iconStyle = iconStyle.Foreground(theme.ColourDirty)
		icon.WriteString(theme.Icon(theme.IconDot, n.NerdFonts))
	} else if n.IsCut() {
		baseStyle = baseStyle.Foreground(theme.ColourBorder)
		iconStyle = iconStyle.Foreground(theme.ColourBorder)
		icon.WriteString(theme.Icon(theme.IconNote, n.NerdFonts))
	} else if n.IsPinned {
		icon.WriteString(theme.Icon(theme.IconPin, n.NerdFonts))
		iconStyle = iconStyle.Foreground(theme.ColourBorderFocused)
	} else {
		icon.WriteString(theme.Icon(theme.IconNote, n.NerdFonts))
	}

	return iconStyle.Render(icon.String()) + baseStyle.Render(name)
}

type NotesList struct {
	shared.List[*NoteItem]

	// The directory path of the currently displayed notes.
	// This path might not match the directory that is selected in the
	// directory tree since we don't automatically display a directory's
	// content on a selection change
	CurrentPath string

	// Contains dirty buffers of the current notes list
	DirtyBuffers []editor.Buffer

	// Buffers holds all the open buffers
	Buffers *editor.Buffers
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
		if l.EditIndex != nil && !l.InputModel.Focused() {
			l.InputModel.Focus()
			return l, nil
		}

		if l.InputModel.Focused() {
			l.InputModel.Focus()
			l.InputModel, cmd = l.InputModel.Update(msg)
			return l, cmd
		}

	case tea.WindowSizeMsg:
		l.Size.Width = msg.Width
		l.Size.Height = msg.Height

		if !l.IsReady {
			l.Viewport = viewport.New()
			l.Viewport.SetContent(l.viewportContent())
			l.Viewport.KeyMap = viewport.KeyMap{}
			l.LastVisibleLine = l.Viewport.VisibleLineCount() - shared.ReservedLines
			l.IsReady = true
		} else {
			l.Viewport.SetWidth(l.Size.Width)
			l.Viewport.SetHeight(l.Size.Height)
		}
	}

	// Handle keyboard and mouse events in the viewport
	l.Viewport, cmd = l.Viewport.Update(msg)
	return l, cmd
}

func (l *NotesList) RefreshSize() {
	vp := l.Viewport
	if vp.Width() != l.Size.Width && vp.Height() != l.Size.Height {
		l.Viewport.SetWidth(l.Size.Width)
		l.Viewport.SetHeight(l.Size.Height)
	}
}

func (l *NotesList) View() tea.View {
	var view tea.View
	view.SetContent(l.Content())
	return view
}

// NewNotesList creates a new model with default settings.
func New(conf *config.Config) *NotesList {
	ti := textinput.New()
	ti.Prompt = " " + theme.Icon(theme.IconPen, conf.NerdFonts()) + " "
	ti.VirtualCursor = true
	ti.CharLimit = 100

	notesDir, err := conf.MetaValue("", config.LastDirectory)
	if err != nil {
		debug.LogErr(err)
	}

	var list shared.List[*NoteItem]
	list.MakeEmpty()
	list.Title = "NOTES"
	list.SelectedIndex = 0
	list.Conf = conf
	list.InputModel = ti

	notesList := &NotesList{
		List:        list,
		CurrentPath: notesDir,
	}

	notesList.SetTheme(theme.New(conf))
	notesList.checkVisibility()
	notesList.Refresh(false, true)

	return notesList
}

func (l NotesList) Name() string {
	return "Notes"
}

func (l *NotesList) Content() string {
	if !l.IsReady {
		return "\n  Initializing..."
	}

	if !l.Visible() {
		return ""
	}

	l.Viewport.SetContent(l.viewportContent())
	l.UpdateViewportInfo()

	l.Viewport.Style = l.Theme().BaseColumnLayout(
		l.Size,
		l.Focused(),
	)

	var view strings.Builder
	view.WriteString(l.BuildHeader(l.Size.Width, false))
	view.WriteString(l.Viewport.View())
	return view.String()
}

// build prepares the notes list as a string
func (l NotesList) viewportContent() string {
	var list strings.Builder

	dirtyMap := make(map[string]struct{}, len(l.DirtyBuffers))
	for _, buf := range l.DirtyBuffers {
		dirtyMap[buf.Path(false)] = struct{}{}
	}

	for i, note := range l.Items {
		note.IsSelected = (l.SelectedIndex == i)
		note.SetIndex(i)

		_, isDirty := dirtyMap[note.Path()]
		note.IsDirty = isDirty

		if *app.Debug {
			// prepend list item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			list.WriteString(style.Render(fmt.Sprintf("%02d", note.Index())))
			list.WriteString(" ")
		}

		if l.EditIndex != nil && i == *l.EditIndex {
			// Show input field instead of text
			list.WriteString(l.InputModel.View())
			list.WriteByte('\n')
		} else {
			list.WriteString(note.String())
			list.WriteByte('\n')
		}
	}

	return list.String()
}

func (l *NotesList) SetBuffers(b *editor.Buffers) {
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
		l.SelectedIndex = 0
	}

	if resetPinned {
		l.PinnedItems.IsLoaded = false
	}

	if err != nil {
		return message.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    message.Error,
		}
	}

	if cap(l.Items) >= len(notesList) {
		l.Items = l.Items[:0]
	} else {
		l.Items = make([]*NoteItem, 0, len(notesList))
	}

	if !l.PinnedItems.IsLoaded {
		// reset pinned and refetch pinned notes when we entered a new directory
		l.PinnedItems.Items = make([]*NoteItem, 0, len(notesList))
		for _, note := range notesList {
			if note.IsPinned {
				item := l.createNoteItem(note, -1, true)
				l.PinnedItems.Add(&item)
			}
		}
	}

	pinnedMap := make(map[string]struct{}, len(l.PinnedItems.Items))
	for _, n := range l.PinnedItems.Items {
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
			noteItem.SetIsCut(buf.IsCut())
		}

		if isPinned {
			pinnedItems = append(pinnedItems, &noteItem)
		} else {
			unpinnedItems = append(unpinnedItems, &noteItem)
		}
	}

	l.Items = append(pinnedItems, unpinnedItems...)
	l.PinnedItems.IsLoaded = true

	l.Length = len(l.Items)
	l.LastIndex = 0

	if l.Length > 0 {
		l.LastIndex = l.Items[len(l.Items)-1].Index()
	}

	l.checkVisibility()

	return message.StatusBarMsg{}
}

// createNoteItem creates a NoteItem from a note, applying styles and pinning logic.
// If the note is pinned and not yet loaded, it is added to the pinned notes list.
func (l *NotesList) createNoteItem(note notes.Note, index int, isPinned bool) NoteItem {
	style := shared.NotesListStyle()
	iconWidth := style.IconWidth

	var item shared.Item
	item.SetIndex(index)
	item.SetName(note.Name())
	item.SetPath(note.Path)
	item.Styles = style
	item.NerdFonts = l.Conf.NerdFonts()
	item.IsPinned = isPinned

	noteItem := NoteItem{Item: item}
	noteItem.Styles.Icon = style.Icon.Width(iconWidth)
	noteItem.Styles.IconSelected = style.Selected.Width(iconWidth)

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
	noteItem.SetIndex(len(l.Items))

	return noteItem
}

// getLastChild returns the last NoteItem in the current directory
func (l NotesList) getLastChild() *NoteItem {
	if len(l.Items) <= 0 {
		return nil
	}
	return l.Items[len(l.Items)-1]
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// l.items. To make it persistent write it to the file system
func (l *NotesList) insertNoteAfter(afterIndex int, note NoteItem) {
	for i, dir := range l.Items {
		if dir.Index() == afterIndex {
			l.Items = append(
				l.Items[:i+1],
				append([]*NoteItem{&note}, l.Items[i+1:]...)...,
			)
			break
		}
	}
}

// Create creates a note after the last child
func (l *NotesList) Create(
	mi *mode.ModeInstance,
	statusBar *sb.StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if l.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		l.EditState = shared.EditStates.Create
		vrtNote := l.createVirtualNote()
		lastChild := l.getLastChild()

		if lastChild == nil {
			l.Items = append(l.Items, &vrtNote)
		} else {
			l.insertNoteAfter(lastChild.Index(), vrtNote)
			l.SelectedIndex = lastChild.Index() + 1
		}

		if l.EditIndex == nil {
			index := l.SelectedIndex
			l.EditIndex = &index
			l.InputModel.SetValue(vrtNote.Name())
			l.InputModel.CursorEnd()
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

	l.EditState = shared.EditStates.Delete

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
	index := l.SelectedIndex
	resultMsg := "213"
	msgType := message.Success

	if err := notes.Delete(note.Path()); err == nil {
		l.Items = slices.Delete(l.Items, index, index+1)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	l.Refresh(false, false)

	// if we deleted the last item in the list select the note
	// that is the last after the deletion
	if l.SelectedIndex >= len(l.Items) {
		l.SelectedIndex = len(l.Items) - 1
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
	if l.EditIndex != nil {
		selectedNote := l.SelectedItem(nil)
		ext := notes.Ext

		if selectedNote != nil {
			ext = filepath.Ext(selectedNote.Path())
		}

		oldPath := ""
		newPath := filepath.Join(l.CurrentPath, l.InputModel.Value()+ext)
		resultMsg := ""
		var cmd tea.Cmd

		switch l.EditState {
		case shared.EditStates.Rename:
			oldPath = selectedNote.Path()

			if err := notes.Rename(oldPath, newPath); err == nil {
				selectedNote.SetName(filepath.Base(newPath))
				selectedNote.SetPath(newPath)

				if oldPath != newPath {
					// update the meta file so we don't lose meta data
					if err := l.Conf.RenameMetaSection(oldPath, newPath); err != nil {
						debug.LogErr(err)
					}

					// Update Buffers so that all other components know
					// what's going on
					if buf := l.Buffers.Find(oldPath); buf != nil {
						buf.SetPath(newPath)
						cmd = editor.SendRefreshBufferMsg(buf.Path(false))
					}

					l.Refresh(false, true)
				}
			}

		case shared.EditStates.Create:
			if note, err := notes.Create(newPath); err == nil {
				l.Refresh(true, true)

				if note, ok := l.ItemsContain(note.Path); ok {
					l.SelectedIndex = note.Index()
				} else {
					debug.LogErr(ok)
				}

				autoOpenNewNote, _ := l.Conf.Value(
					config.General,
					config.AutoOpenNewNote,
				)
				if autoOpenNewNote.GetBool() {
					cmd = editor.SendSwitchBufferMsg(note.Path, true)
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
func (l *NotesList) TogglePinnedItems() message.StatusBarMsg {
	note := l.SelectedItem(nil)

	l.TogglePinned(note)
	l.Refresh(false, false)

	// get the new index and select the newly pinned or unpinned note
	// since the pinned notes are always at the top and the notes order
	// is changed
	for i, it := range l.Items {
		if it.Path() == note.Path() {
			l.SelectedIndex = i
		}
	}

	return message.StatusBarMsg{}
}

// YankSelection clears the yankedItems list and adds the currently selected item
// from the NotesList to it. This simulates copying an item for later pasting.
func (l *NotesList) YankSelection(markCut bool) {
	sel := l.SelectedItem(nil)
	sel.SetIsCut(markCut)

	l.YankedItems = []*NoteItem{}
	l.YankedItems = append(l.YankedItems, sel)
}

// PasteSelection duplicates all yanked notes into the specified directory path.
// It handles name conflicts by appending " Copy" to the note name until a unique
// path is found. Returns an error if any note cannot be created.
func (l *NotesList) PasteSelectedItems() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	dirPath := l.CurrentPath

	for _, note := range l.YankedItems {
		l.PasteSelection(note, dirPath, func(newPath string) {
			err := notes.Copy(note.Path(), newPath)

			if err != nil {
				debug.LogErr(err)
			}

			l.Refresh(true, true)

			// select the currently pasted item
			if note, ok := l.ItemsContain(newPath); ok {
				l.SelectedIndex = note.Index()
			}

			// Remove the original note if it's marked for moving (cut)
			if note.IsCut() {
				if err := notes.Delete(note.Path()); err != nil {
					debug.LogErr(err)
				}
			}
		})
	}

	return statusMsg
}

func (l *NotesList) Toggle() message.StatusBarMsg {
	l.ToggleVisibility()

	l.Conf.SetValue(
		config.Notes,
		config.Visible,
		strconv.FormatBool(l.Visible()),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (l *NotesList) checkVisibility() {
	vis, err := l.Conf.Value(config.Notes, config.Visible)

	if err != nil {
		debug.LogErr(err)
	}

	l.SetVisibility(vis.GetBool())
}
