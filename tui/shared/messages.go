package shared

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type RefreshUiMsg struct{}

func SendRefreshUiMsg() tea.Cmd {
	return func() tea.Msg {
		return RefreshUiMsg{}
	}
}

type DeferredActionMsg struct{}
