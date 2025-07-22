package tui

import (
	"strconv"

	"bellbird-notes/app/config"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/components/textarea"
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"
)

func (m *Model) FnRegistry() ki.FnRegistry {
	return ki.FnRegistry{
		// Movement/Navigation
		"LineDown":       m.lineDown,
		"LineUp":         m.lineUp,
		"DownHalfPage":   bind(m.editor.DownHalfPage),
		"UpHalfPage":     bind(m.editor.UpHalfPage),
		"CharacterLeft":  bind(m.editor.MoveCharacterLeft),
		"CharacterRight": bind(m.editor.MoveCharacterRight),
		"GoToTop":        m.goToTop,
		"GoToBottom":     m.goToBottom,

		"FocusNextColumn": m.focusNextColumn,
		"FocusPrevColumn": m.focusPrevColumn,
		"FocusFolders":    m.focusDirectoryTree,
		"FocusNotes":      m.focusNotesList,
		"FocusEditor":     m.focusEditor,

		// List specific actions
		"RenameListItem":    m.rename,
		"DeleteListItem":    m.delete,
		"YankListItem":      m.yankListItem,
		"CutListItem":       m.cutListItem,
		"PasteListItem":     m.pasteListItem,
		"TogglePinListItem": m.togglePin,

		// directory tree specific
		"TreeExpand":   bind(m.dirTree.Expand),
		"TreeCollapse": bind(m.dirTree.Collapse),
		"CreateFolder": m.createDir,

		// notes list specific
		"CreateNote": m.createNote,

		// General
		"EnterCommand":    m.enterCmdMode,
		"ShowBufferList":  m.showBufferList,
		"CloseBufferList": m.closeBufferList,
		"ConfirmAction":   m.confirmAction,
		"CancelAction":    m.cancelAction,
		"CloseNote":       bind(m.editor.DeleteCurrentBuffer),

		// Text editing
		"EnterNormalMode":   m.enterNormalMode,
		"Replace":           bind(m.editor.EnterReplaceMode),
		"ToggleVisual":      m.toggleVisual,
		"ToggleVisualLine":  m.toggleVisualLine,
		"ToggleVisualBlock": m.toggleVisualBlock,

		"Undo": bind(m.editor.Undo),
		"Redo": bind(m.editor.Redo),

		"InsertBefore":     m.enterInsertMode,
		"InsertAfter":      bind(m.editor.InsertAfter),
		"InsertBelow":      m.insertBelow,
		"InsertAbove":      m.insertAbove,
		"InsertBeforeLine": bind(m.editor.InsertLineStart),
		"InsertAfterLine":  bind(m.editor.InsertLineEnd),

		"SelectWord":             m.selectWord,
		"NextWord":               m.nextWord,
		"PrevWord":               m.prevWord,
		"GoToFirstNonWhiteSpace": bind(m.editor.GoToInputStart),
		"GoToLineStart":          bind(m.editor.GoToLineStart),
		"GoToLineEnd":            bind(m.editor.GoToLineEnd),
		"MergeLines":             bind(m.editor.MergeLineBelow),

		"DeleteLine":        bind(m.editor.DeleteLine),
		"DeleteWord":        m.deleteWord,
		"DeleteAfterCursor": m.deleteAfterCursor,
		"DeleteSelection":   m.deleteSelection,
		"DeleteCharacter":   m.deleteCharacter,

		"SubstituteText":    m.substituteText,
		"ChangeAfterCursor": m.changeAfterCursor,
		"ChangeLine":        m.changeLine,
		"ChangeWord":        m.changeWord,

		"YankSelection": m.yankSelection,
		"YankLine":      bind(m.editor.YankLine),
		"YankWord":      m.yankWord,
		"Paste":         m.paste,
	}
}

type StatusBarMsg = message.StatusBarMsg

func bind(fn func() StatusBarMsg) ki.CmdFn {
	return func(ki.Options) func() StatusBarMsg {
		return fn
	}
}

