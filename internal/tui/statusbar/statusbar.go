package statusbar

import (
	"bellbird-notes/internal/tui/directorytree"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"bellbird-notes/internal/tui/noteslist"
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
	Mode      mode.Mode
	Sender    messages.Sender
	SenderMsg messages.StatusBarMsg
	DirTree   directorytree.DirectoryTree
	NotesList noteslist.NotesList
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
				return s.DirTree.CancelAction()
			}
			if sender == messages.SenderNotesList {
				return s.NotesList.CancelAction()
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
