package single

import (
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

type Game struct {
	matrix    snake.Matrix
	snake     []snake.Point
	food      *snake.Point
	direction Direction
	nextDir   Direction
	scoring   *snake.Scoring
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
		snake:     initialSnake,
		food:      food,
		direction: Right,
		nextDir:   Right,
		scoring:   scoring,
		repo:      repo,
		paused:    false,
		gameOver:  false,
	}

	g.render()

	return g, nil
}

// ChangeDirection queues a direction change, preventing 180-degree reversals.
func (g *Game) ChangeDirection(d Direction) {
	if g.gameOver || g.paused {
		return
	}

	if (d == Up && g.direction == Down) ||
		(d == Down && g.direction == Up) ||
		(d == Left && g.direction == Right) ||
		(d == Right && g.direction == Left) {
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

// Tick advances the game by one step
func (g *Game) Tick() {
	if g.gameOver || g.paused {
		return
	}

	g.direction = g.nextDir

	head := g.snake[0]
	next := head

	switch g.direction {
	case Up:
		next.Y--
	case Down:
		next.Y++
	case Left:
		next.X--
	case Right:
		next.X++
	}

	// Check if the snake is colliding the wall
	if !g.matrix.InBounds(next) {
		g.triggerGameOver()
		return
	}

	// Check if the snake is colliding itself
	if g.isSelfCollision(next) {
		g.triggerGameOver()
		return
	}

	// Check if the food is eaten
	ateFood := next.X == g.food.X && next.Y == g.food.Y

	// Prepend new head
	g.snake = append([]snake.Point{next}, g.snake...)

	if ateFood {
		g.scoring.AddPoints(10)
		g.food = snake.NewFood(g.snake)
	} else {
		// Remove tail
		g.snake = g.snake[:len(g.snake)-1]
	}

	g.render()
}

func (g *Game) SaveScore(name string) error {
	return g.repo.Save(name, g.scoring.Total(), g.scoring.Level())
}

func (g *Game) triggerGameOver() {
	g.gameOver = true
}

// render writes the current game state onto the matrix.
func (g *Game) render() {
	// Clear matrix
	for y := range g.matrix {
		for x := range g.matrix[y] {
			g.matrix[y][x] = 0
		}
	}

	// Draw fool
	if g.food != nil {
		g.matrix.Set(*g.food, 'F')
	}

	// Draw snake
	for i, p := range g.snake {
		if i == 0 {
			g.matrix.Set(p, 'H')
		} else {
			g.matrix.Set(p, 'S')
		}
	}
}

// isSelfCollision determines if the snake is colliding itself
func (g *Game) isSelfCollision(p snake.Point) bool {
	for _, s := range g.snake {
		if s.X == p.X && s.Y == p.Y {
			return true
		}
	}
	return false
}
