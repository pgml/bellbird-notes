package tui

import (
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
)

type c = keyinput.FocusedComponent
type keyAction = keyinput.KeyAction
type keyCond = keyinput.KeyCondition

func (m *Model) KeyInputFn() []keyAction {
	return []keyAction{

		// LINE DOWN
		{
			Keys: "j",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.lineDown,
				},
				m.editorInputAction(m.editor.LineDown),
			},
		},

		// LINE UP
		{
			Keys: "k",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.lineUp,
				},
				m.editorInputAction(m.editor.LineUp),
			},
		},

		// TREE COLLAPSE, EDITOR CHARACTER LEFT
		{
			Keys: "h",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree},
					Action:     m.dirTree.Collapse,
				},
				m.editorInputAction(m.editor.MoveCharacterLeft),
			},
		},

		// TREE EXPAND, EDITOR CHARACTER RIGHT
		{
			Keys: "l",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree},
					Action:     m.dirTree.Expand,
				},
				m.editorInputAction(m.editor.MoveCharacterRight),
			},
		},

		// CREATE DIR
		{
			Keys: "d",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree},
				Action:     m.createDir,
			}},
		},
		{
			Keys: "n",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree},
				Action:     m.createDir,
			}},
		},

		// DELETE DIR/NOTE
		{
			Keys: "D",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.remove,
			}},
		},

		// CREATE NOTE
		{
			Keys: "%",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.notesList},
				Action:     m.createNote,
			}},
		},
		{
			Keys: "n",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.notesList},
				Action:     m.createNote,
			}},
		},

		// TREE/NOTE RENAME
		{
			Keys: "R",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList},
				Action:     m.rename,
			}},
		},

		{
			Keys: "gg",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.goToTop,
				},
				m.editorInputAction(m.editor.GoToTop),
			},
		},
		{
			Keys: "G",
			Cond: []keyCond{
				{
					Mode:       mode.Normal,
					Components: []c{m.dirTree, m.notesList},
					Action:     m.goToBottom,
				},
				m.editorInputAction(m.editor.GoToBottom),
			},
		},

		{
			Keys: "i",
			Cond: []keyCond{m.editorInputAction(m.editor.EnterInsertMode)},
		},
		{
			Keys: "I",
			Cond: []keyCond{m.editorInputAction(m.editor.InsertLineStart)},
		},
		{
			Keys: "a",
			Cond: []keyCond{m.editorInputAction(m.editor.InsertAfter)},
		},
		{
			Keys: "A",
			Cond: []keyCond{m.editorInputAction(m.editor.InsertLineEnd)},
		},
		{
			Keys: "r",
			Cond: []keyCond{m.editorInputAction(m.editor.EnterReplaceMode)},
		},
		{
			Keys: "u",
			Cond: []keyCond{m.editorInputAction(m.editor.Undo)},
		},
		{
			Keys: "ctrl+r",
			Cond: []keyCond{m.editorInputAction(m.editor.Redo)},
		},
		{
			Keys: "w",
			Cond: []keyCond{m.editorInputAction(m.editor.WordRightStart)},
		},
		{
			Keys: "e",
			Cond: []keyCond{m.editorInputAction(m.editor.WordRightEnd)},
		},
		{
			Keys: "b",
			Cond: []keyCond{m.editorInputAction(m.editor.WordBack)},
		},
		{
			Keys: "^",
			Cond: []keyCond{m.editorInputAction(m.editor.GoToInputStart)},
		},
		{
			Keys: "_",
			Cond: []keyCond{m.editorInputAction(m.editor.GoToInputStart)},
		},
		{
			Keys: "0",
			Cond: []keyCond{m.editorInputAction(m.editor.GoToLineStart)},
		},
		{
			Keys: "$",
			Cond: []keyCond{m.editorInputAction(m.editor.GoToLineEnd)},
		},
		{
			Keys: "o",
			Cond: []keyCond{m.editorInputAction(m.editor.InsertLineBelow)},
		},
		{
			Keys: "O",
			Cond: []keyCond{m.editorInputAction(m.editor.InsertLineAbove)},
		},
		{
			Keys: "dd",
			Cond: []keyCond{m.editorInputAction(m.editor.DeleteLine)},
		},
		{
			Keys: "dj",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, false)
				},
			}},
		},
		{
			Keys: "dk",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, true)
				},
			}},
		},
		{
			Keys: "dw",
			Cond: []keyCond{m.editorInputAction(m.editor.DeleteWordRight)},
		},
		{
			Keys: "D",
			Cond: []keyCond{m.editorInputAction(m.editor.DeleteAfterCursor)},
		},
		{
			Keys: "ctrl+d",
			Cond: []keyCond{m.editorInputAction(m.editor.DownHalfPage)},
		},
		{
			Keys: "ctrl+u",
			Cond: []keyCond{m.editorInputAction(m.editor.UpHalfPage)},
		},

		// ENTER CMD MODE
		{
			Keys: ":",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.enterCmdMode,
			}},
		},

		// CONFIRM ACTION
		{
			Keys: "enter",
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
			Keys: "esc",
			Cond: []keyCond{{
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
			Keys: "ctrl+w l",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusNextColumn,
			}},
		},
		{
			Keys: "ctrl+w h",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusPrevColumn,
			}},
		},
		{
			Keys: "1",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusDirectoryTree,
			}},
		},
		{
			Keys: "2",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusNotesList,
			}},
		},
		{
			Keys: "3",
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.dirTree, m.notesList, m.editor},
				Action:     m.focusEditor,
			}},
		},
	}
}

func (m *Model) editorInputAction(fn func() message.StatusBarMsg) keyCond {
	return keyCond{
		Mode:       mode.Normal,
		Components: []c{m.editor},
		Action:     fn,
	}
}
