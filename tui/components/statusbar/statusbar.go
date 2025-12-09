package statusbar

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"bellbird-notes/app/state"
	"bellbird-notes/app/utils"
	"bellbird-notes/internal/interfaces"
	"bellbird-notes/tui/components/editor"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	"bellbird-notes/tui/shared"
	"bellbird-notes/tui/theme"
	sbc "bellbird-notes/tui/types/statusbar_column"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	bl "github.com/winder/bubblelayout"
)

type StateController interface {
	Append(entry state.StateEntry)
}

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

	State *state.State

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
func New() *StatusBar {
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
func (sb StatusBar) Name() string { return "StatusBar" }

// Init initialises the Model on program load.
// It partly implements the tea.Model interface.
func (sb *StatusBar) Init() tea.Cmd {
	return nil
}

// Update processes incoming StatusBarMsgs and tea.Msgs,
// updating the internal state of the StatusBar as needed.
func (sb *StatusBar) Update(
	msgs []message.StatusBarMsg,
	teaMsg tea.Msg,
) (*StatusBar, tea.Cmd) {
	var cmd tea.Cmd

	for i := range msgs {
		msg := msgs[i]

		// Avoid unnecessary update unless the content has changed
		if sb.colContent(msg.Column) == msg.Content && msg.Column != sbc.General {
			continue
		}

		// only update content if there is
		// no kind of input focused since we don't want to overwrite
		// the original message of the prompt
		if sb.Type != message.PromptError {
			sb.setColContent(msg.Column, &msg.Content)
		}

		// we only really need the type for displaying messages in the
		// general column
		if sb.Focused && sb.Mode == mode.Normal && msg.Column == sbc.General {
			sb.Type = msg.Type
		}

		//s.Sender = msg.Sender
	}

	switch sb.Mode {
	case mode.SearchPrompt:
		sb.TeaCmd = func() tea.Msg {
			return editor.SearchMsg{
				SearchTerm: sb.Prompt.Value(),
			}
		}

	default:
		sb.TeaCmd = nil
	}

	switch msg := teaMsg.(type) {
	case tea.KeyMsg:
		if sb.Focused && sb.shouldShowMode() && sb.Prompt.Focused() {
			sb.Prompt, _ = sb.Prompt.Update(teaMsg)
			return sb, cmd
		}

	case tea.WindowSizeMsg:
		termWidth, _ := theme.TerminalSize()
		sb.Size.Width = termWidth
		sb.Size.Height = sb.Height

	case editor.SearchConfirmedMsg:
		sb.Mode = mode.Search
		sb.Focused = false
		sb.State.Append(state.NewEntry(state.Search, sb.Prompt.Value()))
		sb.BlurPrompt(msg.ResetPrompt)

	case editor.SearchCancelMsg:
		sb.Focused = false
		sb.State.Append(state.NewEntry(state.Search, sb.Prompt.Value()))
		sb.BlurPrompt(true)
		sb.CancelAction(func() {})
	}

	return sb, cmd
}

// View renders the StatusBar as a string
func (sb *StatusBar) View() string {
	style := style()

	// Get the content of each column
	colGeneral := sb.colContent(sbc.General)
	colFileInfo := sb.colContent(sbc.FileInfo)
	colKeyInfo := sb.colContent(sbc.KeyInfo)
	colProgress := sb.colContent(sbc.Progress)

	// Display current mode only if there's is no prompt focused
	// and we are not in normal mode
	if sb.shouldShowMode() && !sb.isPrompt() {
		colGeneral = sb.ModeView()
	}

	// Append the prompt to the prompt message
	// and focus for allow quick input
	if sb.isPrompt() {
		switch sb.Mode {
		case mode.Command:
			sb.Prompt.Prompt = ":"
		case mode.SearchPrompt, mode.Search:
			sb.Prompt.Prompt = "/"
		}
		sb.Prompt.Focus()
		promptView := strings.TrimSpace(sb.Prompt.View())
		colGeneral = fmt.Sprint(colGeneral, promptView)
	}

	width, _ := theme.TerminalSize()

	// Set the widths of each column
	wColFileInfo := 70
	wColKeyInfo := 15
	wColProgress := 15
	wColGeneral := max(width-(wColFileInfo+wColKeyInfo+wColProgress), 1)

	colFileInfo = utils.TruncateText(colFileInfo, wColFileInfo)

	promptColour := sb.Type.Colour()

	return lipgloss.JoinHorizontal(lipgloss.Right,
		style.Width(wColGeneral).Foreground(promptColour).Render(colGeneral),
		style.Width(wColFileInfo).Align(lipgloss.Right).Render(colFileInfo),
		style.Width(wColKeyInfo).Align(lipgloss.Center).Render(colKeyInfo),
		style.Width(wColProgress).Align(lipgloss.Right).PaddingRight(1).Render(colProgress),
	)
}

// ModeView returns the rendered mode string
func (sb *StatusBar) ModeView() string {
	style := lipgloss.NewStyle().
		Foreground(sb.Mode.Colour()).
		PaddingLeft(1).
		PaddingRight(1)

	mode := ""

	if !sb.Prompt.Focused() {
		mode = sb.Mode.FullString(true)
	}

	return style.Render(mode)
}

// ConfirmAction finalises the current prompt input, executes the matched command,
// and returns the result as a StatusBarMsg.
func (sb *StatusBar) ConfirmAction(
	sender message.Sender,
	c Focusable,
) message.StatusBarMsg {
	statusMsg := message.StatusBarMsg{}

	if !sb.Prompt.Focused() {
		sb.Focused = false
		return statusMsg
	}

	fnMsg := sb.execPromptFn()

	stateType := state.Command

	if sb.Mode == mode.SearchPrompt {
		stateType = state.Search
	}

	sb.State.Append(state.NewEntry(stateType, sb.Prompt.Value()))
	sb.setColContent(statusMsg.Column, &statusMsg.Content)
	sb.BlurPrompt(true)

	statusMsg.Cmd = tea.Batch(
		sb.SendDeferredActionMsg(),
		fnMsg.Cmd,
	)

	statusMsg.Content = fnMsg.Content

	return statusMsg
}

// SendDeferredActionMsg returns a delayed tea.Msg for post-command behaviour
func (sb *StatusBar) SendDeferredActionMsg() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		sb.Focused = false
		return shared.DeferredActionMsg{}
	}
}

