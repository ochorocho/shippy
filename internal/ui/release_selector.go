package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	releaseCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	releaseSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	releaseMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	releaseHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// ReleaseItem represents a release for the selector UI
type ReleaseItem struct {
	Name      string
	Path      string
	DateTime  string
	GitCommit string
	GitTag    string
	IsCurrent bool
}

// releaseSelectorModel represents the state of the release selector
type releaseSelectorModel struct {
	releases []ReleaseItem
	cursor   int
	selected *ReleaseItem
	quitted  bool
}

func newReleaseSelectorModel(releases []ReleaseItem) releaseSelectorModel {
	// Start cursor at first non-current release
	startCursor := 0
	for i, r := range releases {
		if !r.IsCurrent {
			startCursor = i
			break
		}
	}
	return releaseSelectorModel{
		releases: releases,
		cursor:   startCursor,
	}
}

func (m releaseSelectorModel) Init() tea.Cmd {
	return nil
}

func (m releaseSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitted = true
			return m, tea.Quit

		case "up", "k":
			for i := m.cursor - 1; i >= 0; i-- {
				if !m.releases[i].IsCurrent {
					m.cursor = i
					break
				}
			}

		case "down", "j":
			for i := m.cursor + 1; i < len(m.releases); i++ {
				if !m.releases[i].IsCurrent {
					m.cursor = i
					break
				}
			}

		case "enter":
			if !m.releases[m.cursor].IsCurrent {
				r := m.releases[m.cursor]
				m.selected = &r
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m releaseSelectorModel) View() string {
	var s strings.Builder

	s.WriteString("\nSelect release to rollback to:\n\n")

	for i, release := range m.releases {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		line := formatReleaseLine(release)

		if release.IsCurrent {
			s.WriteString(fmt.Sprintf("  %s\n", releaseCurrentStyle.Render(line+" (current)")))
		} else if m.cursor == i {
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, releaseSelectedStyle.Render(line)))
		} else {
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, line))
		}
	}

	s.WriteString("\n")
	s.WriteString(releaseHelpStyle.Render("↑/↓: navigate • enter: select • q: quit"))
	s.WriteString("\n")

	return s.String()
}

func formatReleaseLine(r ReleaseItem) string {
	parts := []string{r.Name}

	if r.DateTime != "" {
		parts = append(parts, r.DateTime)
	}

	if r.GitCommit != "" {
		parts = append(parts, r.GitCommit)
	}

	if r.GitTag != "" {
		parts = append(parts, r.GitTag)
	}

	return strings.Join(parts, "  ")
}

// FormatReleaseDateTime parses a release timestamp directory name into a human-readable date/time
func FormatReleaseDateTime(name string) string {
	if len(name) != 14 {
		return ""
	}
	t, err := time.Parse("20060102150405", name)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
