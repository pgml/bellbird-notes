package tui

import (
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/message"
)

func (m *Model) CmdRegistry() components.Commands {
	return components.Commands{
		message.CmdPrompt.Yes:       m.statusBarConfirm,
		message.CmdPrompt.No:        m.statusBarCancel,
		message.CmdPrompt.Quit:      m.shouldQuit,
		message.CmdPrompt.WriteBuf:  m.writeBuffer,
		message.CmdPrompt.WriteQuit: m.writeBufferAndQuit,

		message.CmdPrompt.Set:  m.cmdSet,
		message.CmdPrompt.Open: m.cmdOpen,

		message.CmdPrompt.DeleteBufstring: m.deleteCurrentBuffer,
		"%bd":                             m.deleteAllBuffers,
		message.CmdPrompt.ListBufs:        m.listBuffers,
		"buffers":                         m.listBuffers,

		message.CmdPrompt.New: m.newScratchBuffer,
	}
}

func (m *Model) cmdSetRegistry() components.Commands {
	return components.Commands{
		"number":   m.setNumber,
		"nonumber": m.setNoNumber,
	}
}

func (m *Model) cmdOpenRegistry() components.Commands {
	return components.Commands{
		"config":        m.openConfig,
		"keymap":        m.openKeyMap,
		"defaultkeymap": m.openDefaultKeyMap,
	}
}

func (m *Model) cmdSet(args ...string) StatusBarMsg {
	fns := m.cmdSetRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (m *Model) cmdOpen(args ...string) StatusBarMsg {
	fns := m.cmdOpenRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (m *Model) statusBarConfirm(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := m.focusedComponent(); f != nil {
		if m.dirTree.EditState == components.EditStates.Delete ||
			m.notesList.EditState == components.EditStates.Delete {

			msg = f.Remove()
		}
	}
	return msg
}

func (m *Model) statusBarCancel(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := m.focusedComponent(); f != nil {
		msg = f.CancelAction(func() {
			f.Refresh(false, false)
		})
	}
	return msg
}

func (m *Model) shouldQuit(_ ...string) StatusBarMsg {
	m.ShouldQuit = true
	return StatusBarMsg{}
}

func (m *Model) writeBuffer(_ ...string) StatusBarMsg {
	return m.editor.SaveBuffer()
}

func (m *Model) openConfig(_ ...string) StatusBarMsg {
	return m.editor.OpenConfig()
}

func (m *Model) openKeyMap(_ ...string) StatusBarMsg {
	return m.editor.OpenUserKeyMap()
}

func (m *Model) deleteCurrentBuffer(_ ...string) StatusBarMsg {
	return m.editor.DeleteCurrentBuffer()
}

func (m *Model) deleteAllBuffers(_ ...string) StatusBarMsg {
	return m.editor.DeleteAllBuffers()
}

func (m *Model) writeBufferAndQuit(_ ...string) StatusBarMsg {
	m.editor.SaveBuffer()
	m.ShouldQuit = true
	return StatusBarMsg{}
}

func (m *Model) setNumber(_ ...string) StatusBarMsg {
	m.editor.SetNumbers()
	return StatusBarMsg{}
}

func (m *Model) setNoNumber(_ ...string) StatusBarMsg {
	m.editor.SetNoNumbers()
	return StatusBarMsg{}
}

func (m *Model) openDefaultKeyMap(_ ...string) StatusBarMsg {
	statusMsg := m.editor.NewScratchBuffer(
		"Default Keymap",
		string(m.keyInput.DefaultKeyMap),
	)
	m.editor.CurrentBuffer.Writeable = false
	m.editor.Textarea.MoveCursor(0, 0, 0)
	m.editor.SetContent()

	return statusMsg
}

func (m *Model) listBuffers(_ ...string) StatusBarMsg {
	m.editor.ListBuffers = true
	return StatusBarMsg{}
}

func (m *Model) newScratchBuffer(_ ...string) StatusBarMsg {
	statusMsg := m.editor.NewScratchBuffer("Scratch", "")
	m.editor.Textarea.SetValue("")
	return statusMsg
}
