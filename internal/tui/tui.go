package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/textarea"
	bl "github.com/winder/bubblelayout"
)

const noNerdFonts = true

type notesList struct {
	id            bl.ID
	size          bl.Size
	isFocused     bool
	selectedIndex int
	content       string
}

type tuiModel struct {
	layout     bl.BubbleLayout
	editorID   bl.ID
	editorSize bl.Size

	keyInput keyInput
	textarea textarea.Model

	currentColumnFocus int
	// @todo this whole columns stuff seems strange
	// try to make it not strange or try to make it work without it
	columns       []any
	directoryTree *treeModel
}

func InitialModel() tuiModel {
	m := tuiModel{
		layout:             bl.New(),
		currentColumnFocus: 1,
	}

	// this is weird try to make it not weird
	newTree := newDirectoryTree()
	directoryTree := treeModel{
		id:            m.layout.Add("width 30"),
		isFocused:     true,
		selectedIndex: newTree.selectedIndex,
		dirsList:      newTree.dirsList,
		dirsListFlat:  newTree.dirsListFlat,
		content:       newTree.content,
	}

	notesList := notesList{
		id:        m.layout.Add("width 30"),
		isFocused: false,
		content:   "",
	}

	m.columns = []any{directoryTree, notesList}
	m.directoryTree = &directoryTree
	m.editorID = m.layout.Add("grow")
	m.keyInput = NewKeyInput()

	return m
}

func (m tuiModel) Init() tea.Cmd {
	return func() tea.Msg {
		return m.layout.Resize(80, 40)
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		m.handleKeyCombos(msg.String())

	case tea.WindowSizeMsg:
		// Convert WindowSizeMsg to BubbleLayoutMsg.
		return m, func() tea.Msg {
			return m.layout.Resize(msg.Width, msg.Height)
		}

	case bl.BubbleLayoutMsg:
		dTree := m.columns[0].(treeModel)
		dTree.size, _ = msg.Size(dTree.id)
		m.directoryTree = &dTree

		nList := m.columns[1].(notesList)
		nList.size, _ = msg.Size(nList.id)

		m.editorSize, _ = msg.Size(m.editorID)
		m.columns = []any{dTree, nList}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	return m, cmd
}

func (m tuiModel) GetTuiModel() tuiModel {
	return m
}

func baseColumnLayout(size bl.Size, focused bool) lipgloss.Style {
	var borderColour lipgloss.TerminalColor = lipgloss.NoColor{}
	if focused {
		borderColour = lipgloss.Color("#69c8dc")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColour).
		Foreground(lipgloss.NoColor{}).
		Width(size.Width).
		Height(size.Height - 2)
}

func (m tuiModel) View() string {
	t := textarea.New()
	t.Placeholder = "asdasd"
	t.Focus()

	dirTree := m.directoryTree
	notesList := m.columns[1].(notesList)

	return lipgloss.JoinHorizontal(0,
		baseColumnLayout(dirTree.size, dirTree.isFocused).
			Align(lipgloss.Left).
			Render(dirTree.content.String()),
		baseColumnLayout(notesList.size, notesList.isFocused).
			Align(lipgloss.Center).
			Render(notesList.content),
		baseColumnLayout(m.editorSize, false).
			Render(t.View()),
	)
}

func (m *tuiModel) handleKeyCombos(key string) {
	if key == "ctrl+w" {
		m.keyInput.isCtrlWDown = true
	}

	if m.keyInput.isCtrlWDown && strings.Contains(key, "ctrl+") {
		m.keyInput.keysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	m.keyInput.keysDown[key] = true

	actionString := mapToActionString(m.keyInput.keysDown)
	m.executeAction(actionString)

	// special key actions for cmd mode
	switch key {
	case ":":
		m.enterCmdMode()
	case "esc":
		m.exitCmdMode()
	case "enter":
		m.executeCmdModeCommand()
	}
	if key == ":" {
		m.enterCmdMode()
	}
	if key == "esc" {
		m.exitCmdMode()
	}
	if key == "enter" {
		m.executeCmdModeCommand()
	}

	if !m.keyInput.isCmdMode {
		m.keyInput.releaseKey(key)
	}
}

func (m *tuiModel) executeAction(keys string) {
	functions := map[string]func(){
		"focusNextColumn": m.focusNextColumn,
		"focusPrevColumn": m.focusPrevColumn,
		"moveUp":          m.moveUp,
		"moveDown":        m.moveDown,
		"collapse":        m.directoryTree.collapse,
		"expand":          m.directoryTree.expand,
	}

	for _, km := range m.keyInput.keyMaps {
		for combo, fnName := range km.action {
			if combo == keys {
				if fn, exists := functions[fnName]; exists {
					fn()
					m.resetKeysDown()
				}
				return
			}
		}
	}
}

func (m *tuiModel) resetKeysDown() {
	m.keyInput.isCtrlWDown = false
	m.keyInput.keysDown = make(map[string]bool)
}

func (m *tuiModel) focusNextColumn() {
	colIndex := min(m.currentColumnFocus+1, len(m.columns))
	dirTree := m.directoryTree
	notesList := m.columns[1].(notesList)

	switch colIndex {
	case 1:
		dirTree.isFocused = true
		notesList.isFocused = false
	case 2:
		dirTree.isFocused = false
		notesList.isFocused = true
	}

	m.columns[0] = dirTree
	m.columns[1] = notesList
	m.currentColumnFocus = colIndex
}

func (m *tuiModel) focusPrevColumn() {
	colIndex := m.currentColumnFocus - 1
	if colIndex < len(m.columns) {
		colIndex = 1
	}

	dirTree := m.directoryTree
	notesList := m.columns[1].(notesList)

	switch colIndex {
	case 1:
		dirTree.isFocused = true
		notesList.isFocused = false
	case 2:
		dirTree.isFocused = false
		notesList.isFocused = true
	}

	m.columns[0] = dirTree
	m.columns[1] = notesList
	m.currentColumnFocus = colIndex
}

func (m *tuiModel) moveUp() {
	dirTree := m.directoryTree
	//notesList := m.columns[1].(notesList)

	if dirTree.isFocused {
		dirTree.moveUp()
	}
}

func (m *tuiModel) moveDown() {
	dirTree := m.directoryTree
	//notesList := m.columns[1].(notesList)

	if dirTree.isFocused {
		dirTree.moveDown()
	}
}

// Toggle expand/collapse directory
func (m *tuiModel) toggleSelected() {
	dirTree := m.columns[0].(treeModel)
	if dirTree.isFocused {
		//index := dirTree.selectedIndex
		//node := dirTree.content.Children().At(index)
		//dirTree.content.Children = dir{name: node.name, open: !node.open, styles: node.styles}
	}
}

func (m *tuiModel) enterCmdMode() {
	m.keyInput.isCmdMode = true
}

func (m *tuiModel) exitCmdMode() {
	m.keyInput.isCmdMode = false
	m.resetKeysDown()
}

func (m *tuiModel) executeCmdModeCommand() {}

func (m *tuiModel) quit() {
	tea.Quit()
}
