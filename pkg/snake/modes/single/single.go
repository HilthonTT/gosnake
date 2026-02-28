package single

import (
	"slices"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

var _ snake.GameController = (*Game)(nil)

type Game struct {
	matrix    snake.Matrix
	snakeBody []snake.Point
	food      *snake.Point
	direction snake.Direction
	nextDir   snake.Direction
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
		snakeBody: initialSnake,
		food:      food,
		direction: snake.Right,
		nextDir:   snake.Right,
		scoring:   scoring,
		repo:      repo,
		paused:    false,
		gameOver:  false,
	}

	g.render()

	return g, nil
}

// ChangeDirection queues a direction change, preventing 180-degree reversals.snake.
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

// Tick advances the game by one step
func (g *Game) Tick() {
	if g.gameOver || g.paused {
		return
	}

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
	g.snakeBody = append([]snake.Point{next}, g.snakeBody...)

	if ateFood {
		g.scoring.AddPoints(10)
		g.food = snake.NewFood(g.snakeBody)
	} else {
		// Remove tail
		g.snakeBody = g.snakeBody[:len(g.snakeBody)-1]
	}

	g.render()
}

func (g *Game) SaveScore(name string) error {
	_, err := g.repo.Save(name, g.scoring.Total(), g.scoring.Level(), data.GameModeNormal)
	return err
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

	// Draw food
	if g.food != nil {
		g.matrix.Set(*g.food, 'F')
	}

	// Draw snake
	for i, p := range g.snakeBody {
		if i == 0 {
			g.matrix.Set(p, 'H')
		} else {
			g.matrix.Set(p, 'S')
		}
	}
}

// isSelfCollision determines if the snake is colliding itself
func (g *Game) isSelfCollision(p snake.Point) bool {
	return slices.Contains(g.snakeBody, p)
}
