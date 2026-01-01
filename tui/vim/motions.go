package vim

import (
	"strconv"
	"unicode"

	"bellbird-notes/app/config"
	"bellbird-notes/app/utils"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components/editor"
	"bellbird-notes/tui/components/textarea"
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"
)

func (vim *Vim) FnRegistry() ki.MotionRegistry {
	return ki.MotionRegistry{
		// Movement/Navigation
		"LineDown":       vim.lineDown,
		"LineUp":         vim.lineUp,
		"DownHalfPage":   bind(vim.app.Editor.DownHalfPage),
		"UpHalfPage":     bind(vim.app.Editor.UpHalfPage),
		"CharacterLeft":  bind(vim.app.Editor.MoveCharacterLeft),
		"CharacterRight": bind(vim.app.Editor.MoveCharacterRight),
		"GoToTop":        vim.goToTop,
		"GoToBottom":     vim.goToBottom,

		"FocusNextColumn": vim.focusNextColumn,
		"FocusPrevColumn": vim.focusPrevColumn,
		"FocusFolders":    vim.focusDirectoryTree,
		"FocusNotes":      vim.focusNotesList,
		"FocusEditor":     vim.focusEditor,

		// List specific actions
		"RenameListItem":    vim.rename,
		"DeleteListItem":    vim.delete,
		"YankListItem":      vim.yankListItem,
		"CutListItem":       vim.cutListItem,
		"PasteListItem":     vim.pasteListItem,
		"TogglePinListItem": vim.togglePin,
		"RefreshList":       vim.refreshList,

		// directory tree specific
		"TreeExpand":   bind(vim.app.DirTree.Expand),
		"TreeCollapse": bind(vim.app.DirTree.Collapse),
		"CreateFolder": vim.createDir,

		// notes list specific
		"CreateNote": vim.createNote,

		// General
		"EnterCommand":         vim.enterCmdMode,
		"ShowBufferList":       vim.showBufferList,
		"CloseBufferList":      vim.closeBufferList,
		"ConfirmAction":        vim.confirmAction,
		"CancelAction":         vim.cancelAction,
		"CloseNote":            bind(vim.app.Editor.CloseCurrentBuffer),
		"ReopenLastClosedNote": bind(vim.app.Editor.ReopenLastClosedBuffer),
		"CloseSelectedNote":    vim.deleteSelectedBuffer,
		"NewScratch":           vim.newScratchBuffer,
		"ToggleFolders":        bind(vim.app.DirTree.Toggle),
		"ToggleNotes":          bind(vim.app.NotesList.Toggle),

		// Text editing
		"SaveCurrentBuffer": bind(vim.app.Editor.SaveBuffer),
		"EnterNormalMode":   vim.enterNormalMode,
		"Replace":           bind(vim.app.Editor.EnterReplaceMode),
		"ToggleVisual":      vim.toggleVisual,
		"ToggleVisualLine":  vim.toggleVisualLine,
		"ToggleVisualBlock": vim.toggleVisualBlock,

		"Undo": bind(vim.app.Editor.Undo),
		"Redo": bind(vim.app.Editor.Redo),

		"InsertBefore":     vim.enterInsertMode,
		"InsertAfter":      bind(vim.app.Editor.InsertAfter),
		"InsertBelow":      vim.insertBelow,
		"InsertAbove":      vim.insertAbove,
		"InsertBeforeLine": bind(vim.app.Editor.InsertLineStart),
		"InsertAfterLine":  bind(vim.app.Editor.InsertLineEnd),

		"SelectWord":             vim.selectWord,
		"NextWord":               vim.nextWord,
		"PrevWord":               vim.prevWord,
		"FindCharacter":          vim.findCharacter,
		"FindWordUnderCursor":    vim.findWordUnderCursor,
		"Find":                   vim.find,
		"MoveToMatch":            vim.moveToMatch,
		"GoToFirstNonWhiteSpace": bind(vim.app.Editor.GoToInputStart),
		"GoToLineStart":          bind(vim.app.Editor.GoToLineStart),
		"GoToLineEnd":            bind(vim.app.Editor.GoToLineEnd),
		"MergeLines":             bind(vim.app.Editor.MergeLineBelow),

		"DeleteLine":             bind(vim.app.Editor.DeleteLine),
		"DeleteWord":             vim.deleteWord,
		"DeleteAfterCursor":      vim.deleteAfterCursor,
		"DeleteSelection":        vim.deleteSelection,
		"DeleteCharacter":        vim.deleteCharacter,
		"DeleteFromCursorToChar": vim.deleteFromCursorToChar,

		"SubstituteText":    vim.substituteText,
		"ChangeAfterCursor": vim.changeAfterCursor,
		"ChangeLine":        vim.changeLine,
		"ChangeWord":        vim.changeWord,

		"YankAfterCursor":   bind(vim.app.Editor.YankAfterCursor),
		"YankSelection":     vim.yankSelection,
		"YankLine":          bind(vim.app.Editor.YankLine),
		"YankWord":          vim.yankWord,
		"Paste":             vim.paste,
		"ChangeToLowerCase": vim.changeToLowerCase,
		"ChangeToUpperCase": vim.changeToUpperCase,

		// Command
		"CmdHistoryBack":    bind(vim.app.StatusBar.PromptHistoryBack),
		"CmdHistoryForward": bind(vim.app.StatusBar.PromptHistoryForward),

		// Search
		"CmdSearchHistoryBack":    bind(vim.app.StatusBar.SearchHistoryBack),
		"CmdSearchHistoryForward": bind(vim.app.StatusBar.SearchHistoryForward),
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
func (vim *Vim) lineDown(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := vim.focusedComponent(); f != nil {
			f.LineDown()

			if f == vim.app.DirTree {
				msg = vim.app.DirTree.ContentInfo()
			}
		}

		if vim.app.Editor.Focused() {
			multiline := opts.GetBool(ki.Args.MultiLine)
			msg = vim.app.Editor.LineDown(multiline)
		}

		return msg
	}
}

// lineUp moves the cursor one line up in the currently focused column.
// Ignores editor since it is handled differently
func (vim *Vim) lineUp(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}

		if f := vim.focusedComponent(); f != nil {
			f.LineUp()

			if f == vim.app.DirTree {
				msg = vim.app.DirTree.ContentInfo()
			}
		}

		if vim.app.Editor.Focused() {
			multiline := opts.GetBool(ki.Args.MultiLine)
			msg = vim.app.Editor.LineUp(multiline)
		}

		return msg
	}
}

