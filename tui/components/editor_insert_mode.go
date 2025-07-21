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
			tabStr := strings.Repeat(string(input.KeySpace), e.Textarea.TabWidth)
			msg = tea.KeyPressMsg{
				Text:        tabStr,
				Mod:         k.Mod,
				Code:        k.Code,
				ShiftedCode: k.ShiftedCode,
				BaseCode:    k.BaseCode,
				IsRepeat:    k.IsRepeat,
			}
		}
		//debug.LogDebug(
		//	msg.Key().Keystroke(), ",",
		//	msg.Key().Text, ",",
		//	msg.Key().BaseCode, ",",
		//	msg.Key().Code, ",",
		//	msg.Key().ShiftedCode,
		//)
		e.Textarea, cmd = e.Textarea.Update(msg)
	}

	return cmd
}
