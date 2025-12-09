package application

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea/v2"

	"bellbird-notes/app/config"
	"bellbird-notes/app/state"
	bufferlist "bellbird-notes/tui/components/buffer_list"
	directorytree "bellbird-notes/tui/components/directory_tree"
	"bellbird-notes/tui/components/editor"
	noteslist "bellbird-notes/tui/components/notes_list"
	"bellbird-notes/tui/components/overlay"
	"bellbird-notes/tui/components/statusbar"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
)

type FocusController interface {
	FocusColumn(index int) message.StatusBarMsg
}

type App struct {
	// Conf holds the application's user configuration.
	Conf *config.Config

	// Conf holds the application's user configuration.
	State *state.State

	// Mode tracks the current vim mode
	Mode *mode.ModeInstance

	// DirTree manages the display and state of the directory sidebar.
	DirTree *directorytree.DirectoryTree

	// NotesList displays the list of notes in the current context.
	NotesList *noteslist.NotesList

	// Editor handles editing of the current note buffer.
	Editor *editor.Editor

	// BufferList holds and manages all open buffers.
	BufferList *bufferlist.BufferList

	// StatusBar displays current status information at the bottom of the screen.
	StatusBar *statusbar.StatusBar

	// Buffers holds the loaded content of all open files.
	Buffers editor.Buffers

	KeyInput *keyinput.Input

	// ShouldQuit is set to true when the user requests to exit the application.
	ShouldQuit bool

	// CurrColFocus stores the index of the currently focused column.
	CurrColFocus int

	// focus controls which UI component currently has focus.
	focus FocusController

	CurrentOverlay *overlay.Overlay
}

func New(fc FocusController) *App {
	conf := config.New()
	state := state.New()

	state.Read()

	app := App{
		Conf:         conf,
		State:        state,
		Mode:         &mode.ModeInstance{Current: mode.Normal},
		DirTree:      directorytree.New("Folders", conf),
		NotesList:    noteslist.New("Notes", conf),
		Editor:       editor.New("Editor", conf),
		BufferList:   bufferlist.New("BufferList", conf),
		StatusBar:    statusbar.New(),
		Buffers:      make(editor.Buffers, 0),
		CurrColFocus: 1,
		focus:        fc,
	}

	app.StatusBar.State = state

	conf.CleanMetaFile()

	return &app
}

// restoreState restores the state of the TUI from the last session
func (app *App) RestoreState() {
	currComp, err := app.Conf.MetaValue("", config.CurrentComponent)
	colIndex := 1

	if err == nil && currComp != "" {
		index, _ := strconv.Atoi(currComp)
		colIndex = index
	}

	// focus notes list if there's not open note in meta conf but
	currentNote, err := app.Conf.MetaValue("", config.LastOpenNote)
	if err == nil && currentNote == "" {
		colIndex = 2
	}

	app.focus.FocusColumn(colIndex)
}

func (app *App) componentsReady() bool {
	return app.DirTree.IsReady && app.NotesList.IsReady && app.Editor.IsReady
}

// updateComponents dispatches updates to the focused components
// (directory tree, notes list, editor), updates the current editor mode
func (app *App) UpdateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	app.DirTree.RefreshSize()
	app.NotesList.RefreshSize()
	app.BufferList.RefreshSize()
	app.Editor.RefreshSize()

	if app.componentsReady() && !app.Editor.LastOpenNoteLoaded {
		app.Editor.OpenLastNotes()
		app.Editor.LastOpenNoteLoaded = true

		cmds = append(cmds, shared.SendRefreshUiMsg())
	}

	// focus notes list if not buffer is open
	if app.Editor.IsReady && len(*app.Editor.Buffers) == 0 {
		//a.focusColumn(2)
	}

	if app.DirTree.Focused() {
		app.DirTree.Mode = app.Mode.Current
		_, cmd := app.DirTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	if app.NotesList.Focused() {
		app.NotesList.Mode = app.Mode.Current
		_, cmd := app.NotesList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if app.Editor.Focused() || app.BufferList.Focused() {
		_, cmd := app.Editor.Update(msg)
		cmds = append(cmds, cmd)

		// sync modes
		editorMode := app.Editor.Mode.Current
		app.Mode.Current = editorMode

		// ensure we canceled the search and removed all match
		// highlights
		if len(app.Editor.Textarea.Search.Matches) == 0 &&
			app.Mode.Current != mode.SearchPrompt {

			app.Editor.CancelSearch()
		}

		// Hire cursor in when search prompt is active
		app.Editor.Textarea.VirtualCursor = (app.Mode.Current != mode.SearchPrompt)

		// This is probably a dirty workaround - since key events are
		// being executed before the editor receives updates, insert
		// mode is already active which means we already start typing
		// with the initial key that is only supposed to go into insert mode.
		// So we set this flag AFTER the editor update method so that
		// insert mode is activated but doesn't immediately receive any
		// input
		app.Editor.CanInsert = false
		if editorMode == mode.Insert || editorMode == mode.Replace {
			app.Editor.CanInsert = true
		}
	}

	switch msg := msg.(type) {
	case editor.RefreshBufferMsg:
		app.Editor.Update(msg)

	case editor.SwitchBufferMsg:
		app.KeyInput.FetchKeyMap(true)
		app.BufferList.SelectedIndex = 0
		// send the switch request to the editor
		app.Editor.Update(msg)

		if msg.FocusEditor {
			app.focus.FocusColumn(3)
		}
	}

	// let the buffer list know if anything changes
	if app.BufferList.NeedsUpdate() {
		cmds = append(cmds, editor.SendBuffersChangedMsg(app.Editor.Buffers))
	}

	if _, cmd := app.BufferList.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// collect dirty buffers
	app.NotesList.DirtyBuffers = app.Editor.DirtyBuffers()

	if app.StatusBar.Mode != mode.Search {
		app.StatusBar.Mode = app.Mode.Current
	}

	return cmds
}
