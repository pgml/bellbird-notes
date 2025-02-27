package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model implements tea.Model, and manages the browser UI.
type Overlay struct {
	windowWidth  int
	windowHeight int
	title        string
	content      string
}

// Init initialises the Model on program load. It partly implements the tea.Model interface.
func (m *Overlay) Init() tea.Cmd {
	return nil
}

// Update handles event and manages internal state. It partly implements the tea.Model interface.
func (m *Overlay) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	}

	return m, cmd
}

// View applies and styling and handles rendering the view. It partly implements the tea.Model
// interface.
func (m *Overlay) View() string {
	foreStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)

	boldStyle := lipgloss.NewStyle().Bold(true)
	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		boldStyle.Render(m.title),
		m.content,
	)

	return foreStyle.Render(layout)
}

func NewOverlay() *Overlay {
	overlay := &Overlay{}
	return overlay
}