// goToTop moves the current item of focused list to its first item
func (vim *Vim) goToTop(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			return f.GoToTop()
		}

		if vim.app.Editor.Focused() {
			return vim.app.Editor.GoToTop()
		}

		return StatusBarMsg{}
	}
}

// goToBottom moves the current item of the focused list to its last item
func (vim *Vim) goToBottom(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			return f.GoToBottom()
		}

		if vim.app.Editor.Focused() {
			return vim.app.Editor.GoToBottom()
		}

		return StatusBarMsg{}
	}
}

// focusNextColumn selects and highlights the respectivley next of the
// currently selected column.
func (vim *Vim) focusNextColumn(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		nbrCols := 3
		currenColumn := vim.app.CurrColFocus

		cycle := opts.GetBool(ki.Args.Cycle)

		if cycle && currenColumn == nbrCols {
			currenColumn = 0
		}

		index := min(currenColumn+1, nbrCols)
		return vim.FocusColumn(index)
	}
}

// focusPrevColumn selects and highlights the respectivley previous of the
// currently selected column.
func (vim *Vim) focusPrevColumn(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		firstCol := 1
		currenColumn := vim.app.CurrColFocus

		cycle := opts.GetBool(ki.Args.Cycle)

		if cycle && currenColumn == 1 {
			currenColumn = 4
		}

		column := max(currenColumn-1, firstCol)
		return vim.FocusColumn(column)
	}
}

// focusDirectoryTree is a helper function
// for selecting the directory tree
func (vim *Vim) focusDirectoryTree(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.FocusColumn(1)
	}
}

// focusNotesList() is a helper function
// for selecting the notes list
func (vim *Vim) focusNotesList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.FocusColumn(2)
	}
}

