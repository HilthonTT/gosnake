// Package single implements the classic one-snake game. Other modes (crazy)
// build on it by plugging in Hooks rather than duplicating the core rules.
package single

import (
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

var _ snake.GameController = (*Game)(nil)

// Hooks let a wrapping mode extend the classic rules without re-implementing
// them. Every hook is optional.
type Hooks struct {
	// BeforeTick runs at the start of every unpaused tick.
	BeforeTick func()

	// IsHazard reports whether moving onto p is fatal (in addition to walls
	// and the snake's own body).
	IsHazard func(p snake.Point) bool

	// Occupied returns extra cells that food must not spawn on.
	Occupied func() []snake.Point

	// Decorate draws extra cells after food but before the snake, so the
	// snake is always visible on top.
	Decorate func(m snake.Matrix)
}

type Game struct {
	matrix   snake.Matrix
	body     []snake.Point
	food     *snake.Point
	steering snake.Steering
	scoring  *snake.Scoring
	gameOver bool
	paused   bool
	hooks    Hooks
}

// NewGame starts a classic game at the given level (1-10).
func NewGame(level int) (*Game, error) {
	return NewGameWithHooks(level, Hooks{})
}

// NewGameWithHooks starts a classic game whose rules are extended by hooks.
func NewGameWithHooks(level int, hooks Hooks) (*Game, error) {
	scoring, err := snake.NewDefaultScoring(level)
	if err != nil {
		return nil, err
	}

	g := &Game{
		matrix:   snake.NewMatrix(snake.DefaultRows, snake.DefaultCols),
		body:     []snake.Point{{X: snake.DefaultCols / 2, Y: snake.DefaultRows / 2}},
		steering: snake.NewSteering(snake.Right),
		scoring:  scoring,
		hooks:    hooks,
	}
	g.food = snake.NewFood(g.occupied())
	g.render()

	return g, nil
}

// ChangeDirection queues a direction change, preventing 180-degree reversals.
func (g *Game) ChangeDirection(d snake.Direction) {
	if g.gameOver || g.paused {
		return
	}
	g.steering.Queue(d)
}

// TogglePause pauses or resumes the game.
func (g *Game) TogglePause() {
	if g.gameOver {
		return
	}
	g.paused = !g.paused
}

// Tick advances the game by one step.
func (g *Game) Tick() {
	if g.gameOver || g.paused {
		return
	}

	if g.hooks.BeforeTick != nil {
		g.hooks.BeforeTick()
	}

	next := g.body[0].Step(g.steering.Advance())
	ateFood := g.food != nil && next == *g.food

	if !g.matrix.InBounds(next) ||
		snake.HitsBody(g.body, next, !ateFood) ||
		(g.hooks.IsHazard != nil && g.hooks.IsHazard(next)) {
		g.gameOver = true
		g.render()
		return
	}

	g.body = snake.Advance(g.body, next, ateFood)
	if ateFood {
		g.scoring.AddPoints(snake.PointsPerFood)
		g.food = snake.NewFood(g.occupied())
	}

	g.render()
}

// occupied returns every cell food must avoid.
func (g *Game) occupied() []snake.Point {
	pts := append([]snake.Point(nil), g.body...)
	if g.hooks.Occupied != nil {
		pts = append(pts, g.hooks.Occupied()...)
	}
	return pts
}

// render writes the current game state onto the matrix.
func (g *Game) render() {
	g.matrix.Clear()

	if g.food != nil {
		g.matrix.Set(*g.food, snake.CellFood)
	}
	if g.hooks.Decorate != nil {
		g.hooks.Decorate(g.matrix)
	}
	g.matrix.DrawSnake(g.body, snake.CellHead, snake.CellBody)
}

// SetFood moves the food pellet to p. Intended for tests and scripted setups.
func (g *Game) SetFood(p snake.Point) {
	g.food = &p
	g.render()
}

// Refresh redraws the matrix. Wrapping modes call it after changing state
// that Decorate depends on.
func (g *Game) Refresh() { g.render() }

func (g *Game) Matrix() snake.Matrix           { return g.matrix }
func (g *Game) IsGameOver() bool               { return g.gameOver }
func (g *Game) IsPaused() bool                 { return g.paused }
func (g *Game) Score() int                     { return g.scoring.Total() }
func (g *Game) Level() int                     { return g.scoring.Level() }
func (g *Game) Snake() []snake.Point           { return g.body }
func (g *Game) SnakeLength() int               { return len(g.body) }
func (g *Game) Food() *snake.Point             { return g.food }
func (g *Game) Direction() snake.Direction     { return g.steering.Current() }
func (g *Game) GetTickInterval() time.Duration { return snake.GetTickInterval(g.scoring.Level()) }

// Snapshot returns a map of the current game state for crash reports.
func (g *Game) Snapshot() map[string]any {
	snap := map[string]any{
		"score":     g.scoring.Total(),
		"level":     g.scoring.Level(),
		"direction": g.steering.Current().String(),
		"snakeLen":  len(g.body),
		"food":      g.food,
		"paused":    g.paused,
		"gameOver":  g.gameOver,
	}
	if len(g.body) > 0 {
		snap["snakeHead"] = g.body[0]
	}
	return snap
}
