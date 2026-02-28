package snake

import "time"

// GameController is the common interface implemented by both single.Game and
// crazy.Game. SingleModel holds this interface so the view is mode-agnostic.
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
	GetDefaultTickInterval() time.Duration

	Tick()
	TogglePause()
	ChangeDirection(Direction)
}
