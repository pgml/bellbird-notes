package editor

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type SearchMsg struct {
	SearchTerm string
	IgnoreCase bool
}

type SearchConfirmedMsg struct {
	Search      string
	ResetPrompt bool
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
			e.EnterNormalMode(false)
			cmd = e.SendCancelMsg()
		}

	case "enter":
		cmd = e.SendSearchConfirmedMsg(false)
	}

	return cmd
}

func (e *Editor) CancelSearch() {
	e.Textarea.ResetMultiSelection()
}

func (e *Editor) SendSearchConfirmedMsg(resetPrompt bool) tea.Cmd {
	return func() tea.Msg {
		return SearchConfirmedMsg{
			Search:      e.Textarea.Search.Query,
			ResetPrompt: resetPrompt,
		}
	}
}

func (e *Editor) SendCancelMsg() tea.Cmd {
	return func() tea.Msg {
		return SearchCancelMsg{}
	}
}
