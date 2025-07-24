package components

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea/v2"

	"bellbird-notes/app/config"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
)

type FocusController interface {
	FocusColumn(index int) message.StatusBarMsg
}

type App struct {
	// Conf holds the application's user configuration.
	Conf *config.Config

	// Mode tracks the current vim mode
	Mode *mode.ModeInstance

	// DirTree manages the display and state of the directory sidebar.
	DirTree *DirectoryTree

	// NotesList displays the list of notes in the current context.
	NotesList *NotesList

	// Editor handles editing of the current note buffer.
	Editor *Editor

	// BufferList holds and manages all open buffers.
	BufferList *BufferList

	// StatusBar displays current status information at the bottom of the screen.
	StatusBar *StatusBar

	// Buffers holds the loaded content of all open files.
	Buffers Buffers

	// ShouldQuit is set to true when the user requests to exit the application.
	ShouldQuit bool

	// CurrColFocus stores the index of the currently focused column.
	CurrColFocus int

	// focus controls which UI component currently has focus.
	focus FocusController
}

func NewApp(fc FocusController) *App {
	conf := config.New()
	conf.SetDefaults()

	a := App{
		Conf:         conf,
		Mode:         &mode.ModeInstance{Current: mode.Normal},
		DirTree:      NewDirectoryTree(conf),
		NotesList:    NewNotesList(conf),
		Editor:       NewEditor(conf),
		BufferList:   NewBufferList(conf),
		StatusBar:    NewStatusBar(),
		Buffers:      make(Buffers, 0),
		CurrColFocus: 1,
		focus:        fc,
	}

	return &a
}

// restoreState restores the state of the TUI from the last session
func (a *App) RestoreState() {
	currComp, err := a.Conf.MetaValue("", config.CurrentComponent)
	colIndex := 1

	if err == nil && currComp != "" {
		index, _ := strconv.Atoi(currComp)
		colIndex = index
	}

	// focus notes list if there's not open note in meta conf but
	currentNote, err := a.Conf.MetaValue("", config.LastOpenNote)
	if err == nil && currentNote == "" {
		colIndex = 2
	}

	a.focus.FocusColumn(colIndex)
}

func (a *App) componentsReady() bool {
	return a.DirTree.Ready && a.NotesList.Ready && a.Editor.Ready
}

// updateComponents dispatches updates to the focused components
// (directory tree, notes list, editor), updates the current editor mode
func (a *App) UpdateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	a.DirTree.RefreshSize()
	a.NotesList.RefreshSize()
	a.BufferList.RefreshSize()
	a.Editor.RefreshSize()

	if a.componentsReady() && !a.Editor.LastOpenNoteLoaded {
		a.Editor.OpenLastNotes()
		a.Editor.LastOpenNoteLoaded = true
	}

	// focus notes list if not buffer is open
	if a.Editor.Ready && len(*a.Editor.Buffers) == 0 {
		//a.focusColumn(2)
	}

	if a.DirTree.Focused() {
		a.DirTree.Mode = a.Mode.Current
		_, cmd := a.DirTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.NotesList.Focused() {
		a.NotesList.Mode = a.Mode.Current
		_, cmd := a.NotesList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.Editor.Focused() {
		_, cmd := a.Editor.Update(msg)
		cmds = append(cmds, cmd)
		editorMode := a.Editor.Mode.Current
		a.Mode.Current = editorMode

		// This is probably a dirty workaround - since key events are
		// being executed before the editor receives updates, insert
		// mode is already active which means we already start typing
		// with the initial key that is only supposed to go into insert mode.
		// So we set this flag AFTER the editor update method so that
		// insert mode is activated but doesn't immediately receive any
		// input
		a.Editor.CanInsert = false
		if editorMode == mode.Insert || editorMode == mode.Replace {
			a.Editor.CanInsert = true
		}
	}

	// let the buffer list know if anything changes
	if a.BufferList.NeedsUpdate() {
		cmds = append(cmds, a.Editor.SendBuffersChangedMsg())
	}

	_, cmd := a.BufferList.Update(msg)
	cmds = append(cmds, cmd)

	// collect dirty buffers
	a.NotesList.DirtyBuffers = a.Editor.DirtyBuffers()

	return cmds
}

// updateStatusBar synchronises the status bar
// with the current component states and mode.
func (a *App) UpdateStatusBar() {
	a.StatusBar.Editor = *a.Editor

	currMode := a.Mode.Current
	if currMode != mode.Normal {
		a.StatusBar.Mode = currMode
	} else {
		a.StatusBar.Mode = a.Mode.Current
	}
}

// overlayPosition returns the top center position of the application screen
func (a *App) OverlayPosition(overlayWidth int) (int, int) {
	termW, _ := theme.TerminalSize()

	x := (termW / 2) - (overlayWidth / 2)
	y := 2

	return x, y
}
