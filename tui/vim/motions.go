package vim

import (
	"strconv"

	"bellbird-notes/app/config"
	"bellbird-notes/app/utils"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/components/textarea"
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"
)

func (v *Vim) FnRegistry() ki.MotionRegistry {
	return ki.MotionRegistry{
		// Movement/Navigation
		"LineDown":       v.lineDown,
		"LineUp":         v.lineUp,
		"DownHalfPage":   bind(v.app.Editor.DownHalfPage),
		"UpHalfPage":     bind(v.app.Editor.UpHalfPage),
		"CharacterLeft":  bind(v.app.Editor.MoveCharacterLeft),
		"CharacterRight": bind(v.app.Editor.MoveCharacterRight),
		"GoToTop":        v.goToTop,
		"GoToBottom":     v.goToBottom,

		"FocusNextColumn": v.focusNextColumn,
		"FocusPrevColumn": v.focusPrevColumn,
		"FocusFolders":    v.focusDirectoryTree,
		"FocusNotes":      v.focusNotesList,
		"FocusEditor":     v.focusEditor,

		// List specific actions
		"RenameListItem":    v.rename,
		"DeleteListItem":    v.delete,
		"YankListItem":      v.yankListItem,
		"CutListItem":       v.cutListItem,
		"PasteListItem":     v.pasteListItem,
		"TogglePinListItem": v.togglePin,

		// directory tree specific
		"TreeExpand":   bind(v.app.DirTree.Expand),
		"TreeCollapse": bind(v.app.DirTree.Collapse),
		"CreateFolder": v.createDir,

		// notes list specific
		"CreateNote": v.createNote,

		// General
		"EnterCommand":    v.enterCmdMode,
		"ShowBufferList":  v.showBufferList,
		"CloseBufferList": v.closeBufferList,
		"ConfirmAction":   v.confirmAction,
		"CancelAction":    v.cancelAction,
		"CloseNote":       bind(v.app.Editor.DeleteCurrentBuffer),

		// Text editing
		"EnterNormalMode":   v.enterNormalMode,
		"Replace":           bind(v.app.Editor.EnterReplaceMode),
		"ToggleVisual":      v.toggleVisual,
		"ToggleVisualLine":  v.toggleVisualLine,
		"ToggleVisualBlock": v.toggleVisualBlock,

		"Undo": bind(v.app.Editor.Undo),
		"Redo": bind(v.app.Editor.Redo),

		"InsertBefore":     v.enterInsertMode,
		"InsertAfter":      bind(v.app.Editor.InsertAfter),
		"InsertBelow":      v.insertBelow,
		"InsertAbove":      v.insertAbove,
		"InsertBeforeLine": bind(v.app.Editor.InsertLineStart),
		"InsertAfterLine":  bind(v.app.Editor.InsertLineEnd),

		"SelectWord":             v.selectWord,
		"NextWord":               v.nextWord,
		"PrevWord":               v.prevWord,
		"FindCharacter":          v.findCharacter,
		"GoToFirstNonWhiteSpace": bind(v.app.Editor.GoToInputStart),
		"GoToLineStart":          bind(v.app.Editor.GoToLineStart),
		"GoToLineEnd":            bind(v.app.Editor.GoToLineEnd),
		"MergeLines":             bind(v.app.Editor.MergeLineBelow),

		"DeleteLine":        bind(v.app.Editor.DeleteLine),
		"DeleteWord":        v.deleteWord,
		"DeleteAfterCursor": v.deleteAfterCursor,
		"DeleteSelection":   v.deleteSelection,
		"DeleteCharacter":   v.deleteCharacter,

		"SubstituteText":    v.substituteText,
		"ChangeAfterCursor": v.changeAfterCursor,
		"ChangeLine":        v.changeLine,
		"ChangeWord":        v.changeWord,

		"YankSelection": v.yankSelection,
		"YankLine":      bind(v.app.Editor.YankLine),
		"YankWord":      v.yankWord,
		"Paste":         v.paste,
	}
}

type StatusBarMsg = message.StatusBarMsg

func bind(fn func() StatusBarMsg) ki.Motion {
	return func(ki.Options) func() StatusBarMsg {
		return fn
	}
}

///
/// Keyboard shortcut delegations
///

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (v *Vim) lineDown(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := v.focusedComponent(); f != nil {
			f.LineDown()

			if f == v.app.DirTree {
				msg = v.app.DirTree.ContentInfo()
			}
		}

		if v.app.Editor.Focused() {
			multiline := opts.GetBool("multiline")
			msg = v.app.Editor.LineDown(multiline)
		}

		return msg
	}
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (v *Vim) lineUp(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := v.focusedComponent(); f != nil {
			f.LineUp()

			if f == v.app.DirTree {
				msg = v.app.DirTree.ContentInfo()
			}
		}

		if v.app.Editor.Focused() {
			multiline := opts.GetBool("multiline")
			msg = v.app.Editor.LineUp(multiline)
		}

		return msg
	}
}

