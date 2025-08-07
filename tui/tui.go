package tui

import (
	"bellbird-notes/app/debug"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"
	"bellbird-notes/tui/vim"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	bl "github.com/winder/bubblelayout"
)

type Focusable = interfaces.Focusable

// Model is the Bubble Tea model for the TUI
type Model struct {
	// layout manages the spatial arrangement of TUI components.
	layout bl.BubbleLayout

	// keyInput handles user key sequences and maps them to actions.
	keyInput *keyinput.Input

	// app holds the state and behaviour of all core components
	app *components.App

	// vim provides Vim-style motions, commands, and focus logic.
	vim *vim.Vim
}

func InitialModel() *Model {
	layout := bl.New()
	vim := vim.New()
	app := components.NewApp(vim)
	vim.SetApp(app)

	m := Model{
		layout: layout,
		app:    app,
		vim:    vim,
	}

	// Initialise key input handler with Vim commands and components
	m.keyInput = keyinput.New(vim)
	m.app.KeyInput = m.keyInput

	m.keyInput.Components = []keyinput.FocusedComponent{
		m.app.DirTree,
		m.app.NotesList,
		m.app.Editor,
		m.app.BufferList,
	}

	m.vim.KeyMap = m.keyInput

	m.componentsInit()

	// Restore previous session state
	m.app.RestoreState()

	return &m
}

func (m Model) Init() tea.Cmd {
	editorCmd := m.app.Editor.Init()
	statusBarCmd := m.app.StatusBar.Init()

	return tea.Batch(editorCmd, statusBarCmd)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.keyInput.AllowSequences = !m.app.StatusBar.Focused
		statusMsg := m.keyInput.HandleSequences(msg.Key())

		if msg.Key().Code == 32 {
			// Reset spacebar key sequence
			cmds = append(
				cmds,
				m.keyInput.ResetSequence(),
			)
		}

		if msg.String() == "ctrl+c" {
			statusMsg = []message.StatusBarMsg{{
				Content: message.StatusBar.CtrlCExitNote,
				Type:    message.Success,
				Column:  sbc.General,
			}}
		}

		statusMsg = append(
			statusMsg,
			m.app.Editor.StatusBarInfo(),
		)

		var sbCmd tea.Cmd
		m.app.StatusBar, sbCmd = m.app.StatusBar.Update(statusMsg, msg)

		cmds = append(cmds, m.app.StatusBar.TeaCmd, sbCmd)

		for _, m := range statusMsg {
			cmds = append(cmds, m.Cmd)
		}

	case tea.WindowSizeMsg:
		m.app.DirTree.Update(msg)
		m.app.NotesList.Update(msg)
		m.app.Editor.Update(msg)
		m.app.BufferList.Update(msg)

		// Convert WindowSizeMsg to BubbleLayoutMsg.
		return m, func() tea.Msg {
			return m.layout.Resize(
				msg.Width,
				msg.Height,
			)
		}

	case bl.BubbleLayoutMsg:
		m.app.DirTree.Size, _ = msg.Size(m.app.DirTree.ID)
		m.app.NotesList.Size, _ = msg.Size(m.app.NotesList.ID)
		m.app.Editor.Size, _ = msg.Size(m.app.Editor.ID)
		m.app.BufferList.Size, _ = msg.Size(m.app.BufferList.ID)
		m.app.StatusBar.Size, _ = msg.Size(m.app.StatusBar.ID)

	case keyinput.ResetSequenceMsg:
		sb, cmd := m.app.StatusBar.Update(
			[]message.StatusBarMsg{m.keyInput.ResetKeysDown()},
			msg,
		)
		m.app.StatusBar = sb
		cmds = append(cmds, cmd)

	case components.BufferSavedMsg:
		// reload keymap if there's any updates
		if msg.Buffer.Path(false) == m.keyInput.KeyMap.Path() {
			m.keyInput.ReloadKeyMap()
		}

		if msg.Buffer.Path(false) == m.app.Conf.File() {
			m.RefreshUi()
		}

	case components.SearchConfirmedMsg:
		m.app.StatusBar.Update(nil, msg)
		m.app.Editor.Mode.Current = mode.Normal

	case components.SearchCancelMsg:
		m.app.Editor.CancelSearch()
		m.app.StatusBar.Mode = mode.Normal
		m.app.StatusBar.Update(nil, msg)

	case components.RefreshUiMsg:
		m.RefreshUi()
	}

	// exit programme when `:q` is entered in command prompt
	if m.app.ShouldQuit {
		if err := m.app.State.Write(); err != nil {
			debug.LogErr(err)
		}

		return m, tea.Quit
	}

	if m.app.Editor.ListBuffers {
		m.vim.UnfocusAllColumns()
	}

	cmds = append(cmds, m.app.UpdateComponents(msg)...)

	return m, tea.Batch(cmds...)
}

// View renders the TUI layout as a string
func (m Model) View() string {
	view := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.app.DirTree.View(),
			m.app.NotesList.View(),
			m.app.Editor.View(),
		),
		m.app.StatusBar.View(),
	)

	var (
		overlay = ""
		x       = 0
		y       = 0
	)

	// check if any overlays should be displayed
	if m.app.Editor.ListBuffers {
		overlay, x, y = m.vim.OverlayOpenBuffers()
	} else {
		m.app.BufferList.SetFocus(false)
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
	statusBarHeight := m.app.StatusBar.Height + reserverdLines

	m.app.DirTree.ID = m.layout.Add("width 30")

	m.app.NotesList.ID = m.layout.Add("width 30")
	m.app.NotesList.SetBuffers(&m.app.Buffers)

	m.app.Editor.ID = m.layout.Add("grow")
	m.app.Editor.SetBuffers(&m.app.Buffers)
	m.app.Editor.KeyInput = *m.keyInput

	m.app.BufferList.SetBuffers(&m.app.Buffers)

	m.app.StatusBar.ID = m.layout.Dock(bl.Dock{
		Cardinal:  bl.SOUTH,
		Preferred: statusBarHeight,
	})

	m.keyInput.FetchKeyMap(true)
	m.app.StatusBar.Commands = m.vim.CmdRegistry()
}

func (m *Model) RefreshUi() {
	m.app.Conf.Reload()

	m.app.DirTree.RefreshStyles()
	m.app.NotesList.RefreshStyles()
	m.app.Editor.RefreshTextAreaStyles()
	m.app.BufferList.RefreshStyles()

	m.updateEditorWidth()
}

func (m *Model) updateEditorWidth() {
	termW, _ := theme.TerminalSize()
	editorWidth := termW

	if m.app.DirTree.Visible {
		editorWidth -= m.app.DirTree.Size.Width
	}

	if m.app.NotesList.Visible {
		editorWidth -= m.app.NotesList.Size.Width
	}

	m.app.Editor.SetWidth(editorWidth)
}
