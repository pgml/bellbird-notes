package tui

import (
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components"
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"
)

type c = ki.FocusedComponent
type keyAction = ki.KeyAction
type keyCond = ki.KeyCondition
type binding = ki.KeyBinding

const (
	n = mode.Normal
	v = mode.Visual
)

func (m *Model) KeyInputFn() []ki.KeyAction {
	return []keyAction{
		// LINE DOWN
		{
			Bindings: ki.KeyBindings("j"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.lineDown,
				},
				m.editorInputAction(n, m.editor.LineDown),
				m.editorInputAction(v, m.editor.LineDown),
			},
		},

		// LINE UP
		{
			Bindings: ki.KeyBindings("k"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.lineUp,
				},
				m.editorInputAction(n, m.editor.LineUp),
				m.editorInputAction(v, m.editor.LineUp),
			},
		},

		// TREE COLLAPSE, EDITOR CHARACTER LEFT
		{
			Bindings: ki.KeyBindings("h"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree},
					Action:     m.dirTree.Collapse,
				},
				m.editorInputAction(n, m.editor.MoveCharacterLeft),
				m.editorInputAction(v, m.editor.MoveCharacterLeft),
			},
		},

		// TREE EXPAND, EDITOR CHARACTER RIGHT
		{
			Bindings: ki.KeyBindings("l"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree},
					Action:     m.dirTree.Expand,
				},
				m.editorInputAction(n, m.editor.MoveCharacterRight),
				m.editorInputAction(v, m.editor.MoveCharacterRight),
			},
		},

		// CREATE DIR
		{
			Bindings: ki.KeyBindings("d", "n"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree},
				Action:     m.createDir,
			}},
		},

		// DELETE DIR/NOTE
		{
			Bindings: ki.KeyBindings("D"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.remove,
			}},
		},

		// CREATE NOTE
		{
			Bindings: ki.KeyBindings("%", "n"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.notesList},
				Action:     m.createNote,
			}},
		},

		// TREE/NOTE RENAME
		{
			Bindings: ki.KeyBindings("R"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.rename,
			}},
		},

		{
			Bindings: ki.KeyBindings("gg"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.goToTop,
				},
				m.editorInputAction(mode.Normal, m.editor.GoToTop),
			},
		},
		{
			Bindings: ki.KeyBindings("G"),
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.goToBottom,
				},
				m.editorInputAction(mode.Normal, m.editor.GoToBottom),
			},
		},

		{
			Bindings: ki.KeyBindings("i"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.EnterInsertMode)},
		},
		{
			Bindings: ki.KeyBindings("I"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.InsertLineStart)},
		},
		{
			Bindings: ki.KeyBindings("a"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.InsertAfter)},
		},
		{
			Bindings: ki.KeyBindings("A"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.InsertLineEnd)},
		},
		{
			Bindings: ki.KeyBindings("r"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.EnterReplaceMode)},
		},
		{
			Bindings: ki.KeyBindings("v"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.EnterVisualMode)},
		},
		{
			Bindings: ki.KeyBindings("u"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.Undo)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+r"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.Redo)},
		},
		{
			Bindings: ki.KeyBindings("w"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.WordRightStart)},
		},
		{
			Bindings: ki.KeyBindings("e"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.WordRightEnd)},
		},
		{
			Bindings: ki.KeyBindings("b"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.WordBack)},
		},
		{
			Bindings: ki.KeyBindings("^", "_"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.GoToInputStart)},
		},
		{
			Bindings: ki.KeyBindings("0"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.GoToLineStart)},
		},
		{
			Bindings: ki.KeyBindings("$"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.GoToLineEnd)},
		},
		{
			Bindings: ki.KeyBindings("o"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.InsertLineBelow)},
		},
		{
			Bindings: ki.KeyBindings("O"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.InsertLineAbove)},
		},
		{
			Bindings: ki.KeyBindings("dd"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.DeleteLine)},
		},
		{
			Bindings: ki.KeyBindings("dj"),
			Cond: []keyCond{{
				Mode:       n,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, false)
				},
			}},
		},
		{
			Bindings: ki.KeyBindings("dk"),
			Cond: []keyCond{{
				Mode:       n,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, true)
				},
			}},
		},
		{
			Bindings: ki.KeyBindings("dw"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.DeleteWordRight)},
		},
		{
			Bindings: ki.KeyBindings("D"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.DeleteAfterCursor)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+d"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.DownHalfPage)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+u"),
			Cond:     []keyCond{m.editorInputAction(n, m.editor.UpHalfPage)},
		},

		// ENTER CMD MODE
		{
			Bindings: ki.KeyBindings(":"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.enterCmdMode,
			}},
		},

		// CONFIRM ACTION
		{
			Bindings: ki.KeyBindings("enter"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.confirmAction,
			}, {
				Mode:       mode.Insert,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.confirmAction,
			}, {
				Mode:       mode.Command,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.confirmAction,
			}},
		},

		// CANCEL ACTION
		{
			Bindings: ki.KeyBindings("esc"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.cancelAction,
			}, {
				Mode:       mode.Insert,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.cancelAction,
			}, {
				Mode:       mode.Command,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.cancelAction,
			}, {
				Mode:       mode.Replace,
				Components: []c{m.editor},
				Action:     m.cancelAction,
			}},
		},
		{
			Bindings: ki.KeyBindings("ctrl+w l", "ctrl+w ctrl+l", "alt+e"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusNextColumn,
			}},
		},
		{
			Bindings: ki.KeyBindings("ctrl+w h", "ctrl+w ctrl+h", "alt+q"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusPrevColumn,
			}},
		},
		{
			Bindings: ki.KeyBindings("1"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusDirectoryTree,
			}},
		},
		{
			Bindings: ki.KeyBindings("2"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusNotesList,
			}},
		},
		{
			Bindings: ki.KeyBindings("3"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusEditor,
			}},
		},
	}
}