// goToTop moves the current item of focused list to its first item
func (v *Vim) goToTop(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			return f.GoToTop()
		}

		if v.app.Editor.Focused() {
			return v.app.Editor.GoToTop()
		}

		return StatusBarMsg{}
	}
}

// goToTop moves the current item of the focused list to its last item
func (v *Vim) goToBottom(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			return f.GoToBottom()
		}

		if v.app.Editor.Focused() {
			return v.app.Editor.GoToBottom()
		}

		return StatusBarMsg{}
	}
}

// focusNextColumn selects and highlights the respectivley next of the
// currently selected column.
func (v *Vim) focusNextColumn(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		nbrCols := 3
		index := min(v.app.CurrColFocus+1, nbrCols)
		return v.FocusColumn(index)
	}
}

// focusPrevColumn selects and highlights the respectivley previous of the
// currently selected column.
func (v *Vim) focusPrevColumn(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		firstCol := 1
		index := max(v.app.CurrColFocus-1, firstCol)
		return v.FocusColumn(index)
	}
}

// focusDirectoryTree is a helper function
// for selecting the directory tree
func (v *Vim) focusDirectoryTree(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.FocusColumn(1)
	}
}

// focusNotesList() is a helper function
// for selecting the notes list
func (v *Vim) focusNotesList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.FocusColumn(2)
	}
}

// focusEditor is a helper function
// for selecting the editor
func (v *Vim) focusEditor(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.FocusColumn(3)
	}
}

// rename enters insert mode and renames the selected item
// in the directory or note list
func (v *Vim) rename(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.DirTree.Focused() || v.app.NotesList.Focused() {
			v.app.Mode.Current = mode.Insert
			v.app.StatusBar.Focused = false
		}

		if v.app.DirTree.Focused() {
			if v.app.DirTree.SelectedIndex() > 0 {
				return v.app.DirTree.Rename(
					v.app.DirTree.SelectedDir().Name(),
				)
			}
			v.app.Mode.Current = mode.Normal
		}

		if v.app.NotesList.Focused() {
			return v.app.NotesList.Rename(
				v.app.NotesList.SelectedItem(nil).Name(),
			)
		}
		return StatusBarMsg{}
	}
}

// delete enters insert mode and triggers a delete confirmation
// for the focused component
func (v *Vim) delete(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// go into insert mode because we always ask for
		// confirmation before deleting anything
		v.app.Mode.Current = mode.Insert

		statusMsg := StatusBarMsg{}

		if f := v.focusedComponent(); f != nil {
			v.app.StatusBar.Focused = true
			statusMsg = f.ConfirmRemove()
		}

		if statusMsg.Type == message.None {
			v.app.Mode.Current = mode.Normal
		}

		return statusMsg
	}
}

func (v *Vim) yankListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			f.YankSelection(false)
		}

		return StatusBarMsg{}
	}
}

func (v *Vim) cutListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			f.YankSelection(true)
		}
		return StatusBarMsg{}
	}
}

func (v *Vim) pasteListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			return f.PasteSelection()
		}

		return StatusBarMsg{}
	}
}

// togglePin pins or unpins the selected item to the top of the list
func (v *Vim) togglePin(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := v.focusedComponent(); f != nil {
			return f.TogglePinned()
		}
		return StatusBarMsg{}
	}
}

// createDir enters insert mode and triggers directory creation
func (v *Vim) createDir(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.DirTree.Create(v.app.Mode, v.app.StatusBar)
	}
}

// createNote enters insert mode and triggers notes creation
func (v *Vim) createNote(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.NotesList.Create(v.app.Mode, v.app.StatusBar)
	}
}

func (v *Vim) enterCmdMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.Mode.Current != mode.Normal {
			return StatusBarMsg{}
		}

		v.app.Editor.Mode.Current = mode.Command
		v.app.Mode.Current = mode.Command
		v.app.StatusBar.Focused = true

		return StatusBarMsg{
			Type:   message.Prompt,
			Column: sbc.General,
		}
	}
}

// showBufferList opens an overlay showing all open buffers
func (v *Vim) showBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		v.OverlayOpenBuffers()
		return StatusBarMsg{}
	}
}

// closeBufferList opens an overlay showing all open buffers
func (v *Vim) closeBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.BufferList.Focused() {
			v.app.Editor.ListBuffers = false
			v.FocusColumn(v.app.CurrColFocus)
		}
		return StatusBarMsg{}
	}
}

