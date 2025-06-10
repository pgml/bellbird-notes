package tui

import (
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
)

type c = ki.FocusedComponent
type keyAction = ki.KeyAction
type keyCond = ki.KeyCondition
type binding = ki.KeyBinding

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
				m.editorInputAction(m.editor.LineDown),
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
				m.editorInputAction(m.editor.LineUp),
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
				m.editorInputAction(m.editor.MoveCharacterLeft),
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
				m.editorInputAction(m.editor.MoveCharacterRight),
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
				m.editorInputAction(m.editor.GoToTop),
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
				m.editorInputAction(m.editor.GoToBottom),
			},
		},

		{
			Bindings: ki.KeyBindings("i"),
			Cond:     []keyCond{m.editorInputAction(m.editor.EnterInsertMode)},
		},
		{
			Bindings: ki.KeyBindings("I"),
			Cond:     []keyCond{m.editorInputAction(m.editor.InsertLineStart)},
		},
		{
			Bindings: ki.KeyBindings("a"),
			Cond:     []keyCond{m.editorInputAction(m.editor.InsertAfter)},
		},
		{
			Bindings: ki.KeyBindings("A"),
			Cond:     []keyCond{m.editorInputAction(m.editor.InsertLineEnd)},
		},
		{
			Bindings: ki.KeyBindings("r"),
			Cond:     []keyCond{m.editorInputAction(m.editor.EnterReplaceMode)},
		},
		{
			Bindings: ki.KeyBindings("u"),
			Cond:     []keyCond{m.editorInputAction(m.editor.Undo)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+r"),
			Cond:     []keyCond{m.editorInputAction(m.editor.Redo)},
		},
		{
			Bindings: ki.KeyBindings("w"),
			Cond:     []keyCond{m.editorInputAction(m.editor.WordRightStart)},
		},
		{
			Bindings: ki.KeyBindings("e"),
			Cond:     []keyCond{m.editorInputAction(m.editor.WordRightEnd)},
		},
		{
			Bindings: ki.KeyBindings("b"),
			Cond:     []keyCond{m.editorInputAction(m.editor.WordBack)},
		},
		{
			Bindings: ki.KeyBindings("^", "_"),
			Cond:     []keyCond{m.editorInputAction(m.editor.GoToInputStart)},
		},
		{
			Bindings: ki.KeyBindings("0"),
			Cond:     []keyCond{m.editorInputAction(m.editor.GoToLineStart)},
		},
		{
			Bindings: ki.KeyBindings("$"),
			Cond:     []keyCond{m.editorInputAction(m.editor.GoToLineEnd)},
		},
		{
			Bindings: ki.KeyBindings("o"),
			Cond:     []keyCond{m.editorInputAction(m.editor.InsertLineBelow)},
		},
		{
			Bindings: ki.KeyBindings("O"),
			Cond:     []keyCond{m.editorInputAction(m.editor.InsertLineAbove)},
		},
		{
			Bindings: ki.KeyBindings("dd"),
			Cond:     []keyCond{m.editorInputAction(m.editor.DeleteLine)},
		},
		{
			Bindings: ki.KeyBindings("dj"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, false)
				},
			}},
		},
		{
			Bindings: ki.KeyBindings("dk"),
			Cond: []keyCond{{
				Mode:       mode.Normal,
				Components: []c{m.editor},
				Action: func() message.StatusBarMsg {
					return m.editor.DeleteNLines(2, true)
				},
			}},
		},
		{
			Bindings: ki.KeyBindings("dw"),
			Cond:     []keyCond{m.editorInputAction(m.editor.DeleteWordRight)},
		},
		{
			Bindings: ki.KeyBindings("D"),
			Cond:     []keyCond{m.editorInputAction(m.editor.DeleteAfterCursor)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+d"),
			Cond:     []keyCond{m.editorInputAction(m.editor.DownHalfPage)},
		},
		{
			Bindings: ki.KeyBindings("ctrl+u"),
			Cond:     []keyCond{m.editorInputAction(m.editor.UpHalfPage)},
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

func (m *Model) editorInputAction(fn func() message.StatusBarMsg) keyCond {
	return keyCond{
		Mode:       mode.Normal,
		Components: []c{m.editor},
		Action:     fn,
	}
}
