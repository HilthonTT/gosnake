package crazy

import (
	"slices"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

// Ensure *Game satisfies the shared controller interface at compile time.
var _ snake.GameController = (*Game)(nil)

// Game is the crazy-mode variant of the snake game. It behaves identically to
// the normal mode except that timed bombs are scattered around the board.
// Bombs cycle through a warning (blinking) phase followed by an active (lethal)
// phase, and their count scales with the current level.
type Game struct {
	matrix    snake.Matrix
	snakeBody []snake.Point
	food      *snake.Point
	direction snake.Direction
	nextDir   snake.Direction
	scoring   *snake.Scoring
	bombs     []*Bomb
	gameOver  bool
	paused    bool
	repo      *data.LeaderboardRepository
}

func NewGame(repo *data.LeaderboardRepository) (*Game, error) {
	matrix := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)

	scoring, err := snake.NewScoring(1, 10, 100, true, false)
	if err != nil {
		return nil, err
	}

	initialSnake := []snake.Point{
		{X: snake.DefaultCols / 2, Y: snake.DefaultRows / 2},
	}

	food := snake.NewFood(initialSnake)

	g := &Game{
		matrix:    matrix,
		snakeBody: initialSnake,
		food:      food,
		direction: snake.Right,
		nextDir:   snake.Right,
		scoring:   scoring,
		repo:      repo,
	}

	// Spawn the initial set of bombs for level 1.
	g.syncBombs()
	g.render()

	return g, nil
}

// ChangeDirection queues a direction change, preventing 180-degree reversals.
func (g *Game) ChangeDirection(d snake.Direction) {
	if g.gameOver || g.paused {
		return
	}

	if (d == snake.Up && g.direction == snake.Down) ||
		(d == snake.Down && g.direction == snake.Up) ||
		(d == snake.Left && g.direction == snake.Right) ||
		(d == snake.Right && g.direction == snake.Left) {
		return
	}

	g.nextDir = d
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

	// Update bomb lifecycles
	g.updateBombs()

	// Ensure bomb count matches the current level (level-up may require more).
	g.syncBombs()

	// Move snake
	g.direction = g.nextDir

	head := g.snakeBody[0]
	next := head

	switch g.direction {
	case snake.Up:
		next.Y--
	case snake.Down:
		next.Y++
	case snake.Left:
		next.X--
	case snake.Right:
		next.X++
	}

	// Wall collision.
	if !g.matrix.InBounds(next) {
		g.gameOver = true
		return
	}

	// Self collision.
	if g.isSelfCollision(next) {
		g.gameOver = true
		return
	}

	// Active bomb collision.
	if g.isActiveBombCollision(next) {
		g.gameOver = true
		return
	}

	// Food / grow
	ateFood := next.X == g.food.X && next.Y == g.food.Y

	g.snakeBody = append([]snake.Point{next}, g.snakeBody...)

	if ateFood {
		g.scoring.AddPoints(10)
		g.food = snake.NewFood(g.snakeBody)
	} else {
		g.snakeBody = g.snakeBody[:len(g.snakeBody)-1]
	}

	// Re-draw matrix
	g.render()
}

func (g *Game) SaveScore(name string) error {
	_, err := g.repo.Save(name, g.scoring.Total(), g.scoring.Level(), data.GameModeCrazy)
	return err
}

// updateBombs advances every bomb's state machine.
func (g *Game) updateBombs() {
	for _, b := range g.bombs {
		// Build occupied list that excludes this bomb's own point so it can
		// pick a new location freely when it resets.
		occupied := g.occupiedExcluding(b.Point)
		b.update(occupied)
	}
}

// syncBombs ensures the bomb slice contains exactly bombCountForLevel(level)
// entries, adding new ones (in warning phase) whenever the level rises.
func (g *Game) syncBombs() {
	want := bombCountForLevel(g.scoring.Level())
	for len(g.bombs) < want {
		g.bombs = append(g.bombs, newBomb(g.allOccupied()))
	}
}

// allOccupied returns every point currently taken by the snake, food, and
// already-placed bombs so new bombs don't spawn on top of them.
func (g *Game) allOccupied() []snake.Point {
	pts := make([]snake.Point, 0, len(g.snakeBody)+1+len(g.bombs))
	pts = append(pts, g.snakeBody...)
	if g.food != nil {
		pts = append(pts, *g.food)
	}
	for _, b := range g.bombs {
		pts = append(pts, b.Point)
	}
	return pts
}

// occupiedExcluding is like allOccupied but skips the given point (used when
// a bomb is relocating so it doesn't exclude its own current position).
func (g *Game) occupiedExcluding(exclude snake.Point) []snake.Point {
	pts := make([]snake.Point, 0, len(g.snakeBody)+1+len(g.bombs))
	pts = append(pts, g.snakeBody...)
	if g.food != nil {
		pts = append(pts, *g.food)
	}
	for _, b := range g.bombs {
		if b.Point != exclude {
			pts = append(pts, b.Point)
		}
	}
	return pts
}

// render writes the full current game state onto the matrix.
// Cell codes:
//
//	'H' – snake head
//	'S' – snake body
//	'F' – food
//	'B' – active (lethal) bomb
//	'W' – warning (blinking, not yet lethal) bomb
func (g *Game) render() {
	// Clear.
	for y := range g.matrix {
		for x := range g.matrix[y] {
			g.matrix[y][x] = 0
		}
	}

	// Food.
	if g.food != nil {
		g.matrix.Set(*g.food, 'F')
	}

	// Bombs (drawn before snake so head/body always wins any overlap).
	for _, b := range g.bombs {
		switch {
		case b.IsActive():
			g.matrix.Set(b.Point, 'B')
		case b.IsWarning():
			g.matrix.Set(b.Point, 'W')
		}
	}

	// Snake.
	for i, p := range g.snakeBody {
		if i == 0 {
			g.matrix.Set(p, 'H')
		} else {
			g.matrix.Set(p, 'S')
		}
	}
}

func (g *Game) isSelfCollision(p snake.Point) bool {
	return slices.Contains(g.snakeBody, p)
}

func (g *Game) isActiveBombCollision(p snake.Point) bool {
	for _, b := range g.bombs {
		if b.IsActive() && b.Point == p {
			return true
		}
	}
	return false
}