// confirmAction performs the primary action for the focused component,
// or loads note data into the editor if in normal mode.
func (v *Vim) confirmAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := StatusBarMsg{}

		f := v.focusedComponent()

		if v.app.StatusBar.Focused {
			statusMsg = v.app.StatusBar.ConfirmAction(
				statusMsg.Sender,
				f,
				v.app.Editor,
			)
			v.app.Editor.Mode.Current = mode.Normal
		}

		if v.app.Mode.Current != mode.Normal &&
			!v.app.StatusBar.Focused &&
			!v.app.Editor.Focused() {

			editState := v.app.NotesList.EditState
			statusMsg = f.ConfirmAction()

			// Update the editor in case we're renaming the currently open buffer
			if editState == components.EditStates.Rename {
				v.app.Editor.BuildHeader(v.app.Editor.Size.Width, true)
				v.app.Editor.UpdateMetaInfo()
			}
		} else {
			// only open stuff if we're in normal mode
			if v.app.Mode.Current != mode.Normal {
				v.app.Mode.Current = mode.Normal
				return statusMsg
			}

			switch f {
			case v.app.DirTree:
				path := v.app.DirTree.SelectedDir().Path()
				v.app.NotesList.CurrentPath = path
				//m.conf.SetMetaValue("", config.CurrentDirectory, path)
				statusMsg = v.app.NotesList.Refresh(true, true)

			case v.app.NotesList:
				if sel := v.app.NotesList.SelectedItem(nil); sel != nil {
					statusMsg = v.app.Editor.OpenBuffer(sel.Path())
				}

			case v.app.BufferList:
				items := v.app.BufferList.Items()
				sel := items[v.app.BufferList.SelectedIndex()]

				if buf, ok, _ := v.app.Editor.Buffers.Contain(sel.Path()); ok {
					// close buffer list overlay
					v.app.Editor.ListBuffers = false
					v.app.Editor.SwitchBuffer(buf)
					v.app.BufferList.SetSelectedIndex(0)
					v.focusEditor(opts)()
				}

				//default:
				//	f.ConfirmAction()
			}
		}

		v.app.Mode.Current = mode.Normal

		return statusMsg
	}
}

// cancelAction resets mode to normal
// and cancels pending actions in the focused component.
func (v *Vim) cancelAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		v.app.Mode.Current = mode.Normal
		v.app.StatusBar.Focused = false

		if v.app.StatusBar.Prompt.Focused() {
			v.app.StatusBar.CancelAction(func() {})
			v.enterNormalMode(opts)
		} else {
			if f := v.focusedComponent(); f != nil {
				resetIndex := false
				stateCreate := components.EditStates.Create

				if v.app.DirTree.EditState == stateCreate ||
					v.app.NotesList.EditState == stateCreate {
					resetIndex = true
				}

				return f.CancelAction(func() {
					f.Refresh(resetIndex, false)
				})
			}
		}

		//m.keyInput.ResetKeysDown()

		return StatusBarMsg{
			Content: "",
			Column:  sbc.General,
		}
	}
}

func (v *Vim) enterNormalMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		v.app.Editor.Mode.Current = mode.Normal
		v.app.Mode.Current = mode.Normal
		v.app.StatusBar.Focused = false

		return StatusBarMsg{
			Content: "",
			Type:    message.Prompt,
			Column:  sbc.General,
		}
	}
}

func (v *Vim) toggleVisual(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.Mode.Current == mode.Visual {
			return v.app.Editor.EnterNormalMode(true)
		}

		return v.app.Editor.EnterVisualMode(textarea.SelectVisual)
	}
}

func (v *Vim) toggleVisualLine(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.Mode.Current == mode.VisualLine {
			return v.app.Editor.EnterNormalMode(true)
		}

		return v.app.Editor.EnterVisualMode(textarea.SelectVisualLine)
	}
}

func (v *Vim) toggleVisualBlock(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.Mode.Current == mode.VisualBlock {
			return v.app.Editor.EnterNormalMode(true)
		}

		return v.app.Editor.EnterVisualMode(textarea.SelectVisualBlock)
	}
}

func (v *Vim) enterInsertMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.Editor.EnterInsertMode(true)
	}
}

func (v *Vim) insertBelow(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.Editor.InsertLine(false)
	}
}

func (m *Vim) insertAbove(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return m.app.Editor.InsertLine(true)
	}
}

func (v *Vim) selectWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")
		return v.app.Editor.SelectWord(outer)
	}
}

func (v *Vim) nextWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool("end")
		return v.app.Editor.WordForward(end)
	}
}

func (v *Vim) prevWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool("end")
		return v.app.Editor.WordBack(end)
	}
}

func (v *Vim) findCharacter(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		binding := []rune(opts.GetString("binding"))
		back := opts.GetBool("back")
		char := v.KeyMap.KeySequence[len(binding):]
		return v.app.Editor.FindCharacter(char, back)
	}
}

