package components

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Editor struct {
	Component

	Buffers       []Buffer
	CurrentBuffer Buffer
	Textarea      textarea.Model
	Mode          app.ModeInstance

	err error
}
type errMsg error

type Buffer struct {
	Index       int
	CurrentLine int
	CursorPos   int
	Path        string
	Content     string
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (e *Editor) Init() tea.Cmd {
	return textarea.Blink
}

func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	termWidth, termHeight := theme.GetTerminalSize()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if e.Focused && e.Mode.Current == app.InsertMode {
			cmd = e.Textarea.Focus()
		} else {
			e.Textarea.Blur()
		}
		if !e.Textarea.Focused() {
			//_, cmd := e.Textarea.Update(msg)
			//cmds = append(cmds, cmd)
		}
	case tea.WindowSizeMsg:
		if !e.ready {
			e.viewport = viewport.New(termWidth, termHeight-1)
			e.viewport.SetContent(e.build())
			e.viewport.KeyMap = viewport.KeyMap{}
			e.ready = true
		} else {
			e.viewport.Width = termWidth
			e.viewport.Height = termHeight - 1
		}
	case errMsg:
		e.err = msg
		return e, nil
	}

	e.Textarea, cmd = e.Textarea.Update(msg)
	cmds = append(cmds, cmd)
	// Handle keyboard and mouse events in the viewport
	//_, cmd = e.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return e, tea.Batch(cmds...)
}

func (e *Editor) View() string {
	if !e.ready {
		return "\n  Initializing..."
	}

	e.viewport.SetContent(e.build())
	e.viewport.Style = theme.BaseColumnLayout(e.Size, e.Focused)

	if e.Focused {
		e.Textarea.Focus()
	} else {
		e.Textarea.Blur()
	}

	return e.viewport.View()
}

func NewEditor() *Editor {
	termWidth, termHeight := theme.GetTerminalSize()

	textarea := textarea.New()
	textarea.ShowLineNumbers = false
	textarea.Prompt = ""
	textarea.SetWidth(termWidth)
	textarea.SetHeight(termHeight)
	textarea.MaxWidth = termWidth
	textarea.FocusedStyle.CursorLine = lipgloss.NewStyle()

	//conf := config.New()
	editor := &Editor{
		Mode:     app.ModeInstance{Current: app.NormalMode},
		Textarea: textarea,
	}

	//editor.Refresh(false)
	return editor
}

func (e Editor) build() string {
	return e.Textarea.View()
}

func (e *Editor) EnterInsertMode() messages.StatusBarMsg {
	if e.Focused {
		e.Mode.Current = app.InsertMode
	}
	return messages.StatusBarMsg{}
}

func (e *Editor) ExitInsertMode() messages.StatusBarMsg {
	if e.Focused {
		e.Mode.Current = app.NormalMode
	}
	return messages.StatusBarMsg{}
}
