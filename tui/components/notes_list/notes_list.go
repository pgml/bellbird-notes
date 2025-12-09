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
func (item NoteItem) String() string {
	baseStyle := item.Styles.Base
	iconStyle := item.Styles.Icon
	name := utils.TruncateText(item.Name(), 24)

	if item.IsSelected {
		baseStyle = item.Styles.Selected
		iconStyle = item.Styles.IconSelected
	}

	var icon strings.Builder
	icon.WriteByte(' ')

	if item.IsDirty {
		iconStyle = iconStyle.Foreground(theme.ColourDirty)
		icon.WriteString(theme.Icon(theme.IconDot, item.NerdFonts))
	} else if item.IsCut() {
		baseStyle = baseStyle.Foreground(theme.ColourBorder)
		iconStyle = iconStyle.Foreground(theme.ColourBorder)
		icon.WriteString(theme.Icon(theme.IconNote, item.NerdFonts))
	} else if item.IsPinned {
		icon.WriteString(theme.Icon(theme.IconPin, item.NerdFonts))
		iconStyle = iconStyle.Foreground(theme.ColourBorderFocused)
	} else {
		icon.WriteString(theme.Icon(theme.IconNote, item.NerdFonts))
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
func (list *NotesList) Init() tea.Cmd {
	return nil
}

func (list *NotesList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// focus the input field when renaming a list item
		if list.EditIndex != nil && !list.InputModel.Focused() {
			list.InputModel.Focus()
			return list, nil
		}

		if list.InputModel.Focused() {
			list.InputModel.Focus()
			list.InputModel, cmd = list.InputModel.Update(msg)
			return list, cmd
		}

	case tea.WindowSizeMsg:
		list.Size.Width = msg.Width
		list.Size.Height = msg.Height

		if !list.IsReady {
			list.Viewport = viewport.New()
			list.Viewport.SetContent(list.viewportContent())
			list.Viewport.KeyMap = viewport.KeyMap{}
			list.LastVisibleLine = list.Viewport.VisibleLineCount() - shared.ReservedLines
			list.IsReady = true
		} else {
			list.Viewport.SetWidth(list.Size.Width)
			list.Viewport.SetHeight(list.Size.Height)
		}
	}

	// Handle keyboard and mouse events in the viewport
	list.Viewport, cmd = list.Viewport.Update(msg)
	return list, cmd
}

func (list *NotesList) RefreshSize() {
	vp := list.Viewport
	if vp.Width() != list.Size.Width && vp.Height() != list.Size.Height {
		list.Viewport.SetWidth(list.Size.Width)
		list.Viewport.SetHeight(list.Size.Height)
	}
}

func (list *NotesList) View() tea.View {
	var view tea.View
	view.SetContent(list.Content())
	return view
}

// NewNotesList creates a new model with default settings.
func New(title string, conf *config.Config) *NotesList {
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
	list.SelectedIndex = 0
	list.Conf = conf
	list.InputModel = ti

	notesList := &NotesList{
		List:        list,
		CurrentPath: notesDir,
	}

	notesList.SetTitle(title)
	notesList.SetTheme(theme.New(conf))
	notesList.checkVisibility()
	notesList.Refresh(false, true)

	return notesList
}

func (list *NotesList) Content() string {
	if !list.IsReady {
		return "\n  Initializing..."
	}

	if !list.Visible() {
		return ""
	}

	list.Viewport.SetContent(list.viewportContent())
	list.UpdateViewportInfo()

	list.Viewport.Style = list.Theme().BaseColumnLayout(
		list.Size,
		list.Focused(),
	)

	var view strings.Builder
	view.WriteString(list.BuildHeader(list.Size.Width, false))
	view.WriteString(list.Viewport.View())
	return view.String()
}

// build prepares the notes list as a string
func (list NotesList) viewportContent() string {
	var s strings.Builder

	dirtyMap := make(map[string]struct{}, len(list.DirtyBuffers))
	for _, buf := range list.DirtyBuffers {
		dirtyMap[buf.Path(false)] = struct{}{}
	}

	for i, note := range list.Items {
		note.IsSelected = (list.SelectedIndex == i)
		note.SetIndex(i)

		_, isDirty := dirtyMap[note.Path()]
		note.IsDirty = isDirty

		if *app.Debug {
			// prepend list item indices for debugging purposes
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
			s.WriteString(style.Render(fmt.Sprintf("%02d", note.Index())))
			s.WriteString(" ")
		}

		if list.EditIndex != nil && i == *list.EditIndex {
			// Show input field instead of text
			s.WriteString(list.InputModel.View())
			s.WriteByte('\n')
		} else {
			s.WriteString(note.String())
			s.WriteByte('\n')
		}
	}

	return s.String()
}

func (list *NotesList) SetBuffers(b *editor.Buffers) {
	list.Buffers = b
}

// Refresh updates the notes list
//
// If `resetIndex` is set to true, 'l.selectedIndex' will be set to 0
// which representns the first note
func (list *NotesList) Refresh(
	resetSelectedIndex bool,
	resetPinned bool,
) message.StatusBarMsg {
	notesList, err := notes.List(list.CurrentPath)

	if resetSelectedIndex {
		list.SelectedIndex = 0
	}

	if resetPinned {
		list.PinnedItems.IsLoaded = false
	}

	if err != nil {
		return message.StatusBarMsg{
			Content: "Failed to load notes",
			Type:    message.Error,
		}
	}

	if cap(list.Items) >= len(notesList) {
		list.Items = list.Items[:0]
	} else {
		list.Items = make([]*NoteItem, 0, len(notesList))
	}

	if !list.PinnedItems.IsLoaded {
		// reset pinned and refetch pinned notes when we entered a new directory
		list.PinnedItems.Items = make([]*NoteItem, 0, len(notesList))
		for _, note := range notesList {
			if note.IsPinned {
				item := list.createNoteItem(note, -1, true)
				list.PinnedItems.Add(&item)
			}
		}
	}

	pinnedMap := make(map[string]struct{}, len(list.PinnedItems.Items))
	for _, n := range list.PinnedItems.Items {
		pinnedMap[n.Path()] = struct{}{}
	}

	var (
		pinnedItems   []*NoteItem
		unpinnedItems []*NoteItem
	)

	for i, note := range notesList {
		_, isPinned := pinnedMap[note.Path]
		noteItem := list.createNoteItem(note, i, isPinned)

		if buf, ok := list.YankedItemsContain(note.Path); ok {
			noteItem.SetIsCut(buf.IsCut())
		}

		if isPinned {
			pinnedItems = append(pinnedItems, &noteItem)
		} else {
			unpinnedItems = append(unpinnedItems, &noteItem)
		}
	}

	list.Items = append(pinnedItems, unpinnedItems...)
	list.PinnedItems.IsLoaded = true

	list.Length = len(list.Items)
	list.LastIndex = 0

	if list.Length > 0 {
		list.LastIndex = list.Items[len(list.Items)-1].Index()
	}

	list.checkVisibility()

	return message.StatusBarMsg{}
}

// createNoteItem creates a NoteItem from a note, applying styles and pinning logic.
// If the note is pinned and not yet loaded, it is added to the pinned notes list.
func (list *NotesList) createNoteItem(note notes.Note, index int, isPinned bool) NoteItem {
	style := shared.NotesListStyle()
	iconWidth := style.IconWidth

	var item shared.Item
	item.SetIndex(index)
	item.SetName(note.Name())
	item.SetPath(note.Path)
	item.Styles = style
	item.NerdFonts = list.Conf.NerdFonts()
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
func (list *NotesList) createVirtualNote() NoteItem {
	name := "New Note"

	path := filepath.Join(
		filepath.Dir(list.CurrentPath),
		name,
	)

	item := notes.NewNote(path, false)
	noteItem := list.createNoteItem(item, -1, false)
	noteItem.SetIndex(len(list.Items))

	return noteItem
}

// getLastChild returns the last NoteItem in the current directory
func (list NotesList) getLastChild() *NoteItem {
	if len(list.Items) <= 0 {
		return nil
	}
	return list.Items[len(list.Items)-1]
}

// Inserts an item after `afterIndex`
//
// Note: this is only a virtual insertion into to the flat copy
// l.items. To make it persistent write it to the file system
func (list *NotesList) insertNoteAfter(afterIndex int, note NoteItem) {
	for i, dir := range list.Items {
		if dir.Index() == afterIndex {
			list.Items = append(
				list.Items[:i+1],
				append([]*NoteItem{&note}, list.Items[i+1:]...)...,
			)
			break
		}
	}
}

// Create creates a note after the last child
func (list *NotesList) Create(
	mi *mode.ModeInstance,
	statusBar *sb.StatusBar,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if list.Focused() {
		mi.Current = mode.Insert
		statusBar.Focused = false

		list.EditState = shared.EditStates.Create
		vrtNote := list.createVirtualNote()
		lastChild := list.getLastChild()

		if lastChild == nil {
			list.Items = append(list.Items, &vrtNote)
		} else {
			list.insertNoteAfter(lastChild.Index(), vrtNote)
			list.SelectedIndex = lastChild.Index() + 1
		}

		if list.EditIndex == nil {
			index := list.SelectedIndex
			list.EditIndex = &index
			list.InputModel.SetValue(vrtNote.Name())
			list.InputModel.CursorEnd()
		}
	}

	return statusMsg
}

func (list *NotesList) ConfirmRemove() message.StatusBarMsg {
	selectedNote := *list.SelectedItem(nil)
	msgType := message.PromptError

	rootDir, _ := app.NotesRootDir()
	path := strings.TrimPrefix(selectedNote.Path(), rootDir+"/")
	resultMsg := fmt.Sprintf(message.StatusBar.RemovePrompt, path)

	list.EditState = shared.EditStates.Delete

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Sender:  message.SenderNotesList,
		Column:  sbc.General,
	}
}

// Remove deletes the selected note from the file system
func (list *NotesList) Remove() message.StatusBarMsg {
	note := *list.SelectedItem(nil)
	index := list.SelectedIndex
	resultMsg := "213"
	msgType := message.Success

	if err := notes.Delete(note.Path()); err == nil {
		list.Items = slices.Delete(list.Items, index, index+1)
	} else {
		msgType = message.Error
		resultMsg = err.Error()
	}

	list.Refresh(false, false)

	// if we deleted the last item in the list select the note
	// that is the last after the deletion
	if list.SelectedIndex >= len(list.Items) {
		list.SelectedIndex = len(list.Items) - 1
	}

	return message.StatusBarMsg{
		Content: resultMsg,
		Type:    msgType,
		Column:  sbc.General,
	}
}

// ConfirmAction confirms a user action
func (list *NotesList) ConfirmAction() message.StatusBarMsg {
	// if editingindex is set it most likely means that we are
	// renaming or creating a directory
	if list.EditIndex != nil {
		selectedNote := list.SelectedItem(nil)
		ext := notes.Ext

		if selectedNote != nil {
			ext = filepath.Ext(selectedNote.Path())
		}

		oldPath := ""
		newPath := filepath.Join(list.CurrentPath, list.InputModel.Value()+ext)
		resultMsg := ""
		var cmd tea.Cmd

		switch list.EditState {
		case shared.EditStates.Rename:
			oldPath = selectedNote.Path()

			if err := notes.Rename(oldPath, newPath); err == nil {
				selectedNote.SetName(filepath.Base(newPath))
				selectedNote.SetPath(newPath)

				if oldPath != newPath {
					// update the meta file so we don't lose meta data
					if err := list.Conf.RenameMetaSection(oldPath, newPath); err != nil {
						debug.LogErr(err)
					}

					// Update Buffers so that all other components know
					// what's going on
					if buf := list.Buffers.Find(oldPath); buf != nil {
						buf.SetPath(newPath)
						cmd = editor.SendRefreshBufferMsg(buf.Path(false))
					}

					list.Refresh(false, true)
				}
			}

		case shared.EditStates.Create:
			if note, err := notes.Create(newPath); err == nil {
				list.Refresh(true, true)

				if note, ok := list.ItemsContain(note.Path); ok {
					list.SelectedIndex = note.Index()
				} else {
					debug.LogErr(ok)
				}

				autoOpenNewNote, _ := list.Conf.Value(
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

		list.CancelAction(func() {
			list.Refresh(false, false)
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
func (list *NotesList) TogglePinnedItems() message.StatusBarMsg {
	note := list.SelectedItem(nil)

	list.TogglePinned(note)
	list.Refresh(false, false)

	// get the new index and select the newly pinned or unpinned note
	// since the pinned notes are always at the top and the notes order
	// is changed
	for i, it := range list.Items {
		if it.Path() == note.Path() {
			list.SelectedIndex = i
		}
	}

	return message.StatusBarMsg{}
}

// YankSelection clears the yankedItems list and adds the currently selected item
// from the NotesList to it. This simulates copying an item for later pasting.
func (list *NotesList) YankSelection(markCut bool) {
	sel := list.SelectedItem(nil)
	sel.SetIsCut(markCut)

	list.YankedItems = []*NoteItem{}
	list.YankedItems = append(list.YankedItems, sel)
}

// PasteSelection duplicates all yanked notes into the specified directory path.
// It handles name conflicts by appending " Copy" to the note name until a unique
// path is found. Returns an error if any note cannot be created.
func (list *NotesList) PasteSelectedItems() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	dirPath := list.CurrentPath

	for _, note := range list.YankedItems {
		list.PasteSelection(note, dirPath, func(newPath string) {
			err := notes.Copy(note.Path(), newPath)

			if err != nil {
				debug.LogErr(err)
			}

			list.Refresh(true, true)

			// select the currently pasted item
			if note, ok := list.ItemsContain(newPath); ok {
				list.SelectedIndex = note.Index()
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

func (list *NotesList) Toggle() message.StatusBarMsg {
	list.ToggleVisibility()

	list.Conf.SetValue(
		config.Notes,
		config.Visible,
		strconv.FormatBool(list.Visible()),
	)

	return message.StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (list *NotesList) checkVisibility() {
	vis, err := list.Conf.Value(config.Notes, config.Visible)

	if err != nil {
		debug.LogErr(err)
	}

	list.SetVisibility(vis.GetBool())
}
