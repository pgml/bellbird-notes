package tui

import (
	"strconv"

	"bellbird-notes/app/config"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	bl "github.com/winder/bubblelayout"
)

type Focusable = interfaces.Focusable

// Model is the Bubble Tea model for the TUI
type Model struct {
	layout bl.BubbleLayout
	// Current app vim-like mode
	mode         *mode.ModeInstance
	keyInput     *keyinput.Input
	currColFocus int

	dirTree    *components.DirectoryTree
	notesList  *components.NotesList
	editor     *components.Editor
	bufferList *components.BufferList
	statusBar  *components.StatusBar
	Buffers    components.Buffers
	conf       *config.Config
}

func InitialModel() *Model {
	layout := bl.New()

	mode := &mode.ModeInstance{
		Current: mode.Normal,
	}

	conf := config.New()
	conf.SetDefaults()

	m := Model{
		layout:       layout,
		mode:         mode,
		currColFocus: 1,
		keyInput:     keyinput.New(),
		dirTree:      components.NewDirectoryTree(conf),
		notesList:    components.NewNotesList(conf),
		editor:       components.NewEditor(conf),
		bufferList:   components.NewBufferList(conf),
		statusBar:    components.NewStatusBar(),
		Buffers:      make(components.Buffers, 0),
		conf:         conf,
	}

	m.keyInput.Registry = m.FnRegistry()
	m.keyInput.Components = []keyinput.FocusedComponent{
		m.dirTree,
		m.notesList,
		m.editor,
		m.bufferList,
	}

	m.componentsInit()
	m.restoreState()

	return &m
}

func (m Model) Init() tea.Cmd {
	editorCmd := m.editor.Init()
	statusBarCmd := m.statusBar.Init()

	return tea.Batch(editorCmd, statusBarCmd)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.keyInput.AllowSequences = !m.statusBar.Focused
		statusMsg := m.keyInput.HandleSequences(msg.Key())

		// If space is pressed, reset it after a certain delay
		if msg.Key().Code == 32 {
			cmds = append(cmds, m.keyInput.ResetSequence())
		}

		if msg.String() == "ctrl+c" {
			statusMsg = []message.StatusBarMsg{{
				Content: message.StatusBar.CtrlCExitNote,
				Type:    message.Success,
				Column:  sbc.General,
			}}
		}

		statusMsg = append(statusMsg, m.editor.StatusBarInfo())

		for _, m := range statusMsg {
			cmds = append(cmds, m.Cmd)
		}

		m.statusBar = m.statusBar.Update(statusMsg, msg)
		m.keyInput.Mode = m.mode.Current
		m.statusBar.Mode = m.mode.Current

	case tea.WindowSizeMsg:
		m.dirTree.Update(msg)
		m.notesList.Update(msg)
		m.editor.Update(msg)

		// Convert WindowSizeMsg to BubbleLayoutMsg.
		return m, func() tea.Msg {
			return m.layout.Resize(
				msg.Width,
				msg.Height,
			)
		}

	case bl.BubbleLayoutMsg:
		m.dirTree.Size, _ = msg.Size(m.dirTree.ID)
		m.notesList.Size, _ = msg.Size(m.notesList.ID)
		m.editor.Size, _ = msg.Size(m.editor.ID)
		m.bufferList.Size, _ = msg.Size(m.bufferList.ID)
		m.statusBar.Size, _ = msg.Size(m.statusBar.ID)

	case keyinput.ResetSequenceMsg:
		m.statusBar = m.statusBar.Update(
			[]message.StatusBarMsg{m.keyInput.ResetKeysDown()},
			msg,
		)
	}

	m.keyInput.Mode = m.mode.Current

	// exit programme when `:q` is entered in command prompt
	if m.statusBar.ShouldQuit {
		return m, tea.Quit
	}

	cmds = append(cmds, m.updateComponents(msg)...)
	m.updateStatusBar()

	return m, tea.Batch(cmds...)
}

// View renders the TUI layout as a string
func (m Model) View() string {
	view := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.dirTree.View(),
			m.notesList.View(),
			m.editor.View(),
		),
		m.statusBar.View(),
	)

	var (
		overlay = ""
		x       = 0
		y       = 0
	)

	// check if any overlays should be displayed
	if m.editor.ListBuffers {
		overlay, x, y = m.OverlayOpenBuffers()
	} else {
		m.bufferList.SetFocus(false)
	}

	if overlay != "" {
		// place overlay above the application
		return components.PlaceOverlay(x, y, overlay, view)
	}

	return view
}

