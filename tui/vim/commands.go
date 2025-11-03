package vim

import (
	"bellbird-notes/tui/components/statusbar"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/shared"
)

type Commands = statusbar.Commands

func (v *Vim) CmdRegistry() Commands {
	return Commands{
		message.CmdPrompt.Yes:       v.statusBarConfirm,
		message.CmdPrompt.No:        v.statusBarCancel,
		message.CmdPrompt.Quit:      v.shouldQuit,
		message.CmdPrompt.WriteBuf:  v.writeBuffer,
		message.CmdPrompt.WriteQuit: v.writeBufferAndQuit,

		message.CmdPrompt.Set:       v.cmdSet,
		message.CmdPrompt.Open:      v.cmdOpen,
		message.CmdPrompt.Reload:    v.cmdReload,
		message.CmdPrompt.CheckTime: v.cmdCheckTime,

		message.CmdPrompt.DeleteBufstring: v.deleteCurrentBuffer,
		"%bd":                             v.deleteAllBuffers,
		message.CmdPrompt.ListBufs:        v.listBuffers,
		"buffers":                         v.listBuffers,

		message.CmdPrompt.New: v.cmdNewScratchBuffer,

		"ToggleFolders": func(_ ...string) StatusBarMsg {
			return v.app.DirTree.Toggle()
		},
		"ToggleTreeIndentLines": func(_ ...string) StatusBarMsg {
			return v.app.DirTree.ToggleIndentLines()
		},
		"ToggleNotes": func(_ ...string) StatusBarMsg {
			return v.app.NotesList.Toggle()
		},
	}
}

func (v *Vim) cmdSetRegistry() Commands {
	return Commands{
		"number":   v.setNumber,
		"nonumber": v.setNoNumber,
	}
}

func (v *Vim) cmdOpenRegistry() Commands {
	return Commands{
		"config":        v.openConfig,
		"keymap":        v.openKeyMap,
		"defaultkeymap": v.openDefaultKeyMap,
	}
}

func (v *Vim) cmdReloadRegistry() Commands {
	return Commands{
		"config": v.reloadConfig,
		"keymap": v.reloadKeyMap,
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

func (v *Vim) cmdReload(args ...string) StatusBarMsg {
	fns := v.cmdReloadRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (v *Vim) cmdCheckTime(_ ...string) StatusBarMsg {
	v.app.Editor.CheckTime()
	return StatusBarMsg{}
}

func (v *Vim) statusBarConfirm(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := v.focusedComponent(); f != nil {
		if v.app.DirTree.EditState == shared.EditStates.Delete ||
			v.app.NotesList.EditState == shared.EditStates.Delete {

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

func (v *Vim) reloadConfig(_ ...string) StatusBarMsg {
	return StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (v *Vim) reloadKeyMap(_ ...string) StatusBarMsg {
	v.app.KeyInput.ReloadKeyMap()
	return StatusBarMsg{}
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