// focusEditor is a helper function
// for selecting the editor
func (vim *Vim) focusEditor(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.FocusColumn(3)
	}
}

// rename enters insert mode and renames the selected item
// in the directory or note list
func (vim *Vim) rename(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.DirTree.Focused() || vim.app.NotesList.Focused() {
			vim.app.Mode.Current = mode.Insert
			vim.app.StatusBar.Focused = false
		}

		if vim.app.DirTree.Focused() {
			if vim.app.DirTree.SelectedIndex > 0 {
				return vim.app.DirTree.Rename(
					vim.app.DirTree.SelectedDir().Name(),
				)
			}
			vim.app.Mode.Current = mode.Normal
		}

		if vim.app.NotesList.Focused() {
			return vim.app.NotesList.Rename(
				vim.app.NotesList.SelectedItem(nil).Name(),
			)
		}
		return StatusBarMsg{}
	}
}

// delete enters insert mode and triggers a delete confirmation
// for the focused component
func (vim *Vim) delete(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// go into insert mode because we always ask for
		// confirmation before deleting anything
		vim.app.Mode.Current = mode.Insert

		statusMsg := StatusBarMsg{}

		if f := vim.focusedComponent(); f != nil {
			vim.app.StatusBar.Focused = true
			statusMsg = f.ConfirmRemove()
		}

		if statusMsg.Type == message.None {
			vim.app.Mode.Current = mode.Normal
		}

		return statusMsg
	}
}

func (vim *Vim) yankListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			f.YankSelection(false)
		}

		return StatusBarMsg{}
	}
}

func (vim *Vim) cutListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			f.YankSelection(true)
		}
		return StatusBarMsg{}
	}
}

func (vim *Vim) pasteListItem(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			return f.PasteSelectedItems()
		}

		return StatusBarMsg{}
	}
}

// togglePin pins or unpins the selected item to the top of the list
func (vim *Vim) togglePin(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if f := vim.focusedComponent(); f != nil {
			return f.TogglePinnedItems()
		}
		return StatusBarMsg{}
	}
}

func (vim *Vim) refreshList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.DirTree.RefreshBranch(0, vim.app.DirTree.SelectedIndex)
		vim.app.DirTree.Refresh(false, false)
		vim.app.DirTree.Refresh(false, false)
		return StatusBarMsg{}
	}
}

// createDir enters insert mode and triggers directory creation
func (vim *Vim) createDir(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.DirTree.Create(vim.app.Mode, vim.app.StatusBar)
	}
}

// createNote enters insert mode and triggers notes creation
func (vim *Vim) createNote(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.NotesList.Create(vim.app.Mode, vim.app.StatusBar)
	}
}

func (vim *Vim) enterCmdMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.Mode.Current != mode.Normal {
			return StatusBarMsg{}
		}

		vim.app.Editor.Mode.Current = mode.Command
		vim.app.Mode.Current = mode.Command
		vim.app.StatusBar.Focused = true
		vim.app.State.ResetIndex()

		return StatusBarMsg{
			Type:   message.Prompt,
			Column: sbc.General,
		}
	}
}

// showBufferList opens an overlay showing all open buffers
func (vim *Vim) showBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.OverlayOpenBuffers()
		return StatusBarMsg{}
	}
}

// closeBufferList opens an overlay showing all open buffers
func (vim *Vim) closeBufferList(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.BufferList.Focused() {
			vim.app.BufferList.Hide()
			vim.FocusColumn(vim.app.CurrColFocus)
			vim.app.CurrentOverlay = nil
			vim.app.BufferList.Blur()
		}
		return StatusBarMsg{}
	}
}