func (m *Model) editorInputAction(mode mode.Mode, fn func() message.StatusBarMsg) keyCond {
	return keyCond{
		Mode:       mode,
		Components: []c{m.editor},
		Action:     fn,
	}
}

///
/// Keyboard shortcut delegations
///

// focusColumn selects and higlights a column with index `index`
// (1=dirTree, 2=notesList, 3=editor)
func (m *Model) focusColumn(index int) message.StatusBarMsg {
	m.dirTree.SetFocus(index == 1)
	m.notesList.SetFocus(index == 2)
	m.editor.SetFocus(index == 3)
	m.currColFocus = index
	m.keyInput.FetchKeyMap(true)

	if index == 3 {
		relPath := utils.RelativePath(m.editor.CurrentBuffer.Path, true)
		icon := theme.Icon(theme.IconNote)
		return message.StatusBarMsg{
			Content: icon + " " + relPath,
			Column:  sbc.FileInfo,
		}
	}

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
	index := max(m.currColFocus-1, 1)
	return m.focusColumn(index)
}

// focusedComponent returns the component that is currently focused
func (m *Model) focusedComponent() Focusable {
	if m.dirTree.Focused() {
		return m.dirTree
	}
	if m.notesList.Focused() {
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
		if f == m.dirTree {
			statusMsg = m.dirTree.ContentInfo()
		}
	}

	return statusMsg
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (m *Model) lineDown() message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if f := m.focusedComponent(); f != nil {
		statusMsg = f.LineDown()

		if f == m.dirTree {
			statusMsg = m.dirTree.ContentInfo()
		}
	}

	return statusMsg
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
	if m.dirTree.Focused() || m.notesList.Focused() {
		m.mode.Current = mode.Insert
		m.statusBar.Focused = false
	}

	if m.dirTree.Focused() {
		return m.dirTree.Rename(
			m.dirTree.SelectedDir().Name(),
		)
	}

	if m.notesList.Focused() {
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
			m.editor,
		)
	}

	if m.mode.Current != mode.Normal &&
		!m.statusBar.Focused &&
		!m.editor.Focused() {
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
			if sel := m.notesList.SelectedItem(nil); sel != nil {
				statusMsg = m.editor.OpenBuffer(sel.Path())
			}
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
		m.enterNormalMode()
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

	m.keyInput.ResetKeysDown()

	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.General,
	}
}

func (m *Model) enterNormalMode() message.StatusBarMsg {
	m.editor.Vim.Mode.Current = mode.Normal
	m.mode.Current = mode.Normal
	m.statusBar.Focused = false

	return message.StatusBarMsg{
		Content: "",
		Type:    message.Prompt,
		Column:  sbc.General,
	}
}

func (m *Model) enterCmdMode() message.StatusBarMsg {
	if m.mode.Current != mode.Normal {
		return message.StatusBarMsg{}
	}

	m.editor.Vim.Mode.Current = mode.Command
	m.mode.Current = mode.Command
	m.statusBar.Focused = true

	return message.StatusBarMsg{
		Type:   message.Prompt,
		Column: sbc.General,
	}
}
