package tui

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/components"
	"bellbird-notes/internal/tui/messages"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	bl "github.com/winder/bubblelayout"
)

const noNerdFonts = false

type notesList struct {
	id            bl.ID
	size          bl.Size
	isFocused     bool
	selectedIndex int
	content       string
}

type TuiModel struct {
	layout bl.BubbleLayout
	mode   *app.ModeInstance

	keyInput *KeyInput

	currentColumnFocus int

	directoryTree *components.DirectoryTree
	notesList     *components.NotesList
	editor        *components.Editor
	statusBar     *components.StatusBar
}

func InitialModel() TuiModel {
	m := TuiModel{
		layout:             bl.New(),
		currentColumnFocus: 1,
		mode:               &app.ModeInstance{Current: app.NormalMode},
		directoryTree:      components.NewDirectoryTree(),
		notesList:          components.NewNotesList(),
		editor:             components.NewEditor(),
		statusBar:          components.NewStatusBar(),
	}

	m.layout = bl.New()

	m.currentColumnFocus = 1

	m.directoryTree.Id = m.layout.Add("width 30")
	m.directoryTree.Focused = true

	m.notesList.Id = m.layout.Add("width 30")
	m.notesList.Focused = false

	m.editor.Id = m.layout.Add("grow")
	m.editor.Focused = false

	m.keyInput = NewKeyInput()
	m.keyInput.functions = m.KeyInputFn()

	return m
}

func (m TuiModel) Init() tea.Cmd {
	resizeCmd := func() tea.Msg {
		return m.layout.Resize(80, 40)
	}

	editorCmd := m.editor.Init()

	return tea.Batch(resizeCmd, editorCmd)
}

func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		statusMsg := m.keyInput.handleKeyCombos(msg.String())
		m.statusBar = m.statusBar.Update(statusMsg, msg)
		m.statusBar.Mode = m.mode.Current

	case tea.WindowSizeMsg:
		// Convert WindowSizeMsg to BubbleLayoutMsg.
		m.directoryTree.Update(msg)
		m.notesList.Update(msg)
		m.editor.Update(msg)

		return m, func() tea.Msg {
			return m.layout.Resize(msg.Width, msg.Height)
		}

	case bl.BubbleLayoutMsg:
		m.directoryTree.Size, _ = msg.Size(m.directoryTree.Id)
		m.notesList.Size, _ = msg.Size(m.notesList.Id)
		m.editor.Size, _ = msg.Size(m.editor.Id)

	case messages.StatusBarMsg:
		m.statusBar = m.statusBar.Update(msg, msg)
	}

	m.keyInput.mode = m.mode.Current
	var dirTreeCmd, notesCmd, editorCmd tea.Cmd

	if m.directoryTree.Focused {
		m.directoryTree.Mode = m.mode.Current
		_, dirTreeCmd = m.directoryTree.Update(msg)
	}
	if m.notesList.Focused {
		m.notesList.Mode = m.mode.Current
		_, notesCmd = m.notesList.Update(msg)
	}
	if m.editor.Focused {
		_, editorCmd = m.editor.Update(msg)
		m.keyInput.mode = m.editor.Mode.Current
	}

	m.statusBar.DirTree = *m.directoryTree
	m.statusBar.NotesList = *m.notesList
	m.statusBar.Editor = *m.editor

	cmds = append(cmds, cmd, notesCmd, dirTreeCmd, editorCmd)

	return m, tea.Batch(cmds...)
}

func (m TuiModel) GetTuiModel() TuiModel {
	return m
}

func (m TuiModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.directoryTree.View(),
			m.notesList.View(),
			m.editor.View(),
		),
		m.statusBar.View(),
	)
}

///
/// Keyboard shortcut commands
///

