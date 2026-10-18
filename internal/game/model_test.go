package game

import (
	"math/rand/v2"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newTestModel sizes the model for a 50x20 terminal, which gives a 24x16 board.
func newTestModel() model {
	return resize(newModel(rand.New(rand.NewPCG(1, 2))), 50, 20)
}

func resize(m model, width, height int) model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return next.(model)
}

func press(m model, key string) model {
	var msg tea.KeyMsg
	switch key {
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	return next.(model)
}

func step(m model) model {
	next, _ := m.Update(tickMsg{})
	return next.(model)
}

func TestBoardFillsTheTerminal(t *testing.T) {
	m := resize(newModel(rand.New(rand.NewPCG(1, 2))), 80, 24)

	if m.game.Width != 39 || m.game.Height != 20 {
		t.Errorf("board = %dx%d, want 39x20", m.game.Width, m.game.Height)
	}
}

func TestViewFitsTheTerminal(t *testing.T) {
	m := resize(newModel(rand.New(rand.NewPCG(1, 2))), 80, 24)

	lines := strings.Split(m.View(), "\n")
	if len(lines) > 24 {
		t.Errorf("view has %d lines, terminal has 24", len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("line %d is %d wide, terminal is 80", i, w)
		}
	}
}

func TestGrowingTheTerminalGrowsTheBoard(t *testing.T) {
	m := resize(newTestModel(), 80, 24)

	if m.game.Width != 39 || m.game.Height != 20 {
		t.Errorf("board = %dx%d, want 39x20", m.game.Width, m.game.Height)
	}
}

func TestTooSmallTerminalShowsMessageAndHaltsGame(t *testing.T) {
	m := resize(newTestModel(), 15, 8)
	head := m.game.Snake[0]

	m = step(m)

	if m.game.Snake[0] != head {
		t.Error("snake moved while the terminal was too small")
	}
	if !strings.Contains(m.View(), "22x10") {
		t.Errorf("expected the minimum terminal size in the message, got %q", m.View())
	}
}

func TestShrinkingBelowTheSnakeHaltsUntilItFitsAgain(t *testing.T) {
	m := newTestModel()
	// The head sits at {12 8}; a 10-column board would cut it off.
	m = resize(m, 22, 20)
	head := m.game.Snake[0]

	if step(m).game.Snake[0] != head {
		t.Fatal("snake moved although it no longer fits")
	}

	m = resize(m, 50, 20)
	if step(m).game.Snake[0] == head {
		t.Error("snake did not resume after the terminal grew back")
	}
}

func TestRestartUsesCurrentTerminalSize(t *testing.T) {
	m := resize(newTestModel(), 22, 20)
	m = press(m, "r")

	if m.game.Width != 10 {
		t.Errorf("board width = %d, want 10", m.game.Width)
	}
}

func TestArrowKeySteersSnake(t *testing.T) {
	m := press(newTestModel(), "up")
	m.game.Food = Point{X: 0, Y: 0}

	if got := step(m).game.Snake[0]; got != (Point{X: 12, Y: 7}) {
		t.Errorf("head = %v, want {12 7}", got)
	}
}

func TestPausedGameDoesNotMove(t *testing.T) {
	m := press(newTestModel(), "p")
	head := m.game.Snake[0]

	if step(m).game.Snake[0] != head {
		t.Error("snake moved while paused")
	}
}

func TestRestartOnlyAfterGameOver(t *testing.T) {
	m := newTestModel()
	m.game.Score = 3

	if press(m, "r").game.Score != 3 {
		t.Error("restart during a running game")
	}

	m.game.Over = true
	if press(m, "r").game.Score != 0 {
		t.Error("no restart after game over")
	}
}

func TestBestScoreSurvivesRestart(t *testing.T) {
	m := newTestModel()
	m.game.Food = Point{X: 13, Y: 8}

	m = step(m)
	m.game.Over = true
	m = press(m, "r")

	if m.best != 1 {
		t.Errorf("best = %d, want 1", m.best)
	}
}

func TestQuitKeyQuits(t *testing.T) {
	_, cmd := newTestModel().Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q did not quit")
	}
}