// confirmAction performs the primary action for the focused component,
// or loads note data into the editor if in normal mode.
func (vim *Vim) confirmAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := StatusBarMsg{}

		f := vim.focusedComponent()

		if vim.app.StatusBar.Focused {
			statusMsg = vim.app.StatusBar.ConfirmAction(statusMsg.Sender, f)
			vim.app.Editor.Mode.Current = mode.Normal
		}

		if vim.app.Mode.Current != mode.Normal &&
			!vim.app.StatusBar.Focused &&
			!vim.app.Editor.Focused() {

			statusMsg = f.ConfirmAction()
		} else {
			// only open stuff if we're in normal mode
			if vim.app.Mode.Current != mode.Normal {
				vim.app.Mode.Current = mode.Normal
				return statusMsg
			}

			switch f {
			case vim.app.DirTree:
				path := vim.app.DirTree.SelectedDir().Path()
				vim.app.NotesList.CurrentPath = path
				vim.app.Conf.SetMetaValue("", config.LastDirectory, path)
				statusMsg = vim.app.NotesList.Refresh(true, true)

			case vim.app.NotesList:
				if sel := vim.app.NotesList.SelectedItem(nil); sel != nil {
					statusMsg = vim.app.Editor.OpenBuffer(sel.Path())
				}

			case vim.app.BufferList:
				items := vim.app.BufferList.Items
				sel := items[vim.app.BufferList.SelectedIndex]

				statusMsg.Cmd = editor.SendSwitchBufferMsg(
					sel.Path(),
					true,
				)

				vim.app.BufferList.Blur()
				vim.app.CurrentOverlay = nil
			}
		}

		vim.app.Mode.Current = mode.Normal
		vim.app.UpdateComponents(statusMsg)

		return statusMsg
	}
}

// cancelAction resets mode to normal
// and cancels pending actions in the focused component.
func (vim *Vim) cancelAction(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.Mode.Current = mode.Normal
		vim.app.StatusBar.Focused = false

		if vim.app.StatusBar.Prompt.Focused() {
			vim.app.StatusBar.CancelAction(func() {})
			return vim.enterNormalMode(opts)()
		} else {
			if f := vim.focusedComponent(); f != nil {
				resetIndex := false
				stateCreate := shared.EditStates.Create

				if vim.app.DirTree.EditState == stateCreate ||
					vim.app.NotesList.EditState == stateCreate {
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

func (vim *Vim) deleteSelectedBuffer(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if !vim.app.BufferList.Focused() {
			return StatusBarMsg{}
		}

		if buffer := vim.app.BufferList.SelectedBuffer(); buffer != nil {
			vim.app.Editor.DeleteBuffer(buffer.Path())
		}

		return StatusBarMsg{}
	}
}

func (vim *Vim) newScratchBuffer(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := vim.app.Editor.NewScratchBuffer("Scratch", "")
		vim.app.Editor.Textarea.SetValue("")
		return statusMsg
	}
}

func (vim *Vim) enterNormalMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.Editor.Mode.Current = mode.Normal
		vim.app.Mode.Current = mode.Normal
		vim.app.StatusBar.Focused = false

		return StatusBarMsg{
			Content: "",
			Type:    message.Prompt,
			Column:  sbc.General,
		}
	}
}

func (vim *Vim) toggleVisual(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.Editor.Mode.Current == mode.Visual {
			return vim.app.Editor.EnterNormalMode(true)
		}

		return vim.app.Editor.EnterVisualMode(textarea.SelectVisual)
	}
}

func (vim *Vim) toggleVisualLine(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.Editor.Mode.Current == mode.VisualLine {
			return vim.app.Editor.EnterNormalMode(true)
		}

		return vim.app.Editor.EnterVisualMode(textarea.SelectVisualLine)
	}
}

func (vim *Vim) toggleVisualBlock(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.Editor.Mode.Current == mode.VisualBlock {
			return vim.app.Editor.EnterNormalMode(true)
		}

		return vim.app.Editor.EnterVisualMode(textarea.SelectVisualBlock)
	}
}

func (vim *Vim) enterInsertMode(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.EnterInsertMode(true)
	}
}

func (vim *Vim) insertBelow(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.InsertLine(false)
	}
}

func (vim *Vim) insertAbove(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.InsertLine(true)
	}
}

func (vim *Vim) selectWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool(ki.Args.Outer)
		return vim.app.Editor.SelectWord(outer)
	}
}

func (vim *Vim) nextWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool(ki.Args.End)
		return vim.app.Editor.WordForward(end)
	}
}

func (vim *Vim) prevWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		end := opts.GetBool(ki.Args.End)
		return vim.app.Editor.WordBack(end)
	}
}

