package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/charmbracelet/bubbles/textarea"
	bl "github.com/winder/bubblelayout"
)

type directoryTree struct {
	focused bool
	content *tree.Tree
}

type notesList struct {
	focused bool
	content string
}

type model struct {
	layout bl.BubbleLayout

	dirTreeID  bl.ID
	noteListID bl.ID
	editorID   bl.ID

	dirTreeSize   bl.Size
	notesListSize bl.Size
	editorSize    bl.Size

	textarea textarea.Model
	directoryTree
	notesList
}

func InitialModel() model {
	m := model{
		layout: bl.New(),
		directoryTree: directoryTree{
			focused: true,
			content: getDirectoryTree(),
		},
		notesList: notesList{
			focused: false,
			content: "",
		},
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

func baseColumnLayout(size bl.Size, focused bool) lipgloss.Style {
	borderColour := "#eee"
	if focused {
		borderColour = "#69c8dc"
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColour)).
		Foreground(lipgloss.Color("#eee")).
		Width(size.Width).
		Height(size.Height-2)
}

func (m model) View() string {
	t := textarea.New()
	t.Placeholder = "asdasd"
	t.Focus()
	return lipgloss.JoinHorizontal(0,
		baseColumnLayout(m.dirTreeSize, m.directoryTree.focused).
			Align(lipgloss.Left).
			Render(m.directoryTree.content.String()),
		baseColumnLayout(m.notesListSize, m.notesList.focused).
			Align(lipgloss.Center).
			Render(m.notesList.content),
		baseColumnLayout(m.editorSize, false).
			Render(t.View()),
	)
}
