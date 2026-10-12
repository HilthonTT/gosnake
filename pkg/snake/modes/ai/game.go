// Package ai implements a 1v1 mode against a computer-controlled snake.
package ai

import (
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

// Verify interface compliance at compile time.
var _ snake.AIGameController = (*Game)(nil)

// Game is the 1v1 AI mode. Both the player snake and the AI snake share the
// same board and compete for the same food pellet. Moves are resolved
// simultaneously:
//
//   - A snake that runs into the OTHER snake's body dies.
//   - Head-on collision (both snakes step onto the same cell on the same tick)
//     kills both snakes — a draw.
//   - Running into your own body or a wall also kills that snake.
//
// The game ends as soon as either snake dies.
type Game struct {
	matrix snake.Matrix

	// Player snake
	playerBody  []snake.Point
	playerSteer snake.Steering
	playerScore *snake.Scoring
	playerAlive bool

	// AI snake
	aiBody  []snake.Point
	aiSteer snake.Steering
	aiScore *snake.Scoring
	aiAlive bool
	aiSt    *aiState
	brain   Brain

	food     *snake.Point
	paused   bool
	gameOver bool
}

// Brain decides the AI snake's next direction. It is only called while the
// AI is alive and there is food on the board.
type Brain func(g *Game) snake.Direction

// pathfinderBrain is the default Brain: a level-scaled mix of safe BFS,
// aggressive intercepts and deliberate mistakes (see nextDirection).
func pathfinderBrain(g *Game) snake.Direction {
	return nextDirection(
		g.aiBody[0],
		*g.food,
		g.playerBody[0],
		g.playerSteer.Current(),
		g.aiSteer.Current(),
		occupiedSet(g.playerBody, g.aiBody),
		g.aiBody,
		g.playerScore.Level(),
		g.aiSt,
	)
}

// NewGame starts a versus game at the given level (1-10). The level controls
// both the speed and how sharp the AI plays.
func NewGame(level int) (*Game, error) {
	playerScoring, err := snake.NewDefaultScoring(level)
	if err != nil {
		return nil, err
	}

	aiScoring, err := snake.NewDefaultScoring(level)
	if err != nil {
		return nil, err
	}

	// Spawn the two snakes on opposite sides of the board so they don't
	// immediately collide.
	playerStart := []snake.Point{{X: snake.DefaultCols / 4, Y: snake.DefaultRows / 2}}
	aiStart := []snake.Point{{X: (snake.DefaultCols * 3) / 4, Y: snake.DefaultRows / 2}}

	g := &Game{
		matrix:      snake.NewMatrix(snake.DefaultRows, snake.DefaultCols),
		playerBody:  playerStart,
		playerSteer: snake.NewSteering(snake.Right),
		playerScore: playerScoring,
		playerAlive: true,
		aiBody:      aiStart,
		aiSteer:     snake.NewSteering(snake.Left),
		aiScore:     aiScoring,
		aiAlive:     true,
		aiSt:        newAIState(),
		brain:       pathfinderBrain,
	}
	g.food = snake.NewFood(g.allBodies())

	g.render()
	return g, nil
}

// ChangeDirection queues a player direction change, preventing 180-degree
// reversals.
func (g *Game) ChangeDirection(d snake.Direction) {
	if g.gameOver || g.paused || !g.playerAlive {
		return
	}
	g.playerSteer.Queue(d)
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

	// The AI picks its next direction with full knowledge of the board.
	if g.aiAlive && g.food != nil {
		g.aiSteer.Set(g.brain(g))
	}

	playerNext := g.playerBody[0].Step(g.playerSteer.Advance())
	aiNext := g.aiBody[0].Step(g.aiSteer.Advance())

	// Head-on collision: both snakes step onto the same cell, or swap cells.
	if playerNext == aiNext || (playerNext == g.aiBody[0] && aiNext == g.playerBody[0]) {
		g.playerAlive, g.aiAlive, g.gameOver = false, false, true
		g.render()
		return
	}

	playerGrows := g.food != nil && playerNext == *g.food
	aiGrows := g.food != nil && aiNext == *g.food

	newPlayer := snake.Advance(g.playerBody, playerNext, playerGrows)
	newAI := snake.Advance(g.aiBody, aiNext, aiGrows)

	// Judge both moves against the post-move board so a tail that moves away
	// this tick is not an obstacle, but one that stays (because it ate) is.
	g.playerAlive = g.matrix.InBounds(playerNext) &&
		!snake.HitsBody(newPlayer[1:], playerNext, false) &&
		!snake.HitsBody(newAI, playerNext, false)
	g.aiAlive = g.matrix.InBounds(aiNext) &&
		!snake.HitsBody(newAI[1:], aiNext, false) &&
		!snake.HitsBody(newPlayer, aiNext, false)

	if !g.playerAlive || !g.aiAlive {
		g.gameOver = true
		g.render()
		return
	}

	g.playerBody, g.aiBody = newPlayer, newAI
	if playerGrows {
		g.playerScore.AddPoints(snake.PointsPerFood)
	}
	if aiGrows {
		g.aiScore.AddPoints(snake.PointsPerFood)
	}
	if playerGrows || aiGrows {
		g.food = snake.NewFood(g.allBodies())
	}

	g.render()
}

// Outcome reports who won once the game is over.
func (g *Game) Outcome() snake.Outcome {
	switch {
	case !g.gameOver:
		return snake.OutcomePending
	case g.playerAlive && !g.aiAlive:
		return snake.OutcomeWin
	case !g.playerAlive && g.aiAlive:
		return snake.OutcomeLoss
	default:
		return snake.OutcomeDraw
	}
}

func (g *Game) allBodies() []snake.Point {
	return append(append([]snake.Point(nil), g.playerBody...), g.aiBody...)
}

// render writes the full current game state onto the matrix.
func (g *Game) render() {
	g.matrix.Clear()

	if g.food != nil {
		g.matrix.Set(*g.food, snake.CellFood)
	}
	g.matrix.DrawSnake(g.aiBody, snake.CellAIHead, snake.CellAIBody)
	g.matrix.DrawSnake(g.playerBody, snake.CellHead, snake.CellBody)
}

func (g *Game) Matrix() snake.Matrix           { return g.matrix }
func (g *Game) IsGameOver() bool               { return g.gameOver }
func (g *Game) IsPaused() bool                 { return g.paused }
func (g *Game) Score() int                     { return g.playerScore.Total() }
func (g *Game) Level() int                     { return g.playerScore.Level() }
func (g *Game) Snake() []snake.Point           { return g.playerBody }
func (g *Game) SnakeLength() int               { return len(g.playerBody) }
func (g *Game) Food() *snake.Point             { return g.food }
func (g *Game) GetTickInterval() time.Duration { return snake.GetTickInterval(g.playerScore.Level()) }

func (g *Game) AIScore() int        { return g.aiScore.Total() }
func (g *Game) AISnakeLength() int  { return len(g.aiBody) }
func (g *Game) IsAIAlive() bool     { return g.aiAlive }
func (g *Game) IsPlayerAlive() bool { return g.playerAlive }

// Snapshot returns a map of the current game state for crash reports.
func (g *Game) Snapshot() map[string]any {
	snap := map[string]any{
		"playerScore":    g.playerScore.Total(),
		"playerLevel":    g.playerScore.Level(),
		"playerDir":      g.playerSteer.Current().String(),
		"playerLen":      len(g.playerBody),
		"playerAlive":    g.playerAlive,
		"aiScore":        g.aiScore.Total(),
		"aiDir":          g.aiSteer.Current().String(),
		"aiLen":          len(g.aiBody),
		"aiAlive":        g.aiAlive,
		"aiTailChaseTks": g.aiSt.tailChaseTicks,
		"food":           g.food,
		"paused":         g.paused,
		"gameOver":       g.gameOver,
	}

	if len(g.playerBody) > 0 {
		snap["playerHead"] = g.playerBody[0]
	}
	if len(g.aiBody) > 0 {
		snap["aiHead"] = g.aiBody[0]
	}

	return snap
}
