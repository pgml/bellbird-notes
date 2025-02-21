package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	bl "github.com/winder/bubblelayout"
)

type layoutModel struct {
	layout bl.BubbleLayout

	dirTreeID  bl.ID
	noteListID bl.ID
	editorID   bl.ID

	dirTreeSize   bl.Size
	notesListSize bl.Size
	editorSize    bl.Size
}

func New() tea.Model {
	layoutModel := layoutModel{layout: bl.New()}
	layoutModel.dirTreeID = layoutModel.layout.Add("width 30")
	layoutModel.noteListID = layoutModel.layout.Add("width 30")
	layoutModel.editorID = layoutModel.layout.Add("grow")

	return layoutModel
}

func (m layoutModel) Init() tea.Cmd {
	return func() tea.Msg {
		return m.layout.Resize(80, 40)
	}
}

func (m layoutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
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

	return m, nil
}

func boxStyle(size bl.Size) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("#333")).
		Width(size.Width).
		Height(size.Height).
		Align(lipgloss.Center, lipgloss.Center)
}

func (m layoutModel) View() string {
	return lipgloss.JoinHorizontal(0,
		boxStyle(m.dirTreeSize).Render("Directory Tree"),
		boxStyle(m.notesListSize).Render("Notes"),
		boxStyle(m.editorSize).Render("Editor"),
	)
}

func main() {
	p := tea.NewProgram(New())
	p.Run()
}
