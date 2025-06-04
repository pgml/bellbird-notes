package components

import (
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Focusable = interfaces.Focusable

type StatusBar struct {
	Content string
	Type    message.Type
	Prompt  textinput.Model
	// Indicates hether the directory tree column is focused.
	// Used to determine if the status bar should receive keyboard shortcuts
	Focused   bool
	Mode      mode.Mode
	Sender    message.Sender
	SenderMsg message.StatusBarMsg
	DirTree   DirectoryTree
	NotesList NotesList
	Editor    Editor

	Columns [4]string
}

const (
	ResponseYES = "y"
	ResponseNO  = "n"
)

var StatusBarColumn = struct {
	Mode, Message, Info int
}{
	Mode:    0,
	Message: 1,
	Info:    2,
}

func NewStatusBar() *StatusBar {
	ti := textinput.New()
	ti.Prompt = " "
	ti.CharLimit = 100

	statusBar := &StatusBar{
		Prompt: ti,
	}

	return statusBar
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(
	msg message.StatusBarMsg,
	teaMsg tea.Msg,
) *StatusBar {
	s.SetColContent(msg.Column, msg.Content)

	if s.Focused && s.Mode == mode.Normal {
		s.Type = msg.Type
	}

	s.Sender = msg.Sender

	switch teaMsg.(type) {
	case tea.KeyMsg:
		if s.Focused && s.Mode == mode.Insert && s.Prompt.Focused() {
			s.Prompt, _ = s.Prompt.Update(teaMsg)
			return s
		}
		//if s.Mode == mode.Normal {
		//	s.BlurPrompt()
		//}

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

	colGeneral := s.ColContent(sbc.General)
	colInfo := style.Render(s.ColContent(sbc.Info))
	colCursorPos := s.ColContent(sbc.CursorPos)
	colProgress := s.ColContent(sbc.Progress)

	//if s.Mode != mode.Normal &&
	//	s.Type == message.PromptError {
	if s.Mode != mode.Normal && s.Type != message.PromptError {
		colGeneral = s.ModeView()
	}

	if s.Type == message.PromptError {
		s.Prompt.Focus()
		colGeneral += s.Prompt.View()
	}

	width, _ := theme.GetTerminalSize()

	wColInfo := 15
	wColCursorPos := 15
	wColProgress := 15
	wColGeneral := max(width-(wColInfo+wColCursorPos+wColProgress), 1)

	return lipgloss.JoinHorizontal(lipgloss.Right,
		style.
			Width(wColGeneral).
			Foreground(s.Type.Colour()).
			Render(colGeneral),

		style.
			Width(wColInfo).
			Align(lipgloss.Right).
			Render(colInfo),

		style.
			Width(wColCursorPos).
			Align(lipgloss.Right).
			Render(colCursorPos),

		style.
			Width(wColProgress).
			Align(lipgloss.Right).
			Render(colProgress),
	)
}

func (s *StatusBar) ModeView() string {
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

func (s *StatusBar) ConfirmAction(
	sender message.Sender,
	c Focusable,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if c == nil {
		return statusMsg
	}

	if s.Prompt.Focused() {
		switch s.Prompt.Value() {
		case ResponseYES:
			if s.DirTree.EditState == EditStates.Delete ||
				s.NotesList.EditState == EditStates.Delete {
				statusMsg = c.Remove()
			}

		case ResponseNO:
			s.Columns[1] = ""
			statusMsg = c.CancelAction(func() {
				c.Refresh(false)
			})
		}
	}
	s.BlurPrompt()
	return statusMsg
}

func (s *StatusBar) CancelAction(cb func()) message.StatusBarMsg {
	s.Type = message.Error
	s.BlurPrompt()
	return message.StatusBarMsg{}
}

func (s *StatusBar) ColContent(col sbc.Column) string {
	return s.Columns[col]
}

func (s *StatusBar) SetColContent(col sbc.Column, cnt string) {
	s.Columns[col] = cnt
}

func (s *StatusBar) BlurPrompt() {
	s.Prompt.SetValue("")
	s.Prompt.Blur()
	s.Mode = mode.Normal
}

func style() lipgloss.Style {
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1)
}
