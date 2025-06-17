package tui

import (
	"bellbird-notes/app/config"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
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

	dirTree   *components.DirectoryTree
	notesList *components.NotesList
	editor    *components.Editor
	statusBar *components.StatusBar
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
		statusBar:    components.NewStatusBar(),
	}

	m.keyInput.Functions = m.KeyInputFn()
	m.componentsInit()

	return &m
}

func (m Model) Init() tea.Cmd {
	editorCmd := m.editor.Init()
	statusBarCmd := m.statusBar.Init()

	return tea.Batch(editorCmd, statusBarCmd)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		statusMsg := m.keyInput.HandleSequences(msg.String())

		if msg.String() == "ctrl+c" {
			statusMsg = []message.StatusBarMsg{{
				Content: message.StatusBar.CtrlCExitNote,
				Type:    message.Success,
				Column:  sbc.General,
			}}
		}

		statusMsg = append(statusMsg, m.editor.StatusBarInfo())

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
		m.dirTree.Size, _ = msg.Size(m.dirTree.Id)
		m.notesList.Size, _ = msg.Size(m.notesList.Id)
		m.editor.Size, _ = msg.Size(m.editor.Id)
	}

	m.keyInput.Mode = m.mode.Current

	// exit programme when `:q` is entered in command prompt
	if m.statusBar.ShouldQuit {
		return m, tea.Quit
	}

	cmds = m.updateComponents(msg)
	m.updateStatusBar()

	return m, tea.Batch(cmds...)
}

// View renders the TUI layout as a string
func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.dirTree.View(),
			m.notesList.View(),
			m.editor.View(),
		),
		m.statusBar.View(),
	)
}

// componentsInit registers components in the layout
// and sets initial focus
func (m *Model) componentsInit() {
	m.dirTree.Id = m.layout.Add("w 30")
	m.notesList.Id = m.layout.Add("w 30")
	m.editor.Id = m.layout.Add("grow")
	m.focusColumn(1)
}

// updateComponents dispatches updates to the focused components
// (directory tree, notes list, editor), updates the current editor mode
func (m *Model) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

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
		m.notesList.DirtyBuffers = m.editor.DirtyBuffers()
	}

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
