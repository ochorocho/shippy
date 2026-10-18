// Package game contains shippy's hidden easter egg: a game of snake.
package game

import (
	"math/rand/v2"
	"slices"
)

// Point is a cell on the board.
type Point struct {
	X, Y int
}

// Direction is the way the snake is heading.
type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

var deltas = map[Direction]Point{
	Up:    {0, -1},
	Down:  {0, 1},
	Left:  {-1, 0},
	Right: {1, 0},
}

var opposites = map[Direction]Direction{
	Up:    Down,
	Down:  Up,
	Left:  Right,
	Right: Left,
}

// Game holds the state of one round of snake. It has no knowledge of
// terminals or timing, so it can be driven step by step in tests.
type Game struct {
	Width, Height int
	Snake         []Point // head first
	Food          Point
	Score         int
	Over          bool
	Won           bool

	dir  Direction // direction of the last step
	next Direction // direction of the coming step
	rng  *rand.Rand
}

// NewGame starts a round with a three-cell snake in the middle of the board,
// heading right.
func NewGame(width, height int, rng *rand.Rand) *Game {
	head := Point{X: width / 2, Y: height / 2}
	g := &Game{
		Width:  width,
		Height: height,
		Snake:  []Point{head, {head.X - 1, head.Y}, {head.X - 2, head.Y}},
		dir:    Right,
		next:   Right,
		rng:    rng,
	}
	g.placeFood()
	return g
}

// Turn sets the direction for the next step. Reversing straight back into
// the snake's own body is ignored.
func (g *Game) Turn(d Direction) {
	if d == opposites[g.dir] {
		return
	}
	g.next = d
}

// Step moves the snake one cell and resolves eating and collisions.
func (g *Game) Step() {
	if g.Over {
		return
	}

	g.dir = g.next
	delta := deltas[g.dir]
	head := Point{X: g.Snake[0].X + delta.X, Y: g.Snake[0].Y + delta.Y}
	eating := head == g.Food

	if !g.inBounds(head) || g.hitsBody(head, eating) {
		g.Over = true
		return
	}

	g.Snake = append([]Point{head}, g.Snake...)
	if !eating {
		g.Snake = g.Snake[:len(g.Snake)-1]
		return
	}

	g.Score++
	g.placeFood()
}

func (g *Game) inBounds(p Point) bool {
	return p.X >= 0 && p.X < g.Width && p.Y >= 0 && p.Y < g.Height
}

// hitsBody reports whether p collides with the snake. The tail cell is free
// unless the snake is growing, because the tail moves away on this step.
func (g *Game) hitsBody(p Point, growing bool) bool {
	body := g.Snake
	if !growing {
		body = body[:len(body)-1]
	}
	return slices.Contains(body, p)
}

// placeFood puts the food on a random free cell. A board without free cells
// means the snake fills it: the player has won.
func (g *Game) placeFood() {
	occupied := make(map[Point]bool, len(g.Snake))
	for _, s := range g.Snake {
		occupied[s] = true
	}

	var free []Point
	for y := range g.Height {
		for x := range g.Width {
			if p := (Point{x, y}); !occupied[p] {
				free = append(free, p)
			}
		}
	}

	if len(free) == 0 {
		g.Over, g.Won = true, true
		return
	}
	g.Food = free[g.rng.IntN(len(free))]
}

// Resize changes the board size and reports whether the snake fits. A board
// that would cut off part of the snake is refused and the game is unchanged.
func (g *Game) Resize(width, height int) bool {
	bounds := Game{Width: width, Height: height}
	for _, s := range g.Snake {
		if !bounds.inBounds(s) {
			return false
		}
	}

	g.Width, g.Height = width, height
	if !g.inBounds(g.Food) {
		g.placeFood()
	}
	return true
}
