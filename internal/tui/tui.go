package tui

import (
	"bellbird-notes/internal/tui/directorytree"
	"bellbird-notes/internal/tui/mode"
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
	notesList          *notesList
}

func InitialModel() TuiModel {
	m := TuiModel{
		layout:             bl.New(),
		currentColumnFocus: 1,
		mode:               mode.New(),
		directoryTree:      directorytree.New(),
		notesList:          &notesList{},
	}

	m.layout = bl.New()
	m.editorID = m.layout.Add("grow")

	m.directoryTree.Id = m.layout.Add("width 30")
	m.directoryTree.IsFocused = true

	m.notesList = &notesList{}
	m.notesList.id = m.layout.Add("width 30")
	m.notesList.isFocused = false
	m.notesList.content = ""

	m.currentColumnFocus = 1

	m.keyInput = NewKeyInput()
	m.keyInput.functions = m.KeyInputFn()

	return m
}

func (m TuiModel) Init() tea.Cmd {
	return func() tea.Msg {
		return m.layout.Resize(80, 40)
	}
}

func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		m.keyInput.handleKeyCombos(msg.String())

	case tea.WindowSizeMsg:
		// Convert WindowSizeMsg to BubbleLayoutMsg.
		return m, func() tea.Msg {
			return m.layout.Resize(msg.Width, msg.Height)
		}

	case bl.BubbleLayoutMsg:
		m.directoryTree.Size, _ = msg.Size(m.directoryTree.Id)
		m.notesList.size, _ = msg.Size(m.notesList.id)
		m.editorSize, _ = msg.Size(m.editorID)
	}

	m.keyInput.mode = m.mode.Current
	m.directoryTree.Mode = m.mode.Current

	m.directoryTree.Update(msg)

	return m, cmd
}

func (m TuiModel) GetTuiModel() TuiModel {
	return m
}

func (m TuiModel) View() string {
	t := textarea.New()
	t.Placeholder = "asdasd"
	t.Focus()

	notesList := m.notesList

	termWidth, _ := theme.GetTerminalSize()
	footerStyle := lipgloss.NewStyle().
		//Border(lipgloss.RoundedBorder(), true).
		//Background(lipgloss.Color("#424B5D")).
		Align(lipgloss.Center).
		Height(1).
		Width(termWidth)
	footerContent := "Press <space> to toggle the modal window. Press q or <esc> to quit."
	footer := footerStyle.Render(footerContent)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Bottom,
			m.directoryTree.View(),
			theme.BaseColumnLayout(notesList.size, notesList.isFocused).Align(lipgloss.Center).Render(notesList.content),
			theme.BaseColumnLayout(m.editorSize, false).Render(t.View()),
		),
		footer,
	)
}

func (m *TuiModel) focusNextColumn() {
	colIndex := min(m.currentColumnFocus+1, 3)
	dirTree := m.directoryTree
	notesList := m.notesList

	switch colIndex {
	case 1:
		dirTree.IsFocused = true
		notesList.isFocused = false
	case 2:
		dirTree.IsFocused = false
		notesList.isFocused = true
	}

	m.currentColumnFocus = colIndex
}

func (m *TuiModel) focusPrevColumn() {
	colIndex := m.currentColumnFocus - 1
	if colIndex < 3 {
		colIndex = 1
	}

	dirTree := m.directoryTree
	notesList := m.notesList

	switch colIndex {
	case 1:
		dirTree.IsFocused = true
		notesList.isFocused = false
	case 2:
		dirTree.IsFocused = false
		notesList.isFocused = true
	}

	m.currentColumnFocus = colIndex
}

func (m *TuiModel) moveUp() {
	dirTree := m.directoryTree

	if dirTree.IsFocused {
		dirTree.MoveUp()
	}
}

func (m *TuiModel) moveDown() {
	dirTree := m.directoryTree

	if dirTree.IsFocused {
		dirTree.MoveDown()
	}
}

func (m *TuiModel) rename() {
	dirTree := m.directoryTree
	m.mode.Current = mode.Insert

	if dirTree.IsFocused {
		dirTree.Rename()
	}
}

func (m *TuiModel) confirmAction() {
	dirTree := m.directoryTree
	m.mode.Current = mode.Normal

	if dirTree.IsFocused {
		dirTree.ConfirmAction()
	}
}

func (m *TuiModel) cancelAction() {
	dirTree := m.directoryTree
	m.mode.Current = mode.Normal

	if dirTree.IsFocused {
		dirTree.CancelAction()
	}
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

func (m *TuiModel) KeyInputFn() map[string]func() {
	return map[string]func(){
		"focusNextColumn": m.focusNextColumn,
		"focusPrevColumn": m.focusPrevColumn,
		"moveUp":          m.moveUp,
		"moveDown":        m.moveDown,
		"collapse":        m.directoryTree.Collapse,
		"expand":          m.directoryTree.Expand,
		"rename":          m.rename,
		"cancelAction":    m.cancelAction,
		"confirmAction":   m.confirmAction,
	}
}
