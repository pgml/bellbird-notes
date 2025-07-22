package tui

import (
	"bellbird-notes/tui/components"
	ki "bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/message"
)

func (m *Model) CmdRegistry() ki.FnRegistry {
	return ki.FnRegistry{
		message.Response.Yes:       m.statusBarConfirm,
		message.Response.No:        m.statusBarCancel,
		message.Response.Quit:      m.shouldQuit,
		message.Response.WriteBuf:  bind(m.editor.SaveBuffer),
		message.Response.WriteQuit: m.writeBufferAndQuit,

		"set number":   m.setNumber,
		"set nonumber": m.setNoNumber,

		// @todo: improve the open command to be more flexible
		// and potentially open any file
		"open config":        bind(m.editor.OpenConfig),
		"open keymap":        bind(m.editor.OpenUserKeyMap),
		"open defaultkeymap": m.openDefaultKeyMap,

		message.Response.DeleteBufstring: bind(m.editor.DeleteCurrentBuffer),
		"%bd":                            bind(m.editor.DeleteAllBuffers),
		message.Response.ListBufs:        m.listBuffers,
		"buffers":                        m.listBuffers,

		"new": m.newScratchBuffer,
	}
}

func (m *Model) SetCmd() {

}

func (m *Model) statusBarConfirm(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}
		if f := m.focusedComponent(); f != nil {
			if m.dirTree.EditState == components.EditStates.Delete ||
				m.notesList.EditState == components.EditStates.Delete {

				msg = f.Remove()
			}
		}
		return msg
	}
}

func (m *Model) statusBarCancel(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		msg := StatusBarMsg{}
		if f := m.focusedComponent(); f != nil {
			msg = f.CancelAction(func() {
				f.Refresh(false, false)
			})
		}
		return msg
	}
}

func (m *Model) shouldQuit(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.statusBar.ShouldQuit = true
		return StatusBarMsg{}
	}
}

func (m *Model) writeBufferAndQuit(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.SaveBuffer()
		m.statusBar.ShouldQuit = true
		return StatusBarMsg{}
	}
}

func (m *Model) setNumber(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.SetNumbers()
		return StatusBarMsg{}
	}
}

func (m *Model) setNoNumber(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.SetNoNumbers()
		return StatusBarMsg{}
	}
}

func (m *Model) openDefaultKeyMap(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := m.editor.NewScratchBuffer(
			"Default Keymap",
			string(m.keyInput.DefaultKeyMap),
		)
		m.editor.CurrentBuffer.Writeable = false
		m.editor.Textarea.MoveCursor(0, 0, 0)
		m.editor.SetContent()

		return statusMsg
	}
}

func (m *Model) listBuffers(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		m.editor.ListBuffers = true
		return StatusBarMsg{}
	}
}

func (m *Model) newScratchBuffer(_ ki.Options) func() StatusBarMsg {
	return func() StatusBarMsg {
		statusMsg := m.editor.NewScratchBuffer("Scratch", "")
		m.editor.Textarea.SetValue("")
		return statusMsg
	}
}