///
/// Keyboard shortcut delegations
///

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (m *Model) lineDown(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := m.focusedComponent(); f != nil {
			f.LineDown()

			if f == m.dirTree {
				msg = m.dirTree.ContentInfo()
			}
		}

		if m.editor.Focused() {
			multiline := opts.GetBool("multiline")
			msg = m.editor.LineDown(multiline)
		}

		return msg
	}
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (m *Model) lineUp(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := m.focusedComponent(); f != nil {
			f.LineUp()

			if f == m.dirTree {
				msg = m.dirTree.ContentInfo()
			}
		}

		if m.editor.Focused() {
			multiline := opts.GetBool("multiline")
			msg = m.editor.LineUp(multiline)
		}

		return msg
	}
}

// goToTop moves the current item of focused list to its first item
func (m *Model) goToTop(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			return f.GoToTop()
		}

		if m.editor.Focused() {
			return m.editor.GoToTop()
		}

		return StatusBarMsg{}
	}
}

// goToTop moves the current item of the focused list to its last item
func (m *Model) goToBottom(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			return f.GoToBottom()
		}

		if m.editor.Focused() {
			return m.editor.GoToBottom()
		}

		return StatusBarMsg{}
	}
}

// focusNextColumn selects and highlights the respectivley next of the
// currently selected column.
func (m *Model) focusNextColumn(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		nbrCols := 3
		index := min(m.currColFocus+1, nbrCols)
		return m.focusColumn(index)
	}
}

// focusPrevColumn selects and highlights the respectivley previous of the
// currently selected column.
func (m *Model) focusPrevColumn(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		firstCol := 1
		index := max(m.currColFocus-1, firstCol)
		return m.focusColumn(index)
	}
}

// focusDirectoryTree is a helper function
// for selecting the directory tree
func (m *Model) focusDirectoryTree(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.focusColumn(1)
	}
}

// focusNotesList() is a helper function
// for selecting the notes list
func (m *Model) focusNotesList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.focusColumn(2)
	}
}

// focusEditor is a helper function
// for selecting the editor
func (m *Model) focusEditor(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.focusColumn(3)
	}
}

// rename enters insert mode and renames the selected item
// in the directory or note list
func (m *Model) rename(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.dirTree.Focused() || m.notesList.Focused() {
			m.mode.Current = mode.Insert
			m.statusBar.Focused = false
		}

		if m.dirTree.Focused() {
			if m.dirTree.SelectedIndex() > 0 {
				return m.dirTree.Rename(
					m.dirTree.SelectedDir().Name(),
				)
			}
			m.mode.Current = mode.Normal
		}

		if m.notesList.Focused() {
			return m.notesList.Rename(
				m.notesList.SelectedItem(nil).Name(),
			)
		}
		return StatusBarMsg{}
	}
}

// delete enters insert mode and triggers a delete confirmation
// for the focused component
func (m *Model) delete(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// go into insert mode because we always ask for
		// confirmation before deleting anything
		m.mode.Current = mode.Insert

		statusMsg := StatusBarMsg{}

		if f := m.focusedComponent(); f != nil {
			m.statusBar.Focused = true
			statusMsg = f.ConfirmRemove()
		}

		if statusMsg.Type == message.None {
			m.mode.Current = mode.Normal
		}

		return statusMsg
	}
}

func (m *Model) yankListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			f.YankSelection(false)
		}

		return StatusBarMsg{}
	}
}

func (m *Model) cutListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			f.YankSelection(true)
		}
		return StatusBarMsg{}
	}
}

func (m *Model) pasteListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			return f.PasteSelection()
		}

		return StatusBarMsg{}
	}
}

// togglePin pins or unpins the selected item to the top of the list
func (m *Model) togglePin(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := m.focusedComponent(); f != nil {
			return f.TogglePinned()
		}
		return StatusBarMsg{}
	}
}

// createDir enters insert mode and triggers directory creation
func (m *Model) createDir(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.dirTree.Create(m.mode, m.statusBar)
	}
}

// createNote enters insert mode and triggers notes creation
func (m *Model) createNote(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.notesList.Create(m.mode, m.statusBar)
	}
}

func (m *Model) enterCmdMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.mode.Current != mode.Normal {
			return StatusBarMsg{}
		}

		m.editor.Vim.Mode.Current = mode.Command
		m.mode.Current = mode.Command
		m.statusBar.Focused = true

		return StatusBarMsg{
			Type:   message.Prompt,
			Column: sbc.General,
		}
	}
}

// showBufferList opens an overlay showing all open buffers
func (m *Model) showBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.OverlayOpenBuffers()
		return StatusBarMsg{}
	}
}