func (m *TuiModel) focusColumn(index int) messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList
	editor := m.editor

	dirTree.Focused = false
	notesList.Focused = false
	editor.Focused = false

	switch index {
	case 1:
		dirTree.Focused = true
	case 2:
		notesList.Focused = true
	case 3:
		editor.Focused = true
		//default:
		//	editor.ExitInsertMode()
	}

	m.currentColumnFocus = index

	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusDirectoryTree() messages.StatusBarMsg {
	m.focusColumn(1)
	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusNotesList() messages.StatusBarMsg {
	m.focusColumn(2)
	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusEditor() messages.StatusBarMsg {
	m.focusColumn(3)
	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusNextColumn() messages.StatusBarMsg {
	index := min(m.currentColumnFocus+1, 3)
	m.focusColumn(index)
	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusPrevColumn() messages.StatusBarMsg {
	index := m.currentColumnFocus - 1
	if index < 0 {
		index = 1
	}
	m.focusColumn(index)
	return messages.StatusBarMsg{}
}

func (m *TuiModel) lineUp() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if dirTree.Focused {
		return dirTree.LineUp()
	}
	if notesList.Focused {
		return notesList.LineUp()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) lineDown() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList
	statusMsg := messages.StatusBarMsg{}

	if dirTree.Focused {
		statusMsg = dirTree.LineDown()
		statusMsg.Content = strconv.Itoa(dirTree.SelectedDir().NbrFolders) + " folders"
	}
	if notesList.Focused {
		return notesList.LineDown()
	}
	return statusMsg
}

func (m *TuiModel) createDir() messages.StatusBarMsg {
	dirTree := m.directoryTree

	if dirTree.Focused {
		m.mode.Current = app.InsertMode
		m.statusBar.Focused = false
		return dirTree.Create()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) createNote() messages.StatusBarMsg {
	notesList := m.notesList

	if notesList.Focused {
		m.mode.Current = app.InsertMode
		m.statusBar.Focused = false
		return notesList.Create()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) rename() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if dirTree.Focused {
		m.mode.Current = app.InsertMode
		m.statusBar.Focused = false
		return dirTree.Rename(dirTree.SelectedDir().Name)
	}

	if notesList.Focused {
		m.mode.Current = app.InsertMode
		m.statusBar.Focused = false
		return notesList.Rename(notesList.SelectedItem(nil).Name)
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) remove() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList
	// go into insert mode because we always ask for
	// confirmation before deleting anything
	m.mode.Current = app.InsertMode

	if dirTree.Focused {
		m.statusBar.Focused = true
		return dirTree.ConfirmRemove()
	}
	if notesList.Focused {
		m.statusBar.Focused = true
		return notesList.ConfirmRemove()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) goToTop() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if m.mode.Current == app.NormalMode {
		if dirTree.Focused {
			return dirTree.GoToTop()
		}
		if notesList.Focused {
			return notesList.GoToTop()
		}
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) goToBottom() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if m.mode.Current == app.NormalMode {
		if dirTree.Focused {
			return dirTree.GoToBottom()
		}
		if notesList.Focused {
			return notesList.GoToBottom()
		}
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) confirmAction() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList
	editor := m.editor
	statusMsg := messages.StatusBarMsg{}

	if dirTree.Focused {
		if m.mode.Current != app.NormalMode {
			statusMsg = dirTree.ConfirmAction()
		} else {
			notesList.CurrentPath = dirTree.SelectedDir().Path
			statusMsg = notesList.Refresh(true)
		}
	}

	if notesList.Focused {
		if m.mode.Current != app.NormalMode {
			statusMsg = notesList.ConfirmAction()
		} else {
			notePath := notesList.SelectedItem(nil).GetPath()
			editor.NewBuffer(notePath)
		}
	}

	if m.statusBar.Focused {
		statusMsg = m.statusBar.ConfirmAction(statusMsg.Sender)
	}

	m.mode.Current = app.NormalMode
	return statusMsg
}

func (m *TuiModel) cancelAction() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList
	editor := m.editor
	m.mode.Current = app.NormalMode
	m.statusBar.Focused = false

	if dirTree.Focused {
		return dirTree.CancelAction(func() { dirTree.Refresh() })
	}
	if notesList.Focused {
		return notesList.CancelAction(func() { notesList.Refresh(false) })
	}
	if editor.Focused {
		return editor.ExitInsertMode()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) enterCmdMode() {
	m.mode.Current = app.CommandMode
}

func (m *TuiModel) exitCmdMode() {
	m.EnterNormalMode()
	m.keyInput.resetKeysDown()
}

func (m *TuiModel) EnterNormalMode() {
	m.mode.Current = app.NormalMode
	m.editor.ExitInsertMode()
}

func (m *TuiModel) executeCmdModeCommand() {}

func (m *TuiModel) quit() {
	tea.Quit()
}

func (m *TuiModel) KeyInputFn() map[string]func() messages.StatusBarMsg {
	return map[string]func() messages.StatusBarMsg{
		"focusDirectoryTree": m.focusDirectoryTree,
		"focusNotesList":     m.focusNotesList,
		"focusEditor":        m.focusEditor,
		"focusNextColumn":    m.focusNextColumn,
		"focusPrevColumn":    m.focusPrevColumn,
		"lineUp":             m.lineUp,
		"lineDown":           m.lineDown,
		"collapse":           m.directoryTree.Collapse,
		"expand":             m.directoryTree.Expand,
		"createDir":          m.createDir,
		"createNote":         m.createNote,
		"rename":             m.rename,
		"delete":             m.remove,
		"goToTop":            m.goToTop,
		"goToBottom":         m.goToBottom,
		"cancelAction":       m.cancelAction,
		"confirmAction":      m.confirmAction,
		"enterInsertMode":    m.editor.EnterInsertMode,
	}
}
