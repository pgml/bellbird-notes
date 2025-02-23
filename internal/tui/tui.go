package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/charmbracelet/bubbles/textarea"
	bl "github.com/winder/bubblelayout"
)

type model struct {
	layout bl.BubbleLayout

	dirTreeID  bl.ID
	noteListID bl.ID
	editorID   bl.ID

	dirTreeSize   bl.Size
	notesListSize bl.Size
	editorSize    bl.Size

	dirTree *tree.Tree
	textarea textarea.Model
}

func InitialModel() model {
	m := model{
		layout: bl.New(),
		dirTree: directoryTree(),
	}

	m.dirTreeID = m.layout.Add("width 30")
	m.noteListID = m.layout.Add("width 30")
	m.editorID = m.layout.Add("grow")

	return m
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		return m.layout.Resize(80, 40)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	tea.EnterAltScreen()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Convert WindowSizeMsg to BubbleLayoutMsg.
		return m, func() tea.Msg {
			return m.layout.Resize(msg.Width, msg.Height)
		}
	case bl.BubbleLayoutMsg:
		m.dirTreeSize, _ = msg.Size(m.dirTreeID)
		m.notesListSize, _ = msg.Size(m.noteListID)
		m.editorSize, _ = msg.Size(m.editorID)
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	return m, cmd
}

func baseColumnLayout(size bl.Size) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("#eee")).
		Width(size.Width).
		Height(size.Height-2)
}

func (m model) View() string {
	t := textarea.New()
	t.Placeholder = "asdasd"
	t.Focus()
	return lipgloss.JoinHorizontal(0,
		baseColumnLayout(m.dirTreeSize).Align(lipgloss.Left).Render(m.dirTree.String()),
		baseColumnLayout(m.notesListSize).Align(lipgloss.Center).Render("Notes"),
		baseColumnLayout(m.editorSize).Render(t.View()),
	)
}
