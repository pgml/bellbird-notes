package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bl "github.com/winder/bubblelayout"
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

	layout  bl.BubbleLayout
	Columns []StatusBarColumn
}

const (
	ResponseYES = "y"
	ResponseNO  = "n"
)

type StatusBarColumn struct {
	id      bl.ID
	size    bl.Size
	content string
}

func NewStatusBar() *StatusBar {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	statusBar := &StatusBar{
		layout: bl.New(),
		Prompt: ti,
	}

	statusBar.Columns = []StatusBarColumn{
		StatusBarColumn{id: statusBar.layout.Add("width 10")},
		StatusBarColumn{id: statusBar.layout.Add("grow")},
		StatusBarColumn{id: statusBar.layout.Add("width 10")},
	}

	return statusBar
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	resizeCmd := func() tea.Msg {
		return s.layout.Resize(80, 40)
	}

	return tea.Batch(resizeCmd)
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
		//case teaMsg.WindowSizeMsg:
		//	// Convert WindowSizeMsg to BubbleLayoutMsg.
		//	return s, func() teaMsg.(type) {
		//		return s.layout.Resize(teaMsg.Width, teaMsg.Height)
		//	}
	}

	app.LogDebug(s.Columns[0].size.Width, s.Columns[1].size.Width)

	return s
}

func (s *StatusBar) View() string {
	style := style().Foreground(s.Type.Colour())
	output := s.Content

	if s.Type == messages.PromptError {
		termWidth, _ := theme.GetTerminalSize()
		s.Prompt.Focus()
		s.Prompt.PromptStyle.Width(termWidth + 10)
		output += s.Prompt.View() + ""
	}

	rightCol := lipgloss.NewStyle().Align(lipgloss.Right).Background(lipgloss.Color("#f00")).
		Width(10).
		Render("saasdasdad")

	s.Columns[0].content = s.ModeView()
	s.Columns[1].content = style.Render(output)
	s.Columns[2].content = rightCol

	//app.LogDebug(s.Mode)

	return lipgloss.JoinHorizontal(lipgloss.Right,
		s.Columns[0].content,
		s.Columns[1].content,
		s.Columns[2].content,
	)
}

func (s StatusBar) ModeView() string {
	// hide mode if there's a prompt focused
	if s.Prompt.Focused() {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(s.Mode.Colour()).
		Width(10).
		PaddingLeft(1).
		PaddingRight(1)

	return style.Render(s.Mode.FullString())
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
	termWidth, _ := theme.GetTerminalSize()
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1).
		Width(termWidth - 20)
}
