package statusbar

import (
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	Content string
	Type    messages.MsgType
	Prompt  textinput.Model
}

func New() *StatusBar {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	return &StatusBar{
		Prompt: ti,
	}
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(msg messages.StatusBarMsg) *StatusBar {
	s.Content = msg.Content
	s.Type = msg.Type
	return s
}

func (s *StatusBar) View() string {
	style := style().Foreground(s.Type.Colour())
	output := s.Content

	if s.Type == messages.PromptError {
		output += s.Prompt.View() + " yep"
	}

	return style.Render(output)
}

func style() lipgloss.Style {
	termWidth, _ := theme.GetTerminalSize()
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1).
		Width(termWidth)
}
