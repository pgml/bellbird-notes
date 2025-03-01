package statusbar

import (
	"bellbird-notes/internal/tui/directorytree"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	Content   string
	Type      messages.MsgType
	Prompt    textinput.Model
	Mode      mode.Mode
	Sender    messages.Sender
	SenderMsg messages.StatusBarMsg
	DirTree   directorytree.DirectoryTree
}

const (
	ResponseYES = "y"
	ResponseNO  = "n"
)

func New() *StatusBar {
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
	if s.Mode == mode.Normal {
		s.Content = msg.Content
		s.Type = msg.Type
	}

	s.Sender = msg.Sender

	switch teaMsg.(type) {
	case tea.KeyMsg:
		if s.Mode == mode.Insert && s.Prompt.Focused() {
			s.Prompt, _ = s.Prompt.Update(teaMsg)
			return s
		}
		if s.Mode == mode.Normal {
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

func (s *StatusBar) ConfirmAction() messages.StatusBarMsg {
	if s.Prompt.Focused() {
		switch s.Prompt.Value() {
		case ResponseYES:
			if s.Sender == messages.SenderDirTree {
				return s.DirTree.Remove()
			}
		case ResponseNO:
			if s.Sender == messages.SenderDirTree {
				return s.DirTree.CancelAction()
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
