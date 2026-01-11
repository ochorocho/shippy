package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// selectorModel represents the state of the host selector
type selectorModel struct {
	hosts    []string
	cursor   int
	selected string
	quitted  bool
}

// newSelectorModel creates a new selector model
func newSelectorModel(hosts []string) selectorModel {
	return selectorModel{
		hosts:  hosts,
		cursor: 0,
	}
}

// Init initializes the model
func (m selectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitted = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.hosts)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = m.hosts[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the UI
func (m selectorModel) View() string {
	var s strings.Builder

	s.WriteString("\nSelect deployment host:\n\n")

	for i, host := range m.hosts {
		cursor := " "
		if m.cursor == i {
			cursor = cursorStyle.Render(">")
		}

		hostDisplay := host
		if m.cursor == i {
			hostDisplay = selectedStyle.Render(host)
		}

		s.WriteString(fmt.Sprintf("  %s %s\n", cursor, hostDisplay))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q: quit"))
	s.WriteString("\n")

	return s.String()
}
