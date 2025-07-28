package components

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type SearchMsg struct {
	SearchTerm string
	IgnoreCase bool
}

type SearchConfirmedMsg struct {
	Search string
}

type SearchCancelMsg struct{}

func (e *Editor) handleSearchMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		e.EnterNormalMode(true)
		e.Textarea.ResetMultiSelection()

		cmd = e.SendCancelMsg()

	case "backspace":
		if e.Textarea.Search.Query == "" {
			e.CancelSearch()
			cmd = e.SendCancelMsg()
		}

	case "enter":
		cmd = e.SendConfirmedMsg()
	}

	return cmd
}

func (e *Editor) CancelSearch() {
	e.Textarea.ResetMultiSelection()
	e.EnterNormalMode(false)
}

func (e *Editor) SendConfirmedMsg() tea.Cmd {
	return func() tea.Msg {
		return SearchConfirmedMsg{
			Search: e.Textarea.Search.Query,
		}
	}
}

func (e *Editor) SendCancelMsg() tea.Cmd {
	return func() tea.Msg {
		return SearchCancelMsg{}
	}
}