// closeBufferList opens an overlay showing all open buffers
func (m *Model) closeBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.bufferList.Focused() {
			m.editor.ListBuffers = false
			m.focusColumn(m.currColFocus)
		}
		return StatusBarMsg{}
	}
}

// confirmAction performs the primary action for the focused component,
// or loads note data into the editor if in normal mode.
func (m *Model) confirmAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := StatusBarMsg{}

		f := m.focusedComponent()

		if m.statusBar.Focused {
			statusMsg = m.statusBar.ConfirmAction(
				statusMsg.Sender,
				f,
				m.editor,
			)
			m.editor.Vim.Mode.Current = mode.Normal
		}

		if m.mode.Current != mode.Normal &&
			!m.statusBar.Focused &&
			!m.editor.Focused() {

			editState := m.notesList.EditState
			statusMsg = f.ConfirmAction()

			// Update the editor in case we're renaming the currently open buffer
			if editState == components.EditStates.Rename {
				m.editor.BuildHeader(m.editor.Size.Width, true)
				m.editor.UpdateMetaInfo()
			}
		} else {
			// only open stuff if we're in normal mode
			if m.mode.Current != mode.Normal {
				m.mode.Current = mode.Normal
				return statusMsg
			}

			switch f {
			case m.dirTree:
				path := m.dirTree.SelectedDir().Path()
				m.notesList.CurrentPath = path
				m.conf.SetMetaValue("", config.CurrentDirectory, path)
				statusMsg = m.notesList.Refresh(true, true)

			case m.notesList:
				if sel := m.notesList.SelectedItem(nil); sel != nil {
					statusMsg = m.editor.OpenBuffer(sel.Path())
				}

			case m.bufferList:
				items := m.bufferList.Items()
				sel := items[m.bufferList.SelectedIndex()]

				if buf, ok, _ := m.Buffers.Contains(sel.Path()); ok {
					// close buffer list overlay
					m.editor.ListBuffers = false
					m.editor.SwitchBuffer(buf)
					m.bufferList.SetSelectedIndex(0)
					m.focusEditor(opts)()
				}

				//default:
				//	f.ConfirmAction()
			}
		}

		m.mode.Current = mode.Normal

		return statusMsg
	}
}

// cancelAction resets mode to normal
// and cancels pending actions in the focused component.
func (m *Model) cancelAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.mode.Current = mode.Normal
		m.statusBar.Focused = false

		if m.statusBar.Prompt.Focused() {
			m.statusBar.CancelAction(func() {})
			m.enterNormalMode(opts)
		} else {
			if f := m.focusedComponent(); f != nil {
				resetIndex := false
				stateCreate := components.EditStates.Create

				if m.dirTree.EditState == stateCreate ||
					m.notesList.EditState == stateCreate {
					resetIndex = true
				}

				return f.CancelAction(func() {
					f.Refresh(resetIndex, false)
				})
			}
		}

		m.keyInput.ResetKeysDown()

		return StatusBarMsg{
			Content: "",
			Column:  sbc.General,
		}
	}
}

func (m *Model) enterNormalMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.Vim.Mode.Current = mode.Normal
		m.mode.Current = mode.Normal
		m.statusBar.Focused = false

		return StatusBarMsg{
			Content: "",
			Type:    message.Prompt,
			Column:  sbc.General,
		}
	}
}

func (m *Model) toggleVisual(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.mode.Current == mode.Visual {
			return m.editor.EnterNormalMode(true)
		}

		return m.editor.EnterVisualMode(textarea.SelectVisual)
	}
}

func (m *Model) toggleVisualLine(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.mode.Current == mode.VisualLine {
			return m.editor.EnterNormalMode(true)
		}

		return m.editor.EnterVisualMode(textarea.SelectVisualLine)
	}
}

func (m *Model) toggleVisualBlock(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.mode.Current == mode.VisualBlock {
			return m.editor.EnterNormalMode(true)
		}

		return m.editor.EnterVisualMode(textarea.SelectVisualBlock)
	}
}

func (m *Model) enterInsertMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.EnterInsertMode(true)
	}
}

func (m *Model) insertBelow(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.InsertLine(false)
	}
}

func (m *Model) insertAbove(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.InsertLine(true)
	}
}

func (m *Model) selectWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")
		return m.editor.SelectWord(outer)
	}
}

