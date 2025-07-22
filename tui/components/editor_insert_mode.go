package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/x/input"
)

func (e *Editor) handleInsertMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.EnterNormalMode(true)
		return nil
	}

	var cmd tea.Cmd

	// only allow input when this flag is true.
	// See tui.updateComponents() for further explanation
	if e.CanInsert {
		if msg.Key().Code == 9 {
			k := msg.Key()
			// Just for now, will be setting when tab support is finished
			tabWidth := 4
			// simulate soft tabs
			tabStr := strings.Repeat(string(input.KeySpace), tabWidth)
			msg = tea.KeyPressMsg{
				Text:        tabStr,
				Mod:         k.Mod,
				Code:        k.Code,
				ShiftedCode: k.ShiftedCode,
				BaseCode:    k.BaseCode,
				IsRepeat:    k.IsRepeat,
			}
		}
		e.Textarea, cmd = e.Textarea.Update(msg)
	}

	return cmd
}
