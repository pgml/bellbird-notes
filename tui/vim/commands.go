package vim

import (
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/message"
)

func (v *Vim) CmdRegistry() components.Commands {
	return components.Commands{
		message.CmdPrompt.Yes:       v.statusBarConfirm,
		message.CmdPrompt.No:        v.statusBarCancel,
		message.CmdPrompt.Quit:      v.shouldQuit,
		message.CmdPrompt.WriteBuf:  v.writeBuffer,
		message.CmdPrompt.WriteQuit: v.writeBufferAndQuit,

		message.CmdPrompt.Set:  v.cmdSet,
		message.CmdPrompt.Open: v.cmdOpen,

		message.CmdPrompt.DeleteBufstring: v.deleteCurrentBuffer,
		"%bd":                             v.deleteAllBuffers,
		message.CmdPrompt.ListBufs:        v.listBuffers,
		"buffers":                         v.listBuffers,

		message.CmdPrompt.New: v.cmdNewScratchBuffer,
	}
}

func (v *Vim) cmdSetRegistry() components.Commands {
	return components.Commands{
		"number":   v.setNumber,
		"nonumber": v.setNoNumber,
	}
}

func (v *Vim) cmdOpenRegistry() components.Commands {
	return components.Commands{
		"config":        v.openConfig,
		"keymap":        v.openKeyMap,
		"defaultkeymap": v.openDefaultKeyMap,
	}
}

func (v *Vim) cmdSet(args ...string) StatusBarMsg {
	fns := v.cmdSetRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (v *Vim) cmdOpen(args ...string) StatusBarMsg {
	fns := v.cmdOpenRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (v *Vim) statusBarConfirm(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := v.focusedComponent(); f != nil {
		if v.app.DirTree.EditState == components.EditStates.Delete ||
			v.app.NotesList.EditState == components.EditStates.Delete {

			msg = f.Remove()
		}
	}
	return msg
}

func (v *Vim) statusBarCancel(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := v.focusedComponent(); f != nil {
		msg = f.CancelAction(func() {
			f.Refresh(false, false)
		})
	}
	return msg
}

func (v *Vim) shouldQuit(_ ...string) StatusBarMsg {
	v.app.ShouldQuit = true
	return StatusBarMsg{}
}

func (v *Vim) writeBuffer(_ ...string) StatusBarMsg {
	return v.app.Editor.SaveBuffer()
}

func (v *Vim) openConfig(_ ...string) StatusBarMsg {
	return v.app.Editor.OpenConfig()
}

func (v *Vim) openKeyMap(_ ...string) StatusBarMsg {
	return v.app.Editor.OpenUserKeyMap()
}

func (v *Vim) deleteCurrentBuffer(_ ...string) StatusBarMsg {
	return v.app.Editor.DeleteCurrentBuffer()
}

func (v *Vim) deleteAllBuffers(_ ...string) StatusBarMsg {
	return v.app.Editor.DeleteAllBuffers()
}

func (v *Vim) writeBufferAndQuit(_ ...string) StatusBarMsg {
	v.app.Editor.SaveBuffer()
	v.app.ShouldQuit = true
	return StatusBarMsg{}
}

func (v *Vim) setNumber(_ ...string) StatusBarMsg {
	v.app.Editor.SetNumbers()
	return StatusBarMsg{}
}

func (v *Vim) setNoNumber(_ ...string) StatusBarMsg {
	v.app.Editor.SetNoNumbers()
	return StatusBarMsg{}
}

func (v *Vim) openDefaultKeyMap(_ ...string) StatusBarMsg {
	statusMsg := v.app.Editor.NewScratchBuffer(
		"Default Keymap",
		string(v.KeyMap.DefaultKeyMap),
	)
	v.app.Editor.CurrentBuffer.Writeable = false
	v.app.Editor.Textarea.MoveCursor(0, 0, 0)
	v.app.Editor.SetContent()

	return statusMsg
}

func (v *Vim) listBuffers(_ ...string) StatusBarMsg {
	v.app.Editor.ListBuffers = true
	return StatusBarMsg{}
}

func (v *Vim) cmdNewScratchBuffer(_ ...string) StatusBarMsg {
	statusMsg := v.app.Editor.NewScratchBuffer("Scratch", "")
	v.app.Editor.Textarea.SetValue("")
	return statusMsg
}
