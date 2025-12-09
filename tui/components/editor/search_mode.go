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

func (editor *Editor) handleSearchMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		editor.EnterNormalMode(true)
		editor.Textarea.ResetMultiSelection()

		cmd = editor.SendCancelMsg()

	case "backspace":
		if editor.Textarea.Search.Query == "" {
			editor.CancelSearch()
			editor.EnterNormalMode(false)
			cmd = editor.SendCancelMsg()
		}

	case "enter":
		cmd = editor.SendSearchConfirmedMsg(false)
	}

	return cmd
}

func (editor *Editor) CancelSearch() {
	editor.Textarea.ResetMultiSelection()
}

func (editor *Editor) SendSearchConfirmedMsg(resetPrompt bool) tea.Cmd {
	return func() tea.Msg {
		return SearchConfirmedMsg{
			Search:      editor.Textarea.Search.Query,
			ResetPrompt: resetPrompt,
		}
	}
}

func (editor *Editor) SendCancelMsg() tea.Cmd {
	return func() tea.Msg {
		return SearchCancelMsg{}
	}
}
