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

	Columns []StatusBarColumn
}

const (
	ResponseYES = "y"
	ResponseNO  = "n"
)

type StatusBarColumn struct {
	content string
}

func NewStatusBar() *StatusBar {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	statusBar := &StatusBar{
		Prompt: ti,
	}

	statusBar.Columns = []StatusBarColumn{
		StatusBarColumn{},
		StatusBarColumn{},
		StatusBarColumn{},
		StatusBarColumn{},
	}

	return statusBar
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(msg messages.StatusBarMsg, teaMsg tea.Msg) *StatusBar {
	s.Columns[msg.Column].content = msg.Content
	if s.Focused && s.Mode == app.NormalMode {
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
		//case teaMsg.WindowSizeMsg:
		//	// Convert WindowSizeMsg to BubbleLayoutMsg.
		//	return s, func() teaMsg.(type) {
		//		return s.layout.Resize(teaMsg.Width, teaMsg.Height)
		//	}
	}

	//app.LogDebug(s.Columns[0].size.Width, s.Columns[1].size.Width)

	return s
}

func (s *StatusBar) View() string {
	style := style()
	//output := s.Content

	c1 := s.ModeView()
	c2 := style.Render(s.Columns[1].content)
	c3 := s.Columns[2].content
	c4 := s.Columns[3].content

	if s.Type == messages.PromptError {
		s.Prompt.Focus()
		c2 += s.Prompt.View() + ""
	}

	width, _ := theme.GetTerminalSize()
	c1width := 10
	if s.Prompt.Focused() {
		c1width = 0
	}
	c3width := 15
	c4width := 15
	c2width := max(width-(c1width+c3width+c4width), 1)

	return lipgloss.JoinHorizontal(lipgloss.Right,
		style.Width(c1width).Render(c1),
		style.Width(c2width).Foreground(s.Type.Colour()).Render(c2),
		style.Width(c3width).Align(lipgloss.Right).Render(c3),
		style.Width(c4width).Align(lipgloss.Right).Render(c4),
	)
}

func (s StatusBar) ModeView() string {
	style := lipgloss.NewStyle().
		Foreground(s.Mode.Colour()).
		PaddingLeft(1).
		PaddingRight(1)

	mode := s.Mode.FullString()
	if s.Prompt.Focused() {
		mode = ""
	}

	return style.Render(mode)
}

func (s *StatusBar) ConfirmAction(sender messages.Sender) messages.StatusBarMsg {
	if s.Prompt.Focused() {
		switch s.Prompt.Value() {
		case ResponseYES:
			s.BlurPrompt()
			if sender == messages.SenderDirTree {
				return s.DirTree.Remove()
			}
			if sender == messages.SenderNotesList {
				return s.NotesList.Remove()
			}
		case ResponseNO:
			s.BlurPrompt()
			s.Columns[1].content = ""
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
	s.Prompt.SetValue("")
	s.Prompt.Blur()
}

func style() lipgloss.Style {
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1)
}
