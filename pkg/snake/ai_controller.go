package snake

// AIGameController extends GameController for modes that pit the player
// against an AI opponent. The view type-asserts to this interface to decide
// whether to render the second-snake info panel and AI cells.
type AIGameController interface {
	GameController

	// AIScore returns the AI snake's current score.
	AIScore() int

	// AISnakeLength returns the number of cells the AI snake currently occupies.
	AISnakeLength() int

	// IsAIAlive reports whether the AI snake is still in play.
	IsAIAlive() bool

	// IsPlayerAlive reports whether the player snake is still in play.
	// Both snakes can die independently; the game ends when either is dead.
	IsPlayerAlive() bool
}