// execPromptFn parses the current prompt and executes the associated command function.
func (sb *StatusBar) execPromptFn() message.StatusBarMsg {
	var fnMsg message.StatusBarMsg

	promptCmd := sb.Prompt.Value()
	args := ""

	re := regexp.MustCompile(`^(open|set|reload)\s+(\S+)\s*(.*)`)
	matches := re.FindStringSubmatch(promptCmd)

	if len(matches) > 0 {
		promptCmd = matches[1]
		args = matches[2]
	}

	for cmd, fn := range sb.Commands {
		if cmd == promptCmd {
			fnMsg = fn(args)
			break
		}
	}

	return fnMsg
}

// CancelAction cancels the current prompt input and resets the status bar state.
func (sb *StatusBar) CancelAction(cb func()) message.StatusBarMsg {
	sb.Type = message.Success
	sb.BlurPrompt(true)
	return message.StatusBarMsg{}
}

func (sb *StatusBar) PromptHistoryBack() message.StatusBarMsg {
	entry := sb.State.CycleCommands(false)
	sb.Prompt.SetValue(entry.Content())
	sb.Prompt.CursorEnd()
	return message.StatusBarMsg{}
}

func (sb *StatusBar) PromptHistoryForward() message.StatusBarMsg {
	entry := sb.State.CycleCommands(true)
	sb.Prompt.SetValue(entry.Content())
	sb.Prompt.CursorEnd()
	return message.StatusBarMsg{}
}

func (sb *StatusBar) SearchHistoryBack() message.StatusBarMsg {
	entry := sb.State.CycleSearchResults(false)
	sb.Prompt.SetValue(entry.Content())
	sb.Prompt.CursorEnd()
	return message.StatusBarMsg{}
}

func (sb *StatusBar) SearchHistoryForward() message.StatusBarMsg {
	entry := sb.State.CycleSearchResults(true)
	sb.Prompt.SetValue(entry.Content())
	sb.Prompt.CursorEnd()
	return message.StatusBarMsg{}
}

// colContent returns the content string of the specified column.
func (sb *StatusBar) colContent(col sbc.Column) string {
	return sb.Columns[col]
}

// setColContent updates the content of the specified status bar col
func (sb *StatusBar) setColContent(col sbc.Column, cnt *string) {
	sb.Columns[col] = *cnt
}

func (sb *StatusBar) BlurPrompt(resetValue bool) {
	if resetValue {
		sb.Prompt.SetValue("")
	}

	sb.Prompt.Blur()
	sb.Mode = mode.Normal
}

func (sb *StatusBar) isPrompt() bool {
	return sb.Type == message.Prompt ||
		sb.Type == message.PromptError
}

func (sb *StatusBar) shouldShowMode() bool {
	return sb.Mode == mode.Insert ||
		sb.Mode == mode.Command ||
		sb.Mode == mode.Search ||
		sb.Mode == mode.SearchPrompt ||
		sb.Mode == mode.Replace ||
		sb.Mode == mode.Visual ||
		sb.Mode == mode.VisualLine ||
		sb.Mode == mode.VisualBlock
}

func style() lipgloss.Style {
	return lipgloss.NewStyle().
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Height(1)
}