// componentsInit registers components in the layout
// and sets initial focus
func (m *Model) componentsInit() {
	const reserverdLines = 1
	statusBarHeight := m.statusBar.Height + reserverdLines

	m.dirTree.ID = m.layout.Add("width 30")

	m.notesList.ID = m.layout.Add("width 30")
	m.notesList.SetBuffers(&m.Buffers)

	m.editor.ID = m.layout.Add("grow")
	m.editor.SetBuffers(&m.Buffers)

	m.bufferList.SetBuffers(&m.Buffers)

	m.statusBar.ID = m.layout.Dock(bl.Dock{
		Cardinal:  bl.SOUTH,
		Preferred: statusBarHeight,
	})
}

// updateComponents dispatches updates to the focused components
// (directory tree, notes list, editor), updates the current editor mode
func (m *Model) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	m.dirTree.RefreshSize()
	m.notesList.RefreshSize()
	m.bufferList.RefreshSize()
	m.editor.RefreshSize()

	if m.componentsReady() && !m.editor.LastOpenNoteLoaded {
		m.editor.OpenLastNotes()
		m.editor.LastOpenNoteLoaded = true
	}

	// focus notes list if not buffer is open
	if m.editor.Ready && len(m.Buffers) == 0 {
		m.focusColumn(2)
	}

	if m.dirTree.Focused() {
		m.dirTree.Mode = m.mode.Current
		_, cmd := m.dirTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.notesList.Focused() {
		m.notesList.Mode = m.mode.Current
		_, cmd := m.notesList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.editor.Focused() {
		_, cmd := m.editor.Update(msg)
		cmds = append(cmds, cmd)
		editorMode := m.editor.Vim.Mode.Current
		m.mode.Current = editorMode
		m.keyInput.Mode = editorMode

		// This is probably a dirty workaround - since key events are
		// being executed before the editor receives updates, insert
		// mode is already active which means we already start typing
		// with the initial key that is only supposed to go into insert mode.
		// So we set this flag AFTER the editor update method so that
		// insert mode is activated but doesn't immediately receive any
		// input
		m.editor.CanInsert = false
		if editorMode == mode.Insert || editorMode == mode.Replace {
			m.editor.CanInsert = true
		}
	}

	// let the buffer list know if anything changes
	if m.bufferList.NeedsUpdate() {
		cmds = append(cmds, m.editor.SendBuffersChangedMsg())
	}

	_, cmd := m.bufferList.Update(msg)
	cmds = append(cmds, cmd)

	if m.editor.ListBuffers {
		m.unfocusAllColumns()
	}

	// collect dirty buffers
	m.notesList.DirtyBuffers = m.editor.DirtyBuffers()

	return cmds
}

// updateStatusBar synchronises the status bar
// with the current component states and mode.
func (m *Model) updateStatusBar() {
	m.statusBar.DirTree = *m.dirTree
	m.statusBar.NotesList = *m.notesList
	m.statusBar.Editor = *m.editor

	currMode := m.editor.Vim.Mode.Current
	if currMode != mode.Normal {
		m.statusBar.Mode = currMode
	} else {
		m.statusBar.Mode = m.mode.Current
	}
}

// restoreState restores the state of the TUI from the last session
func (m *Model) restoreState() {
	currComp, err := m.conf.MetaValue("", config.CurrentComponent)
	colIndex := 1

	if err == nil && currComp != "" {
		index, _ := strconv.Atoi(currComp)
		colIndex = index
	}

	// focus notes list if there's not open note in meta conf but
	currentNote, err := m.conf.MetaValue("", config.LastOpenNote)
	if err == nil && currentNote == "" {
		colIndex = 2
	}

	m.focusColumn(colIndex)
}

// overlayPosition returns the top center position of the application screen
func (m *Model) overlayPosition(overlayWidth int) (int, int) {
	termW, _ := theme.TerminalSize()

	x := (termW / 2) - (overlayWidth / 2)
	y := 2

	return x, y
}

func (m *Model) componentsReady() bool {
	return m.dirTree.Ready && m.notesList.Ready && m.editor.Ready
}