func (vim *Vim) findCharacter(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		action := *vim.KeyMap.AwaitInputAction
		binding := []rune(action.Binding())

		prev := action.Opts().GetBool(ki.Args.Prev)
		input := action.Opts().GetBool(ki.Args.Insert)
		char := vim.KeyMap.KeySequence[len(binding):]

		vim.app.Editor.FindCharacter(char, prev)

		if input {
			vim.app.Editor.EnterInsertMode(true)
		}

		return message.StatusBarMsg{}
	}
}

func (vim *Vim) findWordUnderCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if len(vim.app.Editor.Textarea.Search.Matches) > 0 {
			vim.app.Editor.Textarea.FindNextMatch()
			return message.StatusBarMsg{}
		} else {

			vim.app.Editor.Textarea.SelectInnerWord()
			word := vim.app.Editor.Textarea.SelectionStr()
			vim.app.Editor.Textarea.ResetSelection()

			if !unicode.IsLetter([]rune(word)[0]) {
				vim.app.Editor.Textarea.WordRight()
				return vim.findWordUnderCursor(opts)()
			}

			vim.app.Editor.Textarea.Search = textarea.Search{
				IgnoreCase: opts.GetBool(ki.Args.IgnoreCase),
				Matches:    make(map[int][]int, 1),
				Query:      word,
				ExactWord:  true,
			}

			vim.app.Editor.Mode.Current = mode.SearchPrompt
			vim.app.Mode.Current = mode.SearchPrompt

			return StatusBarMsg{
				Type:   message.Prompt,
				Column: sbc.General,
				Cmd:    vim.app.Editor.SendSearchConfirmedMsg(true),
			}
		}
	}
}

func (vim *Vim) find(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.Editor.Textarea.Search = textarea.Search{
			IgnoreCase: opts.GetBool(ki.Args.IgnoreCase),
			Matches:    make(map[int][]int, 1),
		}

		vim.app.Editor.Mode.Current = mode.SearchPrompt
		vim.app.Mode.Current = mode.SearchPrompt
		vim.app.StatusBar.Focused = true

		return StatusBarMsg{
			Type:   message.Prompt,
			Column: sbc.General,
		}
	}
}

func (vim *Vim) moveToMatch(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if opts.GetBool("prev") {
			vim.app.Editor.Textarea.FindPrevMatch()
		} else {
			vim.app.Editor.Textarea.FindNextMatch()
		}
		return StatusBarMsg{}
	}
}

func (vim *Vim) deleteWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool(ki.Args.Outer)

		if opts.GetBool(ki.Args.Remaining) {
			return vim.app.Editor.DeleteWordRight()
		}

		return vim.app.Editor.DeleteWord(outer, false)
	}
}

func (vim *Vim) deleteAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.DeleteAfterCursor(false)
	}
}

func (vim *Vim) deleteSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		if vim.app.Mode.IsAnyVisual() {
			return vim.app.Editor.DeleteRune(false, true, false)
		}
		return StatusBarMsg{}
	}
}

func (vim *Vim) deleteCharacter(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.DeleteRune(false, true, false)
	}
}

func (vim *Vim) deleteFromCursorToChar(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		action := *vim.KeyMap.AwaitInputAction
		binding := []rune(action.Binding())

		includeChar := action.Opts().GetBool(ki.Args.Include)
		prev := action.Opts().GetBool(ki.Args.Prev)
		char := vim.KeyMap.KeySequence[len(binding):]

		if ok := vim.app.Editor.Textarea.DeleteFromCursorToChar(
			char,
			includeChar,
			prev,
		); ok {
			if action.Opts().GetBool(ki.Args.Insert) {
				vim.app.Editor.EnterInsertMode(true)
			}
		}

		return message.StatusBarMsg{}
	}
}

func (vim *Vim) substituteText(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := vim.app.Editor.DeleteRune(false, true, false)

		if opts.GetBool(ki.Args.NewLine) {
			vim.app.Editor.Textarea.EmptyLineAbove()
		}

		vim.app.Editor.EnterInsertMode(false)

		return msg
	}
}

