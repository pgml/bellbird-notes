package components

import (
	"fmt"
	"strings"
	"time"

	"bellbird-notes/app/utils"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	bl "github.com/winder/bubblelayout"
)

type Focusable = interfaces.Focusable

type StatusBar struct {
	ID   bl.ID
	Size bl.Size

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

	ShouldQuit      bool
	ShouldWriteFile bool

	Height int
}

var StatusBarColumn = struct {
	Mode, Message, Info int
}{
	Mode:    0,
	Message: 1,
	Info:    2,
}

func NewStatusBar() *StatusBar {
	ti := textinput.New()
	ti.Prompt = ":"
	ti.CharLimit = 100
	ti.VirtualCursor = true

	statusBar := &StatusBar{
		Prompt: ti,
		Height: 1,
	}

	return statusBar
}

func (s StatusBar) Name() string { return "StatusBar" }

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(
	msgs []message.StatusBarMsg,
	teaMsg tea.Msg,
) *StatusBar {
	for i := range msgs {
		msg := msgs[i]

		if s.ColContent(msg.Column) == msg.Content && msg.Column != sbc.General {
			continue
		}

		// only update content if there is
		// no kind of input focused since we don't want to overwrite
		// the original message of the prompt
		if s.Type != message.PromptError {
			s.SetColContent(msg.Column, &msg.Content)
		}

		// we only really need the type for displaying messages in the
		// general column
		if s.Focused && s.Mode == mode.Normal && msg.Column == sbc.General {
			s.Type = msg.Type
		}

		s.Sender = msg.Sender
	}

	switch teaMsg.(type) {
	case tea.KeyMsg:
		if s.Focused && s.shouldShowMode() && s.Prompt.Focused() {
			s.Prompt, _ = s.Prompt.Update(teaMsg)
			return s
		}

	case tea.WindowSizeMsg:
		termWidth, _ := theme.TerminalSize()
		s.Size.Width = termWidth
		s.Size.Height = s.Height
	}

	return s
}

func (s *StatusBar) View() string {
	style := style()

	// get the content of each column
	colGeneral := s.ColContent(sbc.General)
	colFileInfo := s.ColContent(sbc.FileInfo)
	colKeyInfo := s.ColContent(sbc.KeyInfo)
	colProgress := s.ColContent(sbc.Progress)

	// display current mode only if there's is no prompt focused
	// and we are not in normal mode
	if s.shouldShowMode() && !s.isPrompt() {
		colGeneral = s.ModeView()
	}

	// append the prompt to the prompt message
	// and focus for allow quick input
	if s.isPrompt() {
		s.Prompt.Focus()
		promptView := strings.TrimSpace(s.Prompt.View())
		colGeneral = fmt.Sprint(colGeneral, promptView)
	}

	width, _ := theme.TerminalSize()

	wColFileInfo := 70
	wColKeyInfo := 15
	wColProgress := 15
	wColGeneral := max(width-(wColFileInfo+wColKeyInfo+wColProgress), 1)

	colFileInfo = utils.TruncateText(colFileInfo, wColFileInfo)

	return lipgloss.JoinHorizontal(lipgloss.Right,
		style.Width(wColGeneral).
			Foreground(s.Type.Colour()).
			Render(colGeneral),

		style.Width(wColFileInfo).
			Align(lipgloss.Right).
			Render(colFileInfo),

		style.Width(wColKeyInfo).
			Align(lipgloss.Center).
			Render(colKeyInfo),

		style.Width(wColProgress).
			Align(lipgloss.Right).
			PaddingRight(1).
			Render(colProgress),
	)
}

func (s *StatusBar) ModeView() string {
	style := lipgloss.NewStyle().
		Foreground(s.Mode.Colour()).
		PaddingLeft(1).
		PaddingRight(1)

	mode := ""

	if !s.Prompt.Focused() {
		mode = s.Mode.FullString(true)
	}

	return style.Render(mode)
}

func (s *StatusBar) ConfirmAction(
	sender message.Sender,
	c Focusable,
	e *Editor,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if !s.Prompt.Focused() {
		s.Focused = false
		return statusMsg
	}

	s.ShouldQuit = false
	s.ShouldWriteFile = false

	switch s.Prompt.Value() {
	case message.Response.Yes:
		if s.DirTree.EditState == EditStates.Delete ||
			s.NotesList.EditState == EditStates.Delete {

			statusMsg = c.Remove()
		}

	case message.Response.No:
		statusMsg = c.CancelAction(func() {
			c.Refresh(false, false)
		})
	case "q":
		s.ShouldQuit = true
	case "w":
		statusMsg = e.SaveBuffer()
	case "wq":
		statusMsg = e.SaveBuffer()
		s.ShouldQuit = true
	// this `set` stuff should only be here temporarily
	// this needs to be done better
	case "set number":
		e.SetNumbers()
	case "set nonumber":
		e.SetNoNumbers()
	case "config":
		statusMsg = e.OpenConfig()
	case "keymap":
		statusMsg = e.OpenUserKeyMap()
	case "defaultkeymap":
		statusMsg = e.NewScratchBuffer(
			"Default Keymap",
			string(s.Editor.KeyInput.DefaultKeyMap),
		)
		e.CurrentBuffer.Writeable = false
		e.Textarea.MoveCursor(0, 0, 0)
		e.SetContent()

	case "bd":
		e.DeleteCurrentBuffer()
	case "%bd": // temporary
		e.DeleteAllBuffers()
	case "buffers":
	case "b":
		e.ListBuffers = true

	case "new":
		statusMsg = e.NewScratchBuffer("Scratch", "")
		e.Textarea.SetValue("")
	}

	s.SetColContent(statusMsg.Column, &statusMsg.Content)
	s.BlurPrompt()

	statusMsg.Cmd = func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		s.Focused = false
		return DeferredActionMsg{}
	}

	return statusMsg
}

func (s *StatusBar) CancelAction(cb func()) message.StatusBarMsg {
	s.Type = message.Success
	s.BlurPrompt()
	return message.StatusBarMsg{}
}

func (s *StatusBar) ColContent(col sbc.Column) string {
	return s.Columns[col]
}

func (s *StatusBar) SetColContent(col sbc.Column, cnt *string) {
	s.Columns[col] = *cnt
}

func (s *StatusBar) FocusPrompt() {
	s.Prompt.Focus()
}

func (s *StatusBar) BlurPrompt() {
	s.Prompt.SetValue("")
	s.Prompt.Blur()
	s.Mode = mode.Normal
}

func (s *StatusBar) isPrompt() bool {
	return s.Type == message.Prompt || s.Type == message.PromptError
}

func (s *StatusBar) shouldShowMode() bool {
	return s.Mode == mode.Insert ||
		s.Mode == mode.Command ||
		s.Mode == mode.Replace ||
		s.Mode == mode.Visual ||
		s.Mode == mode.VisualLine ||
		s.Mode == mode.VisualBlock
}

func style() lipgloss.Style {
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1)
}
