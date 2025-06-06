package tui

import (
	"strconv"

	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	sbc "bellbird-notes/tui/types/statusbar_column"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

func InitialModel() Model {
	layout := bl.New()

	mode := &mode.ModeInstance{
		Current: mode.Normal,
	}

	m := Model{
		layout:       layout,
		mode:         mode,
		currColFocus: 1,
		keyInput:     keyinput.New(),
		dirTree:      components.NewDirectoryTree(),
		notesList:    components.NewNotesList(),
		editor:       components.NewEditor(),
		statusBar:    components.NewStatusBar(),
	}

	m.keyInput.Functions = m.KeyInputFn()
	m.componentsInit()

	return m
}

func (m Model) Init() tea.Cmd {
	resizeCmd := func() tea.Msg {
		return m.layout.Resize(80, 40)
	}

	editorCmd := m.editor.Init()
	statusBarCmd := m.statusBar.Init()

	return tea.Batch(resizeCmd, editorCmd, statusBarCmd)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		statusMsg := m.keyInput.HandleSequences(msg.String())

		if msg.String() == "ctrl+c" {
			statusMsg = message.StatusBarMsg{
				Content: message.StatusBar.CtrlCExitNote,
				Type:    message.Success,
				Column:  sbc.General,
			}
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
		m.dirTree.Size, _ = msg.Size(m.dirTree.Id)
		m.notesList.Size, _ = msg.Size(m.notesList.Id)
		m.editor.Size, _ = msg.Size(m.editor.Id)

	case message.StatusBarMsg:
		m.statusBar = m.statusBar.Update(msg, msg)
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
	m.dirTree.Id = m.layout.Add("width 30")
	m.dirTree.Focused = true

	m.notesList.Id = m.layout.Add("width 30")
	m.notesList.Focused = false

	m.editor.Id = m.layout.Add("grow")
	m.editor.Focused = false
}

// updateComponents dispatches updates to the focused components
// (directory tree, notes list, editor), updates the current editor mode
func (m *Model) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	if m.dirTree.Focused {
		m.dirTree.Mode = m.mode.Current
		_, cmd := m.dirTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.notesList.Focused {
		m.notesList.Mode = m.mode.Current
		_, cmd := m.notesList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.editor.Focused {
		_, cmd := m.editor.Update(msg)
		cmds = append(cmds, cmd)
		m.keyInput.Mode = m.editor.Vim.Mode.Current
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

///
/// Keyboard shortcut delegations
///

// focusColumn selects and higlights a column with index `index`
// (1=dirTree, 2=notesList, 3=editor)
func (m *Model) focusColumn(index int) message.StatusBarMsg {
	m.dirTree.Focused = index == 1
	m.notesList.Focused = index == 2
	m.editor.Focused = index == 3
	m.currColFocus = index

	return message.StatusBarMsg{}
}

// focusDirectoryTree is a helper function
// for selecting the directory tree
func (m *Model) focusDirectoryTree() message.StatusBarMsg {
	return m.focusColumn(1)
}

// focusNotesList() is a helper function
// for selecting the notes list
func (m *Model) focusNotesList() message.StatusBarMsg {
	return m.focusColumn(2)
}

// focusEditor is a helper function
// for selecting the editor
func (m *Model) focusEditor() message.StatusBarMsg {
	return m.focusColumn(3)
}

// focusNextColumn selects and highlights the respectivley next of the
// currently selected column.
// Selects the first if the currently selected column is the last column...
func (m *Model) focusNextColumn() message.StatusBarMsg {
	index := min(m.currColFocus+1, 3)
	return m.focusColumn(index)
}

// focusNextColumn selects and highlights the respectivley next of the
// currently selected column.
// Selects the first if the currently selected column is the last column...
func (m *Model) focusPrevColumn() message.StatusBarMsg {
	index := m.currColFocus - 1
	if index < 0 {
		index = 1
	}
	return m.focusColumn(index)
}

// focusedComponent returns the component that is currently focused
func (m *Model) focusedComponent() Focusable {
	if m.dirTree.Focused {
		return m.dirTree
	}
	if m.notesList.Focused {
		return m.notesList
	}
	return nil
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (m *Model) lineUp() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if f := m.focusedComponent(); f != nil {
		statusMsg = f.LineUp()
		statusMsg.Content = strconv.Itoa(m.nbrFolders()) + " Folders"
	}

	return statusMsg
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (m *Model) lineDown() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if f := m.focusedComponent(); f != nil {
		statusMsg = f.LineDown()
		statusMsg.Content = strconv.Itoa(m.nbrFolders()) + " Folders"
	}

	return statusMsg
}

// nbrFolders returns the number of folders
// in the currently selected directory
func (m *Model) nbrFolders() int {
	return m.dirTree.SelectedDir().NbrFolders
}

// createDir enters insert mode
// and triggers directory creation
func (m *Model) createDir() message.StatusBarMsg {
	return m.dirTree.Create(m.mode, m.statusBar)
}

// createNote enters insert mode
// and triggers notes creation
func (m *Model) createNote() message.StatusBarMsg {
	return m.notesList.Create(m.mode, m.statusBar)
}

// rename enters insert mode and renames the selected item
// in the directory or note list
func (m *Model) rename() message.StatusBarMsg {
	if m.dirTree.Focused || m.notesList.Focused {
		m.mode.Current = mode.Insert
		m.statusBar.Focused = false
	}

	if m.dirTree.Focused {
		return m.dirTree.Rename(
			m.dirTree.SelectedDir().Name(),
		)
	}

	if m.notesList.Focused {
		return m.notesList.Rename(
			m.notesList.SelectedItem(nil).Name(),
		)
	}
	return message.StatusBarMsg{}
}

// remove enters insert mode and triggers a delete confirmation
// for the focused component
func (m *Model) remove() message.StatusBarMsg {
	// go into insert mode because we always ask for
	// confirmation before deleting anything
	m.mode.Current = mode.Insert

	if f := m.focusedComponent(); f != nil {
		m.statusBar.Focused = true
		return f.ConfirmRemove()
	}
	return message.StatusBarMsg{}
}

// goToTop moves the focused list to its first item
func (m *Model) goToTop() message.StatusBarMsg {
	if f := m.focusedComponent(); f != nil {
		return f.GoToTop()
	}
	return message.StatusBarMsg{}
}

// goToTop moves the focused list to its last item
func (m *Model) goToBottom() message.StatusBarMsg {
	if f := m.focusedComponent(); f != nil {
		return f.GoToBottom()
	}
	return message.StatusBarMsg{}
}

// confirmAction performs the primary action for the focused component,
// or loads note data into the editor if in normal mode.
func (m *Model) confirmAction() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	f := m.focusedComponent()

	if m.statusBar.Focused {
		statusMsg = m.statusBar.ConfirmAction(
			statusMsg.Sender,
			f,
			*m.editor,
		)
	}

	if m.mode.Current != mode.Normal && !m.statusBar.Focused {
		statusMsg = f.ConfirmAction()
	} else {
		// only open stuff if we're in normal mode
		if m.mode.Current != mode.Normal {
			m.mode.Current = mode.Normal
			return statusMsg
		}

		if f == m.dirTree {
			m.notesList.CurrentPath = m.dirTree.SelectedDir().Path()
			statusMsg = m.notesList.Refresh(true)
		}

		if f == m.notesList {
			notePath := m.notesList.SelectedItem(nil).Path()
			m.editor.OpenBuffer(notePath)
		}
	}

	m.mode.Current = mode.Normal
	return statusMsg
}

// cancelAction resets mode to normal
// and cancels pending actions in the focused component.
func (m *Model) cancelAction() message.StatusBarMsg {
	m.mode.Current = mode.Normal
	m.statusBar.Focused = false

	if m.statusBar.Prompt.Focused() {
		m.statusBar.CancelAction(func() {})
	} else {
		if f := m.focusedComponent(); f != nil {
			resetIndex := false
			stateCreate := components.EditStates.Create

			if m.dirTree.EditState == stateCreate ||
				m.notesList.EditState == stateCreate {
				resetIndex = true
			}

			return f.CancelAction(func() {
				f.Refresh(resetIndex)
			})
		}
	}

	return message.StatusBarMsg{}
}

func (m *Model) enterCmdMode() message.StatusBarMsg {
	if m.mode.Current != mode.Normal {
		return message.StatusBarMsg{}
	}

	m.mode.Current = mode.Command
	m.statusBar.Focused = true
	m.statusBar.Type = message.Prompt
	statusMsg := message.StatusBar.CmdPrompt

	return message.StatusBarMsg{
		Content: statusMsg,
		Type:    message.Prompt,
		Column:  sbc.General,
	}
}

//func (m *TuiModel) exitCmdMode() {
//	m.mode.Current = mode.NormalMode
//	m.keyInput.ResetKeysDown()
//}

//func (m *TuiModel) executeCmdModeCommand() {}

//func (m *TuiModel) quit() {
//	tea.Quit()
//}

// KeyInputFn maps command strings to actions for key sequence input.
func (m *Model) KeyInputFn() map[string]func() message.StatusBarMsg {
	return map[string]func() message.StatusBarMsg{
		"focusDirectoryTree": m.focusDirectoryTree,
		"focusNotesList":     m.focusNotesList,
		"focusEditor":        m.focusEditor,
		"focusNextColumn":    m.focusNextColumn,
		"focusPrevColumn":    m.focusPrevColumn,
		"lineUp":             m.lineUp,
		"lineDown":           m.lineDown,
		"collapse":           m.dirTree.Collapse,
		"expand":             m.dirTree.Expand,
		"createDir":          m.createDir,
		"createNote":         m.createNote,
		"rename":             m.rename,
		"delete":             m.remove,
		"goToTop":            m.goToTop,
		"goToBottom":         m.goToBottom,
		"cancelAction":       m.cancelAction,
		"confirmAction":      m.confirmAction,
		"enterCmdMode":       m.enterCmdMode,
	}
}