func (m *Model) nextWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool("end")
		return m.editor.WordForward(end)
	}
}

func (m *Model) prevWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool("end")
		return m.editor.WordBack(end)
	}
}

func (m *Model) deleteWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")

		if opts.GetBool("remaining") {
			return m.editor.DeleteWordRight()
		}

		return m.editor.DeleteWord(outer, false)
	}
}

func (m *Model) deleteAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.DeleteAfterCursor(false)
	}
}

func (m *Model) deleteSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if m.mode.IsAnyVisual() {
			return m.editor.DeleteRune(false, true, false)
		}
		return StatusBarMsg{}
	}
}

func (m *Model) deleteCharacter(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.DeleteRune(false, true, false)
	}
}

func (m *Model) substituteText(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := m.editor.DeleteRune(false, true, false)

		if opts.GetBool("new_line") {
			m.editor.Textarea.EmptyLineAbove()
		}

		m.editor.EnterInsertMode(false)

		return msg
	}
}

func (m *Model) changeAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.DeleteAfterCursor(true)
		m.editor.MoveCharacterRight()
		m.editor.EnterInsertMode(false)

		return StatusBarMsg{}
	}
}

func (m *Model) changeLine(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.GoToLineStart()
		m.editor.DeleteAfterCursor(false)
		m.editor.EnterInsertMode(false)

		return m.editor.ResetSelectedRowsCount()
	}
}

func (m *Model) changeWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.DeleteWord(opts.GetBool("outer"), true)
	}
}

func (m *Model) yankSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.editor.YankSelection(false)
	}
}

func (m *Model) yankWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")
		return m.editor.YankWord(outer)
	}
}

func (m *Model) paste(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// If we're in any visual mode delete the selection without
		// history and yanking right away
		//
		// @todo when undoing this step the cursor is one character left of
		// the word we replace - fix this
		if m.mode.IsAnyVisual() {
			m.editor.DeleteRune(true, false, true)
			m.editor.EnterInsertMode(false)
			m.editor.MoveCharacterLeft()
		}

		msg := m.editor.Paste()

		m.selectWord(opts)
		return msg
	}
}

func (m *Model) OverlayOpenBuffers() (string, int, int) {
	m.editor.ListBuffers = true
	m.bufferList.SetFocus(true)

	x, y := m.overlayPosition(m.bufferList.Width)
	overlay := m.bufferList.View()

	m.updateComponents(false)
	return overlay, x, y
}

// focusColumn selects and higlights a column with index `index`
// (1=dirTree, 2=notesList, 3=editor)
func (m *Model) focusColumn(index int) StatusBarMsg {
	m.conf.SetMetaValue("", config.CurrentComponent, strconv.Itoa(index))

	m.dirTree.SetFocus(index == 1)
	m.dirTree.BuildHeader(m.dirTree.Size.Width, true)

	m.notesList.SetFocus(index == 2)
	m.notesList.BuildHeader(m.notesList.Size.Width, true)

	m.editor.SetFocus(index == 3)
	m.editor.BuildHeader(m.notesList.Size.Width, true)

	m.currColFocus = index
	m.keyInput.FetchKeyMap(true)

	if index == 3 {
		relPath := utils.RelativePath(m.editor.CurrentBuffer.Path(false), true)
		icon := theme.Icon(theme.IconNote, m.conf.NerdFonts())
		return StatusBarMsg{
			Content: icon + " " + relPath,
			Column:  sbc.FileInfo,
		}
	}

	return StatusBarMsg{}
}

// focusedComponent returns the component that is currently focused
func (m *Model) focusedComponent() Focusable {
	if m.dirTree.Focused() {
		return m.dirTree
	}

	if m.notesList.Focused() {
		return m.notesList
	}

	if m.bufferList.Focused() {
		return m.bufferList
	}

	return nil
}

func (m *Model) unfocusAllColumns() StatusBarMsg {
	m.dirTree.SetFocus(false)
	m.dirTree.BuildHeader(m.dirTree.Size.Width, true)

	m.notesList.SetFocus(false)
	m.notesList.BuildHeader(m.notesList.Size.Width, true)

	m.editor.SetFocus(false)
	m.editor.BuildHeader(m.editor.Size.Width, true)

	m.keyInput.FetchKeyMap(true)

	m.statusBar.Focused = false

	return StatusBarMsg{}
}
