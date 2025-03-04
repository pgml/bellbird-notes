package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBarMode int

const (
	Normal StatusBarMode = iota
	Insert
	Command
)

type StatusBar struct {
	Content string
	Type    messages.MsgType
	Prompt  textinput.Model
	// Indicates hether the directory tree column is focused.
	// Used to determine if the status bar should receive keyboard shortcuts
	Focused   bool
	Mode      app.Mode
	Sender    messages.Sender
	SenderMsg messages.StatusBarMsg
	DirTree   DirectoryTree
	NotesList NotesList
	Editor    Editor
}

const (
	ResponseYES = "y"
	ResponseNO  = "n"
)

func NewStatusBar() *StatusBar {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	return &StatusBar{
		Prompt: ti,
	}
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(msg messages.StatusBarMsg, teaMsg tea.Msg) *StatusBar {
	if s.Focused && s.Mode == app.NormalMode {
		s.Content = msg.Content
		s.Type = msg.Type
	}

	s.Sender = msg.Sender

	switch teaMsg.(type) {
	case tea.KeyMsg:
		if s.Focused && s.Mode == app.InsertMode && s.Prompt.Focused() {
			s.Prompt, _ = s.Prompt.Update(teaMsg)
			return s
		}
		if s.Mode == app.NormalMode {
			s.BlurPrompt()
		}
	}

	return s
}

func (s *StatusBar) View() string {
	style := style().Foreground(s.Type.Colour())
	output := s.Content

	if s.Type == messages.PromptError {
		s.Prompt.Focus()
		output += s.Prompt.View() + ""
	}

	return style.Render(output)
}

func (s *StatusBar) ConfirmAction(sender messages.Sender) messages.StatusBarMsg {
	if s.Prompt.Focused() {
		switch s.Prompt.Value() {
		case ResponseYES:
			if sender == messages.SenderDirTree {
				return s.DirTree.Remove()
			}
			if sender == messages.SenderNotesList {
				return s.NotesList.Remove()
			}
		case ResponseNO:
			if sender == messages.SenderDirTree {
				return s.DirTree.CancelAction(func() { s.DirTree.Refresh() })
			}
			if sender == messages.SenderNotesList {
				return s.NotesList.CancelAction(func() { s.DirTree.Refresh() })
			}
		}
	}
	s.BlurPrompt()
	return messages.StatusBarMsg{}
}

func (s *StatusBar) BlurPrompt() {
	s.Prompt.Blur()
	s.Prompt.SetValue("")
}

func style() lipgloss.Style {
	termWidth, _ := theme.GetTerminalSize()
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1).
		Width(termWidth)
}
