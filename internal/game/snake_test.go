package game

import (
	"math/rand/v2"
	"testing"
)

func newTestGame() *Game {
	return NewGame(10, 10, rand.New(rand.NewPCG(1, 2)))
}

func TestNewGameStartsWithSnakeInTheMiddleMovingRight(t *testing.T) {
	g := newTestGame()

	if got := g.Snake[0]; got != (Point{X: 5, Y: 5}) {
		t.Errorf("head = %v, want {5 5}", got)
	}
	if len(g.Snake) != 3 {
		t.Errorf("length = %d, want 3", len(g.Snake))
	}
}

func TestStepMovesHeadInCurrentDirection(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}

	g.Step()

	if got := g.Snake[0]; got != (Point{X: 6, Y: 5}) {
		t.Errorf("head = %v, want {6 5}", got)
	}
	if len(g.Snake) != 3 {
		t.Errorf("length = %d, want 3", len(g.Snake))
	}
}

func TestTurnChangesDirectionOnNextStep(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}

	g.Turn(Up)
	g.Step()

	if got := g.Snake[0]; got != (Point{X: 5, Y: 4}) {
		t.Errorf("head = %v, want {5 4}", got)
	}
}

func TestTurnIgnoresReversal(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}

	g.Turn(Left)
	g.Step()

	if got := g.Snake[0]; got != (Point{X: 6, Y: 5}) {
		t.Errorf("head = %v, want {6 5}", got)
	}
}

func TestTurnIgnoresReversalViaTwoQuickTurns(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}

	// Up then Left before the next step must not let the snake reverse into itself.
	g.Turn(Up)
	g.Turn(Left)
	g.Step()

	if g.Over {
		t.Fatal("game over after two quick turns")
	}
}

func TestEatingFoodGrowsSnakeAndScores(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 6, Y: 5}

	g.Step()

	if len(g.Snake) != 4 {
		t.Errorf("length = %d, want 4", len(g.Snake))
	}
	if g.Score != 1 {
		t.Errorf("score = %d, want 1", g.Score)
	}
}

func TestFoodIsNeverPlacedOnSnake(t *testing.T) {
	g := newTestGame()

	for range 100 {
		g.placeFood()
		for _, p := range g.Snake {
			if p == g.Food {
				t.Fatalf("food placed on snake at %v", p)
			}
		}
	}
}

func TestHittingWallEndsGame(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}

	for range 5 {
		g.Step()
	}

	if !g.Over {
		t.Error("expected game over after hitting the right wall")
	}
}

func TestHittingItselfEndsGame(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}
	g.Snake = []Point{{5, 5}, {4, 5}, {4, 4}, {5, 4}, {6, 4}, {6, 5}}

	g.Turn(Up)
	g.Step()

	if !g.Over {
		t.Error("expected game over after biting itself")
	}
}

func TestMovingIntoTailCellIsAllowed(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 0, Y: 0}
	// A 2x2 loop: the head moves into the cell the tail is leaving.
	g.Snake = []Point{{5, 5}, {5, 4}, {6, 4}, {6, 5}}
	g.dir, g.next = Down, Down

	g.Turn(Right)
	g.Step()

	if g.Over {
		t.Error("moving into the vacated tail cell should not end the game")
	}
}

func TestStepDoesNothingWhenGameIsOver(t *testing.T) {
	g := newTestGame()
	g.Over = true
	head := g.Snake[0]

	g.Step()

	if g.Snake[0] != head {
		t.Error("snake moved after game over")
	}
}

func TestResizeGrowsBoardAndKeepsSnake(t *testing.T) {
	g := newTestGame()
	snake := append([]Point(nil), g.Snake...)

	if !g.Resize(20, 15) {
		t.Fatal("growing the board should always fit")
	}

	if g.Width != 20 || g.Height != 15 {
		t.Errorf("size = %dx%d, want 20x15", g.Width, g.Height)
	}
	if g.Snake[0] != snake[0] {
		t.Errorf("head moved from %v to %v", snake[0], g.Snake[0])
	}
}

func TestResizeRefusesToCutOffSnake(t *testing.T) {
	g := newTestGame()

	if g.Resize(5, 5) {
		t.Fatal("expected resize to fail, the head at {5 5} would be outside")
	}
	if g.Width != 10 || g.Height != 10 {
		t.Errorf("size changed to %dx%d after a refused resize", g.Width, g.Height)
	}
}

func TestResizeMovesFoodBackOnTheBoard(t *testing.T) {
	g := newTestGame()
	g.Food = Point{X: 9, Y: 9}

	if !g.Resize(8, 8) {
		t.Fatal("snake fits in 8x8, resize should succeed")
	}
	if !g.inBounds(g.Food) {
		t.Errorf("food %v left outside the 8x8 board", g.Food)
	}
}