func (v *Vim) deleteWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")

		if opts.GetBool("remaining") {
			return v.app.Editor.DeleteWordRight()
		}

		return v.app.Editor.DeleteWord(outer, false)
	}
}

func (v *Vim) deleteAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.Editor.DeleteAfterCursor(false)
	}
}

func (v *Vim) deleteSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if v.app.Mode.IsAnyVisual() {
			return v.app.Editor.DeleteRune(false, true, false)
		}
		return StatusBarMsg{}
	}
}

func (v *Vim) deleteCharacter(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.Editor.DeleteRune(false, true, false)
	}
}

func (v *Vim) substituteText(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := v.app.Editor.DeleteRune(false, true, false)

		if opts.GetBool("new_line") {
			v.app.Editor.Textarea.EmptyLineAbove()
		}

		v.app.Editor.EnterInsertMode(false)

		return msg
	}
}

func (v *Vim) changeAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		v.app.Editor.DeleteAfterCursor(true)
		v.app.Editor.MoveCharacterRight()
		v.app.Editor.EnterInsertMode(false)

		return StatusBarMsg{}
	}
}

func (v *Vim) changeLine(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		v.app.Editor.GoToLineStart()
		v.app.Editor.DeleteAfterCursor(false)
		v.app.Editor.EnterInsertMode(false)

		return v.app.Editor.ResetSelectedRowsCount()
	}
}

func (v *Vim) changeWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")

		if opts.GetBool("remaining") {
			v.app.Editor.DeleteWordRight()
			return v.app.Editor.EnterInsertMode(false)
		}

		return v.app.Editor.DeleteWord(outer, true)
	}
}

func (v *Vim) yankSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return v.app.Editor.YankSelection(false)
	}
}

func (v *Vim) yankWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")
		return v.app.Editor.YankWord(outer)
	}
}

func (v *Vim) paste(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// If we're in any visual mode delete the selection without
		// history and yanking right away
		//
		// @todo when undoing this step the cursor is one character left of
		// the word we replace - fix this
		if v.app.Mode.IsAnyVisual() {
			v.app.Editor.DeleteRune(true, false, true)
			v.app.Editor.EnterInsertMode(false)
			v.app.Editor.MoveCharacterLeft()
		}

		msg := v.app.Editor.Paste()

		v.selectWord(opts)
		return msg
	}
}

func (v *Vim) OverlayOpenBuffers() (string, int, int) {
	v.app.Editor.ListBuffers = true
	v.app.BufferList.SetFocus(true)

	x, y := v.app.OverlayPosition(v.app.BufferList.Width())
	overlay := v.app.BufferList.View()

	v.app.UpdateComponents(false)
	return overlay, x, y
}

// FocusColumn selects and higlights a column with index `index`
// (1=dirTree, 2=notesList, 3=editor)
func (v *Vim) FocusColumn(index int) StatusBarMsg {
	v.app.Conf.SetMetaValue("", config.CurrentComponent, strconv.Itoa(index))

	v.app.DirTree.SetFocus(index == 1)
	v.app.DirTree.BuildHeader(v.app.DirTree.Size.Width, true)

	v.app.NotesList.SetFocus(index == 2)
	v.app.NotesList.BuildHeader(v.app.NotesList.Size.Width, true)

	v.app.Editor.SetFocus(index == 3)
	v.app.Editor.BuildHeader(v.app.NotesList.Size.Width, true)

	v.app.CurrColFocus = index
	v.KeyMap.FetchKeyMap(true)

	if index == 3 {
		relPath := utils.RelativePath(v.app.Editor.CurrentBuffer.Path(false), true)
		icon := theme.Icon(theme.IconNote, v.app.Conf.NerdFonts())
		return StatusBarMsg{
			Content: icon + " " + relPath,
			Column:  sbc.FileInfo,
		}
	}

	return StatusBarMsg{}
}

// focusedComponent returns the component that is currently focused
func (v *Vim) focusedComponent() interfaces.Focusable {
	if v.app.DirTree.Focused() {
		return v.app.DirTree
	}

	if v.app.NotesList.Focused() {
		return v.app.NotesList
	}

	if v.app.BufferList.Focused() {
		return v.app.BufferList
	}

	return nil
}

func (v *Vim) UnfocusAllColumns() StatusBarMsg {
	v.app.DirTree.SetFocus(false)
	v.app.DirTree.BuildHeader(v.app.DirTree.Size.Width, true)

	v.app.NotesList.SetFocus(false)
	v.app.NotesList.BuildHeader(v.app.NotesList.Size.Width, true)

	v.app.Editor.SetFocus(false)
	v.app.Editor.BuildHeader(v.app.Editor.Size.Width, true)

	v.KeyMap.FetchKeyMap(true)

	v.app.StatusBar.Focused = false

	return StatusBarMsg{}
}
