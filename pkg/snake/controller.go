package snake

import "time"

// GameController is the common interface implemented by every single-player
// mode. The TUI holds this interface so the view is mode-agnostic.
type GameController interface {
	Matrix() Matrix
	IsGameOver() bool
	IsPaused() bool
	Score() int
	Level() int
	Snake() []Point
	SnakeLength() int
	Food() *Point
	GetTickInterval() time.Duration

	Tick()
	TogglePause()
	ChangeDirection(Direction)
}

// Outcome describes how a versus game ended from the player's perspective.
type Outcome int

const (
	OutcomePending Outcome = iota
	OutcomeWin
	OutcomeLoss
	OutcomeDraw
)

// AIGameController extends GameController for modes that pit the player
// against an AI opponent. The view type-asserts to this interface to decide
// whether to render the second-snake info panel.
type AIGameController interface {
	GameController

	// AIScore returns the AI snake's current score.
	AIScore() int

	// AISnakeLength returns the number of cells the AI snake currently occupies.
	AISnakeLength() int

	// IsAIAlive reports whether the AI snake is still in play.
	IsAIAlive() bool

	// IsPlayerAlive reports whether the player snake is still in play.
	IsPlayerAlive() bool

	// Outcome reports who won once the game is over.
	Outcome() Outcome
}
