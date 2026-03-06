package ai

import (
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

// Verify interface compliance at compile time.
var _ snake.AIGameController = (*Game)(nil)

// Game is the 1v1 AI mode. Both the player snake and the AI snake share the
// same board and compete for the same food pellet. Either snake can kill the
// other by cutting across its path:
//
//   - A snake that runs into the OTHER snake's body dies immediately.
//   - Head-on collision (both snakes step onto the same cell on the same tick)
//     kills both snakes and ends the game.
//   - Running into your own body or a wall also kills that snake.
type Game struct {
	matrix snake.Matrix

	// Player snake
	playerBody  []snake.Point
	playerDir   snake.Direction
	playerNext  snake.Direction
	playerScore *snake.Scoring
	playerAlive bool

	// AI snake
	aiBody  []snake.Point
	aiDir   snake.Direction
	aiScore *snake.Scoring
	aiAlive bool

	food     *snake.Point
	paused   bool
	gameOver bool

	repo *data.LeaderboardRepository
}

func NewGame(repo *data.LeaderboardRepository) (*Game, error) {
	matrix := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)

	playerScoring, err := snake.NewScoring(1, 10, 100, true, false)
	if err != nil {
		return nil, err
	}

	aiScoring, err := snake.NewScoring(1, 10, 100, true, false)
	if err != nil {
		return nil, err
	}

	// Spawn the two snakes on opposite sides of the board so they don't
	// immediately collide.
	playerStart := []snake.Point{
		{X: snake.DefaultCols / 4, Y: snake.DefaultRows / 2},
	}
	aiStart := []snake.Point{
		{X: (snake.DefaultCols * 3) / 4, Y: snake.DefaultRows / 2},
	}

	allOccupied := append(playerStart, aiStart...)
	food := snake.NewFood(allOccupied)

	g := &Game{
		matrix:      matrix,
		playerBody:  playerStart,
		playerDir:   snake.Right,
		playerNext:  snake.Right,
		playerScore: playerScoring,
		playerAlive: true,
		aiBody:      aiStart,
		aiDir:       snake.Left, // AI starts moving toward the player's side
		aiScore:     aiScoring,
		aiAlive:     true,
		food:        food,
		repo:        repo,
	}

	g.render()
	return g, nil
}

// ChangeDirection queues a player direction change, preventing 180-degree
// reversals.
func (g *Game) ChangeDirection(d snake.Direction) {
	if g.gameOver || g.paused || !g.playerAlive {
		return
	}

	if (d == snake.Up && g.playerDir == snake.Down) ||
		(d == snake.Down && g.playerDir == snake.Up) ||
		(d == snake.Left && g.playerDir == snake.Right) ||
		(d == snake.Right && g.playerDir == snake.Left) {
		return
	}

	g.playerNext = d
}

// TogglePause pauses or resumes the game.
func (g *Game) TogglePause() {
	if g.gameOver {
		return
	}
	g.paused = !g.paused
}

// Tick advances both snakes by one step simultaneously.
func (g *Game) Tick() {
	if g.gameOver || g.paused {
		return
	}

	// AI picks its next direction via BFS
	if g.aiAlive {
		occupied := occupiedSet(g.playerBody, g.aiBody)
		g.aiDir = nextDirection(g.aiBody[0], *g.food, g.playerBody[0], g.playerDir, g.aiDir, occupied)
	}

	//  Compute next head positions
	var playerNext, aiNext snake.Point
	if g.playerAlive {
		g.playerDir = g.playerNext
		playerNext = step(g.playerBody[0], g.playerDir)
	}
	if g.aiAlive {
		aiNext = step(g.aiBody[0], g.aiDir)
	}

	//  Head-on collision: both snakes step onto the same cell
	if g.playerAlive && g.aiAlive && playerNext == aiNext {
		g.playerAlive = false
		g.aiAlive = false
		g.gameOver = true
		g.render()
		return
	}

	//  Move player
	if g.playerAlive {
		if !g.matrix.InBounds(playerNext) || g.isSelfCollision(playerNext, g.playerBody) {
			g.playerAlive = false
		} else if g.isBodyCollision(playerNext, g.aiBody) {
			// Player ran into the AI's body.
			g.playerAlive = false
		}

		if g.playerAlive {
			ateFood := playerNext == *g.food
			g.playerBody = append([]snake.Point{playerNext}, g.playerBody...)
			if ateFood {
				g.playerScore.AddPoints(10)
				g.food = snake.NewFood(append(g.playerBody, g.aiBody...))
			} else {
				g.playerBody = g.playerBody[:len(g.playerBody)-1]
			}
		}
	}

	//  Move AI
	if g.aiAlive {
		if !g.matrix.InBounds(aiNext) || g.isSelfCollision(aiNext, g.aiBody) {
			g.aiAlive = false
		} else if g.isBodyCollision(aiNext, g.playerBody) {
			// AI ran into the player's body.
			g.aiAlive = false
		}

		if g.aiAlive {
			// The food might have been eaten by the player this tick — check
			// the current food position (which may have moved).
			ateFood := aiNext == *g.food
			g.aiBody = append([]snake.Point{aiNext}, g.aiBody...)
			if ateFood {
				g.aiScore.AddPoints(10)
				g.food = snake.NewFood(append(g.playerBody, g.aiBody...))
			} else {
				g.aiBody = g.aiBody[:len(g.aiBody)-1]
			}
		}
	}

	// Game ends when either snake dies.
	if !g.playerAlive || !g.aiAlive {
		g.gameOver = true
	}

	g.render()
}

func (g *Game) SaveScore(name string) error {
	_, err := g.repo.Save(name, g.playerScore.Total(), g.playerScore.Level(), data.GameModeNormal)
	return err
}

func (g *Game) isSelfCollision(p snake.Point, body []snake.Point) bool {
	for _, s := range body {
		if s == p {
			return true
		}
	}
	return false
}

func (g *Game) isBodyCollision(p snake.Point, body []snake.Point) bool {
	// Don't include the tail tip — it will vacate this tick.
	limit := len(body) - 1
	for i := 0; i < limit; i++ {
		if body[i] == p {
			return true
		}
	}
	return false
}

// render writes the full current game state onto the matrix.
// Cell codes:
//
//	'H' – player head      'S' – player body
//	'A' – AI head          'Z' – AI body
//	'F' – food
func (g *Game) render() {
	for y := range g.matrix {
		for x := range g.matrix[y] {
			g.matrix[y][x] = 0
		}
	}

	if g.food != nil {
		g.matrix.Set(*g.food, 'F')
	}

	// Draw AI first so the player always renders on top if they overlap.
	for i, p := range g.aiBody {
		if i == 0 {
			g.matrix.Set(p, 'A')
		} else {
			g.matrix.Set(p, 'Z')
		}
	}

	for i, p := range g.playerBody {
		if i == 0 {
			g.matrix.Set(p, 'H')
		} else {
			g.matrix.Set(p, 'S')
		}
	}
}
