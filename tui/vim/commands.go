package vim

import (
	"bellbird-notes/tui/components/statusbar"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/shared"
)

type Commands = statusbar.Commands

func (vim *Vim) CmdRegistry() Commands {
	return Commands{
		message.CmdPrompt.Yes:       vim.statusBarConfirm,
		message.CmdPrompt.No:        vim.statusBarCancel,
		message.CmdPrompt.Quit:      vim.shouldQuit,
		message.CmdPrompt.WriteBuf:  vim.writeBuffer,
		message.CmdPrompt.WriteQuit: vim.writeBufferAndQuit,

		message.CmdPrompt.Set:       vim.cmdSet,
		message.CmdPrompt.Open:      vim.cmdOpen,
		message.CmdPrompt.Reload:    vim.cmdReload,
		message.CmdPrompt.CheckTime: vim.cmdCheckTime,

		message.CmdPrompt.DeleteBufstring: vim.deleteCurrentBuffer,
		"%bd":                             vim.deleteAllBuffers,
		message.CmdPrompt.ListBufs:        vim.listBuffers,
		"buffers":                         vim.listBuffers,

		message.CmdPrompt.New: vim.cmdNewScratchBuffer,

		"ToggleFolders": func(_ ...string) StatusBarMsg {
			return vim.app.DirTree.Toggle()
		},
		"ToggleTreeIndentLines": func(_ ...string) StatusBarMsg {
			return vim.app.DirTree.ToggleIndentLines()
		},
		"ToggleNotes": func(_ ...string) StatusBarMsg {
			return vim.app.NotesList.Toggle()
		},
	}
}

func (vim *Vim) cmdSetRegistry() Commands {
	return Commands{
		"number":   vim.setNumber,
		"nonumber": vim.setNoNumber,
	}
}

func (vim *Vim) cmdOpenRegistry() Commands {
	return Commands{
		"config":        vim.openConfig,
		"keymap":        vim.openKeyMap,
		"defaultkeymap": vim.openDefaultKeyMap,
	}
}

func (vim *Vim) cmdReloadRegistry() Commands {
	return Commands{
		"config": vim.reloadConfig,
		"keymap": vim.reloadKeyMap,
	}
}

func (vim *Vim) cmdSet(args ...string) StatusBarMsg {
	fns := vim.cmdSetRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (vim *Vim) cmdOpen(args ...string) StatusBarMsg {
	fns := vim.cmdOpenRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (vim *Vim) cmdReload(args ...string) StatusBarMsg {
	fns := vim.cmdReloadRegistry()
	if fn, ok := fns[args[0]]; ok {
		return fn()
	}
	return StatusBarMsg{}
}

func (vim *Vim) cmdCheckTime(_ ...string) StatusBarMsg {
	vim.app.Editor.CheckTime()
	return StatusBarMsg{}
}

func (vim *Vim) statusBarConfirm(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := vim.focusedComponent(); f != nil {
		if vim.app.DirTree.EditState == shared.EditStates.Delete ||
			vim.app.NotesList.EditState == shared.EditStates.Delete {

			msg = f.Remove()
		}
	}
	return msg
}

func (vim *Vim) statusBarCancel(_ ...string) StatusBarMsg {
	msg := StatusBarMsg{}
	if f := vim.focusedComponent(); f != nil {
		msg = f.CancelAction(func() {
			f.Refresh(false, false)
		})
	}
	return msg
}

func (vim *Vim) shouldQuit(_ ...string) StatusBarMsg {
	vim.app.ShouldQuit = true
	return StatusBarMsg{}
}

func (vim *Vim) writeBuffer(_ ...string) StatusBarMsg {
	return vim.app.Editor.SaveBuffer()
}

func (vim *Vim) openConfig(_ ...string) StatusBarMsg {
	return vim.app.Editor.OpenConfig()
}

func (vim *Vim) openKeyMap(_ ...string) StatusBarMsg {
	return vim.app.Editor.OpenUserKeyMap()
}

func (vim *Vim) reloadConfig(_ ...string) StatusBarMsg {
	return StatusBarMsg{
		Cmd: shared.SendRefreshUiMsg(),
	}
}

func (vim *Vim) reloadKeyMap(_ ...string) StatusBarMsg {
	vim.app.KeyInput.ReloadKeyMap()
	return StatusBarMsg{}
}

func (vim *Vim) deleteCurrentBuffer(_ ...string) StatusBarMsg {
	return vim.app.Editor.DeleteCurrentBuffer()
}

func (vim *Vim) deleteAllBuffers(_ ...string) StatusBarMsg {
	return vim.app.Editor.DeleteAllBuffers()
}

func (vim *Vim) writeBufferAndQuit(_ ...string) StatusBarMsg {
	vim.app.Editor.SaveBuffer()
	vim.app.ShouldQuit = true
	return StatusBarMsg{}
}

func (vim *Vim) setNumber(_ ...string) StatusBarMsg {
	vim.app.Editor.SetNumbers()
	return StatusBarMsg{}
}

func (vim *Vim) setNoNumber(_ ...string) StatusBarMsg {
	vim.app.Editor.SetNoNumbers()
	return StatusBarMsg{}
}

func (vim *Vim) openDefaultKeyMap(_ ...string) StatusBarMsg {
	statusMsg := vim.app.Editor.NewScratchBuffer(
		"Default Keymap",
		string(vim.KeyMap.DefaultKeyMap),
	)
	vim.app.Editor.CurrentBuffer.Writeable = false
	vim.app.Editor.Textarea.MoveCursor(0, 0, 0)
	vim.app.Editor.SetContent()

	return statusMsg
}

func (vim *Vim) listBuffers(_ ...string) StatusBarMsg {
	vim.app.BufferList.Show()
	return StatusBarMsg{}
}

func (vim *Vim) cmdNewScratchBuffer(_ ...string) StatusBarMsg {
	statusMsg := vim.app.Editor.NewScratchBuffer("Scratch", "")
	vim.app.Editor.Textarea.SetValue("")
	return statusMsg
}
