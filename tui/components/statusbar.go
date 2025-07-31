package components

import (
	"fmt"
	"regexp"
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

type Commands map[string]func(opts ...string) message.StatusBarMsg

// StatusBar represents the bottom bar UI component that displays messages,
// input prompts, and application mode information.
type StatusBar struct {
	ID   bl.ID
	Size bl.Size

	Content string
	Type    message.Type
	Prompt  textinput.Model

	// Indicates whether the directory tree column is focused.
	// Used to determine if the status bar should receive keyboard shortcuts
	Focused bool

	// The Current editing/view mode
	Mode mode.Mode

	//Sender message.Sender
	//SenderMsg message.StatusBarMsg

	Editor Editor

	// The content for each column
	Columns [4]string

	// The height of the status bar
	Height int

	// Registered prompt commands
	Commands Commands

	TeaCmd tea.Cmd
}

var StatusBarColumn = struct {
	Mode, Message, Info int
}{
	Mode:    0,
	Message: 1,
	Info:    2,
}

// NewStatusBar creates and returns a new StatusBar with default settings.
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

// Name returns the unique name of the StatusBar
func (s StatusBar) Name() string { return "StatusBar" }

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

// Update processes incoming StatusBarMsgs and tea.Msgs,
// updating the internal state of the StatusBar as needed.
func (s *StatusBar) Update(
	msgs []message.StatusBarMsg,
	teaMsg tea.Msg,
) (*StatusBar, tea.Cmd) {
	var cmd tea.Cmd

	for i := range msgs {
		msg := msgs[i]

		// Avoid unnecessary update unless the content has changed
		if s.colContent(msg.Column) == msg.Content && msg.Column != sbc.General {
			continue
		}

		// only update content if there is
		// no kind of input focused since we don't want to overwrite
		// the original message of the prompt
		if s.Type != message.PromptError {
			s.setColContent(msg.Column, &msg.Content)
		}

		// we only really need the type for displaying messages in the
		// general column
		if s.Focused && s.Mode == mode.Normal && msg.Column == sbc.General {
			s.Type = msg.Type
		}

		//s.Sender = msg.Sender
	}

	switch s.Mode {
	case mode.SearchPrompt:
		s.TeaCmd = func() tea.Msg {
			return SearchMsg{
				SearchTerm: s.Prompt.Value(),
			}
		}

	default:
		s.TeaCmd = nil
	}

	switch msg := teaMsg.(type) {
	case tea.KeyMsg:
		if s.Focused && s.shouldShowMode() && s.Prompt.Focused() {
			s.Prompt, _ = s.Prompt.Update(teaMsg)
			return s, cmd
		}

	case tea.WindowSizeMsg:
		termWidth, _ := theme.TerminalSize()
		s.Size.Width = termWidth
		s.Size.Height = s.Height

	case SearchConfirmedMsg:
		s.Mode = mode.Search
		s.Focused = false
		s.BlurPrompt(msg.ResetPrompt)

	case SearchCancelMsg:
		s.Focused = false
		s.BlurPrompt(true)
		s.CancelAction(func() {})
	}

	return s, cmd
}

// View renders the StatusBar as a string
func (s *StatusBar) View() string {
	style := style()

	// Get the content of each column
	colGeneral := s.colContent(sbc.General)
	colFileInfo := s.colContent(sbc.FileInfo)
	colKeyInfo := s.colContent(sbc.KeyInfo)
	colProgress := s.colContent(sbc.Progress)

	// Display current mode only if there's is no prompt focused
	// and we are not in normal mode
	if s.shouldShowMode() && !s.isPrompt() {
		colGeneral = s.ModeView()
	}

	// Append the prompt to the prompt message
	// and focus for allow quick input
	if s.isPrompt() {
		switch s.Mode {
		case mode.Command:
			s.Prompt.Prompt = ":"
		case mode.SearchPrompt, mode.Search:
			s.Prompt.Prompt = "/"
		}
		s.Prompt.Focus()
		promptView := strings.TrimSpace(s.Prompt.View())
		colGeneral = fmt.Sprint(colGeneral, promptView)
	}

	width, _ := theme.TerminalSize()

	// Set the widths of each column
	wColFileInfo := 70
	wColKeyInfo := 15
	wColProgress := 15
	wColGeneral := max(width-(wColFileInfo+wColKeyInfo+wColProgress), 1)

	colFileInfo = utils.TruncateText(colFileInfo, wColFileInfo)

	promptColour := s.Type.Colour()

	return lipgloss.JoinHorizontal(lipgloss.Right,
		style.Width(wColGeneral).Foreground(promptColour).Render(colGeneral),
		style.Width(wColFileInfo).Align(lipgloss.Right).Render(colFileInfo),
		style.Width(wColKeyInfo).Align(lipgloss.Center).Render(colKeyInfo),
		style.Width(wColProgress).Align(lipgloss.Right).PaddingRight(1).Render(colProgress),
	)
}

// ModeView returns the rendered mode string
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

// ConfirmAction finalises the current prompt input, executes the matched command,
// and returns the result as a StatusBarMsg.
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

	fnMsg := s.execPromptFn()

	s.setColContent(statusMsg.Column, &statusMsg.Content)
	s.BlurPrompt(true)

	statusMsg.Cmd = tea.Batch(
		s.SendDeferredActionMsg(),
		fnMsg.Cmd,
	)

	statusMsg.Content = fnMsg.Content

	return statusMsg
}

// SendDeferredActionMsg returns a delayed tea.Msg for post-command behaviour
func (s *StatusBar) SendDeferredActionMsg() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		s.Focused = false
		return DeferredActionMsg{}
	}
}

// execPromptFn parses the current prompt and executes the associated command function.
func (s *StatusBar) execPromptFn() message.StatusBarMsg {
	var fnMsg message.StatusBarMsg

	promptCmd := s.Prompt.Value()
	args := ""

	re := regexp.MustCompile(`^(open|set|reload)\s+(\S+)\s*(.*)`)
	matches := re.FindStringSubmatch(promptCmd)

	if len(matches) > 0 {
		promptCmd = matches[1]
		args = matches[2]
	}

	for cmd, fn := range s.Commands {
		if cmd == promptCmd {
			fnMsg = fn(args)
			break
		}
	}

	return fnMsg
}

// CancelAction cancels the current prompt input and resets the status bar state.
func (s *StatusBar) CancelAction(cb func()) message.StatusBarMsg {
	s.Type = message.Success
	s.BlurPrompt(true)
	return message.StatusBarMsg{}
}

// colContent returns the content string of the specified column.
func (s *StatusBar) colContent(col sbc.Column) string {
	return s.Columns[col]
}

// setColContent updates the content of the specified status bar col
func (s *StatusBar) setColContent(col sbc.Column, cnt *string) {
	s.Columns[col] = *cnt
}

func (s *StatusBar) BlurPrompt(resetValue bool) {
	if resetValue {
		s.Prompt.SetValue("")
	}

	s.Prompt.Blur()
	s.Mode = mode.Normal
}

func (s *StatusBar) isPrompt() bool {
	return s.Type == message.Prompt ||
		s.Type == message.PromptError
}

func (s *StatusBar) shouldShowMode() bool {
	return s.Mode == mode.Insert ||
		s.Mode == mode.Command ||
		s.Mode == mode.Search ||
		s.Mode == mode.SearchPrompt ||
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
