package game

import (
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tickRate = 110 * time.Millisecond

	// Smallest board worth playing on, in cells.
	minBoardWidth  = 10
	minBoardHeight = 6

	// Terminal space around the board: the border left and right, and the
	// title, border top and bottom, and help line.
	chromeCols = 2
	chromeRows = 4
)

var (
	headCell  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700")).Render("██")
	bodyCell  = lipgloss.NewStyle().Foreground(lipgloss.Color("#D75F00")).Render("██")
	foodCell  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD7FF")).Render("◆ ")
	emptyCell = "  "

	boardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241"))
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700")).Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type tickMsg struct{}

// model drives a Game from key presses and a fixed tick, and sizes the board
// to the terminal.
type model struct {
	game   *Game // nil until the terminal size is known
	best   int
	paused bool
	fits   bool // false while the terminal is too small for the board
	rng    *rand.Rand

	width, height int // terminal size
}

func newModel(rng *rand.Rand) model {
	return model{rng: rng}
}

// boardSize returns the largest board that fits the terminal. Cells are two
// columns wide so the board looks square.
func (m model) boardSize() (int, int) {
	return (m.width - chromeCols) / 2, m.height - chromeRows
}

func (m model) playable() bool {
	w, h := m.boardSize()
	return w >= minBoardWidth && h >= minBoardHeight
}

func (m model) newRound() model {
	w, h := m.boardSize()
	m.game = NewGame(w, h, m.rng)
	m.fits = true
	m.paused = false
	return m
}

func tick() tea.Cmd {
	return tea.Tick(tickRate, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		switch {
		case !m.playable():
			m.fits = false
		case m.game == nil:
			m = m.newRound()
		default:
			m.fits = m.game.Resize(m.boardSize())
		}
		return m, nil

	case tickMsg:
		if m.game != nil && m.fits && !m.paused {
			m.game.Step()
			m.best = max(m.best, m.game.Score)
		}
		return m, tick()

	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" || key == "esc" {
			return m, tea.Quit
		}
		if m.game == nil {
			return m, nil
		}

		switch key {
		case "up", "k", "w":
			m.game.Turn(Up)
		case "down", "j", "s":
			m.game.Turn(Down)
		case "left", "h", "a":
			m.game.Turn(Left)
		case "right", "l", "d":
			m.game.Turn(Right)
		case "p", " ":
			if !m.game.Over && m.fits {
				m.paused = !m.paused
			}
		case "r":
			if (m.game.Over || !m.fits) && m.playable() {
				m = m.newRound()
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.game == nil || !m.playable() {
		minW := minBoardWidth*2 + chromeCols
		minH := minBoardHeight + chromeRows
		return m.message(fmt.Sprintf("Terminal too small for snake. Make it at least %dx%d. q: quit", minW, minH))
	}

	g := m.game
	title := titleStyle.Render("Shippy Snake") + fmt.Sprintf("   Score: %d   Best: %d", g.Score, m.best)

	if !m.fits {
		return m.message(title + "\n\nThe window got too small for your snake. Enlarge it to continue. r: new round • q: quit")
	}

	help := "←↑↓→ / hjkl / wasd: steer • p: pause • q: quit"
	switch {
	case g.Won:
		help = "You filled the board! r: play again • q: quit"
	case g.Over:
		help = "Game over! r: play again • q: quit"
	case m.paused:
		help = "Paused • p: resume • q: quit"
	}

	return m.center(title + "\n" + boardStyle.Render(m.renderBoard()) + "\n" + helpStyle.Render(help))
}

func (m model) renderBoard() string {
	g := m.game

	cells := make(map[Point]string, len(g.Snake)+1)
	cells[g.Food] = foodCell
	for i, p := range g.Snake {
		cells[p] = bodyCell
		if i == 0 {
			cells[p] = headCell
		}
	}

	var board strings.Builder
	for y := range g.Height {
		for x := range g.Width {
			cell, ok := cells[Point{x, y}]
			if !ok {
				cell = emptyCell
			}
			board.WriteString(cell)
		}
		if y < g.Height-1 {
			board.WriteString("\n")
		}
	}
	return board.String()
}

// message centers text in the terminal, wrapping it to the terminal width.
func (m model) message(text string) string {
	text = lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(text)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, text)
}

// center places content in the middle of the terminal, cutting lines that
// are wider than the terminal so nothing wraps.
func (m model) center(content string) string {
	content = lipgloss.NewStyle().MaxWidth(m.width).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// Play runs snake full-screen until the player quits and returns the best
// score of the session.
func Play(in io.Reader, out io.Writer) (int, error) {
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) // #nosec G404 -- game randomness, not security

	p := tea.NewProgram(newModel(rng), tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to run snake: %w", err)
	}

	m, ok := final.(model)
	if !ok {
		return 0, fmt.Errorf("unexpected model type %T", final)
	}
	return m.best, nil
}
