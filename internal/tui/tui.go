package tui

import (
	"bellbird-notes/internal/tui/directorytree"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/noteslist"
	"bellbird-notes/internal/tui/statusbar"
	"bellbird-notes/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/textarea"
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
	layout     bl.BubbleLayout
	editorID   bl.ID
	editorSize bl.Size
	mode       *mode.ModeInstance

	keyInput *KeyInput
	textarea textarea.Model

	currentColumnFocus int
	directoryTree      *directorytree.DirectoryTree
	notesList          *noteslist.NotesList
	statusBar          *statusbar.StatusBar
}

func InitialModel() TuiModel {
	m := TuiModel{
		layout:             bl.New(),
		currentColumnFocus: 1,
		mode:               mode.New(),
		directoryTree:      directorytree.New(),
		notesList:          noteslist.New(),
		statusBar:          statusbar.New(),
	}

	m.layout = bl.New()
	m.editorID = m.layout.Add("grow")

	m.directoryTree.Id = m.layout.Add("width 30")
	m.directoryTree.IsFocused = true

	m.notesList.Id = m.layout.Add("width 30")
	m.notesList.IsFocused = false

	m.currentColumnFocus = 1

	m.keyInput = NewKeyInput()
	m.keyInput.functions = m.KeyInputFn()

	return m
}

func (m TuiModel) Init() tea.Cmd {
	resizeCmd := func() tea.Msg {
		return m.layout.Resize(80, 40)
	}

	return tea.Batch(resizeCmd)
}

func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.textarea, cmd = m.textarea.Update(msg)

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
		m.notesList.Update(msg)
		return m, func() tea.Msg {
			return m.layout.Resize(msg.Width, msg.Height)
		}

	case bl.BubbleLayoutMsg:
		m.directoryTree.Size, _ = msg.Size(m.directoryTree.Id)
		m.notesList.Size, _ = msg.Size(m.notesList.Id)
		m.editorSize, _ = msg.Size(m.editorID)
	case messages.StatusBarMsg:
		m.statusBar = m.statusBar.Update(msg, msg)
	}

	m.keyInput.mode = m.mode.Current
	m.directoryTree.Mode = m.mode.Current
	m.directoryTree.Update(msg)

	_, notesCmd := m.notesList.Update(msg)
	cmds = append(cmds, notesCmd)

	m.statusBar.DirTree = *m.directoryTree

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m TuiModel) GetTuiModel() TuiModel {
	return m
}

func (m TuiModel) View() string {
	t := textarea.New()
	t.Placeholder = "asdasd"
	t.Focus()

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Bottom,
			m.directoryTree.View(),
			m.notesList.View(),
			theme.BaseColumnLayout(m.editorSize, false).Render(t.View()),
		),
		m.statusBar.View(),
	)
}

func (m *TuiModel) focusNextColumn() messages.StatusBarMsg {
	colIndex := min(m.currentColumnFocus+1, 3)
	dirTree := m.directoryTree
	notesList := m.notesList

	switch colIndex {
	case 1:
		dirTree.IsFocused = true
		notesList.IsFocused = false
	case 2:
		dirTree.IsFocused = false
		notesList.IsFocused = true
	}

	m.currentColumnFocus = colIndex
	return messages.StatusBarMsg{}
}

func (m *TuiModel) focusPrevColumn() messages.StatusBarMsg {
	colIndex := m.currentColumnFocus - 1
	if colIndex < 3 {
		colIndex = 1
	}

	dirTree := m.directoryTree
	notesList := m.notesList

	switch colIndex {
	case 1:
		dirTree.IsFocused = true
		notesList.IsFocused = false
	case 2:
		dirTree.IsFocused = false
		notesList.IsFocused = true
	}

	m.currentColumnFocus = colIndex
	return messages.StatusBarMsg{}
}

func (m *TuiModel) moveUp() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if dirTree.IsFocused {
		return dirTree.MoveUp()
	}
	if notesList.IsFocused {
		return notesList.LineUp()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) moveDown() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if dirTree.IsFocused {
		return dirTree.MoveDown()
	}
	if notesList.IsFocused {
		return notesList.LineDown()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) createDir() messages.StatusBarMsg {
	dirTree := m.directoryTree
	m.mode.Current = mode.Insert

	if dirTree.IsFocused {
		m.statusBar.IsFocused = false
		return dirTree.Create()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) rename() messages.StatusBarMsg {
	dirTree := m.directoryTree
	m.mode.Current = mode.Insert

	if dirTree.IsFocused {
		m.statusBar.IsFocused = false
		return dirTree.Rename()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) remove() messages.StatusBarMsg {
	dirTree := m.directoryTree
	// go into insert mode because we always ask for
	// confirmation before deleting anything
	m.mode.Current = mode.Insert

	if dirTree.IsFocused {
		m.statusBar.IsFocused = true
		return dirTree.ConfirmRemove()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) goToTop() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if m.mode.Current == mode.Normal {
		if dirTree.IsFocused {
			return dirTree.GoToTop()
		}
		if notesList.IsFocused {
			return notesList.GoToTop()
		}
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) goToBottom() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	if m.mode.Current == mode.Normal {
		if dirTree.IsFocused {
			return dirTree.GoToBottom()
		}
		if notesList.IsFocused {
			return notesList.GoToBottom()
		}
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) confirmAction() messages.StatusBarMsg {
	dirTree := m.directoryTree
	notesList := m.notesList

	statusMsg := messages.StatusBarMsg{}

	if m.statusBar.IsFocused {
		statusMsg = m.statusBar.ConfirmAction()
	}

	if dirTree.IsFocused {
		if m.mode.Current != mode.Normal {
			statusMsg = dirTree.ConfirmAction()
		} else {
			notesList.CurrentPath = dirTree.SelectedDir().Path
			statusMsg = notesList.Refresh()
		}
	}

	m.mode.Current = mode.Normal
	return statusMsg
}

func (m *TuiModel) cancelAction() messages.StatusBarMsg {
	dirTree := m.directoryTree
	m.mode.Current = mode.Normal

	if dirTree.IsFocused {
		m.statusBar.IsFocused = false
		return dirTree.CancelAction()
	}
	return messages.StatusBarMsg{}
}

func (m *TuiModel) enterCmdMode() {
	m.mode.Current = mode.Command
}

func (m *TuiModel) exitCmdMode() {
	m.mode.Current = mode.Normal
	m.keyInput.resetKeysDown()
}

func (m *TuiModel) executeCmdModeCommand() {}

func (m *TuiModel) quit() {
	tea.Quit()
}

func (m *TuiModel) KeyInputFn() map[string]func() messages.StatusBarMsg {
	return map[string]func() messages.StatusBarMsg{
		"focusNextColumn": m.focusNextColumn,
		"focusPrevColumn": m.focusPrevColumn,
		"moveUp":          m.moveUp,
		"moveDown":        m.moveDown,
		"collapse":        m.directoryTree.Collapse,
		"expand":          m.directoryTree.Expand,
		"createDir":       m.createDir,
		"rename":          m.rename,
		"delete":          m.remove,
		"goToTop":         m.goToTop,
		"goToBottom":      m.goToBottom,
		"cancelAction":    m.cancelAction,
		"confirmAction":   m.confirmAction,
	}
}