func (vim *Vim) changeAfterCursor(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.Editor.DeleteAfterCursor(true)
		vim.app.Editor.MoveCharacterRight()
		vim.app.Editor.EnterInsertMode(false)

		return StatusBarMsg{}
	}
}

func (vim *Vim) changeLine(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		vim.app.Editor.GoToLineStart()
		vim.app.Editor.DeleteAfterCursor(false)
		vim.app.Editor.EnterInsertMode(false)

		return vim.app.Editor.ResetSelectedRowsCount()
	}
}

func (vim *Vim) changeWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool(ki.Args.Outer)

		if opts.GetBool(ki.Args.Remaining) {
			vim.app.Editor.DeleteWordRight()
			return vim.app.Editor.EnterInsertMode(false)
		}

		return vim.app.Editor.DeleteWord(outer, true)
	}
}

func (vim *Vim) yankSelection(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.YankSelection(false)
	}
}

func (vim *Vim) yankWord(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		outer := opts.GetBool("outer")
		return vim.app.Editor.YankWord(outer)
	}
}

func (vim *Vim) paste(opts ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		// If we're in any visual mode delete the selection without
		// history and yanking right away
		//
		// @todo when undoing this step the cursor is one character left of
		// the word we replace - fix this
		if vim.app.Mode.IsAnyVisual() {
			vim.app.Editor.DeleteRune(true, false, true)
			vim.app.Editor.MoveCharacterLeft()
		}

		msg := vim.app.Editor.Paste()

		vim.selectWord(opts)
		return msg
	}
}

func (vim *Vim) changeToLowerCase(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.ChangeCaseOfSelection(false)
	}
}

func (vim *Vim) changeToUpperCase(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		return vim.app.Editor.ChangeCaseOfSelection(true)
	}
}

func (vim *Vim) OverlayOpenBuffers() {
	ov := vim.app.BufferList.Overlay

	vim.app.BufferList.Show()
	vim.app.BufferList.Focus()
	vim.app.CurrentOverlay = ov
	vim.app.UpdateComponents(false)
}

// FocusColumn selects and higlights a column with index `index`
// (1=dirTree, 2=notesList, 3=editor)
func (vim *Vim) FocusColumn(index int) StatusBarMsg {
	vim.app.Conf.SetMetaValue("", config.CurrentComponent, strconv.Itoa(index))

	vim.app.DirTree.SetFocus(index == 1)
	vim.app.DirTree.BuildHeader(vim.app.DirTree.Size.Width, true)

	vim.app.NotesList.SetFocus(index == 2)
	vim.app.NotesList.BuildHeader(vim.app.NotesList.Size.Width, true)

	vim.app.Editor.SetFocus(index == 3)
	vim.app.Editor.BuildHeader(vim.app.Editor.Size.Width, true)

	vim.app.CurrColFocus = index
	vim.KeyMap.FetchKeyMap(true)

	if index == 3 {
		relPath := utils.RelativePath(vim.app.Editor.CurrentBuffer.Path(false), true)
		icon := theme.Icon(theme.IconNote, vim.app.Conf.NerdFonts())
		return StatusBarMsg{
			Content: icon + " " + relPath,
			Column:  sbc.FileInfo,
		}
	}

	return StatusBarMsg{}
}

// focusedComponent returns the component that is currently focused
func (vim *Vim) focusedComponent() interfaces.Focusable {
	if vim.app.DirTree.Focused() {
		return vim.app.DirTree
	}

	if vim.app.NotesList.Focused() {
		return vim.app.NotesList
	}

	if vim.app.BufferList.Focused() {
		return vim.app.BufferList
	}

	return nil
}

func (vim *Vim) UnfocusAllColumns() StatusBarMsg {
	vim.app.DirTree.Blur()
	vim.app.DirTree.BuildHeader(vim.app.DirTree.Size.Width, true)

	vim.app.NotesList.Blur()
	vim.app.NotesList.BuildHeader(vim.app.NotesList.Size.Width, true)

	vim.app.Editor.Blur()
	vim.app.Editor.BuildHeader(vim.app.Editor.Size.Width, true)

	vim.KeyMap.FetchKeyMap(true)

	vim.app.StatusBar.Focused = false

	return StatusBarMsg{}
}
