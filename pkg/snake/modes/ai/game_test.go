package ai

import (
	"testing"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

func newTestGame(t *testing.T) *Game {
	t.Helper()
	g, err := NewGame(1)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestHeadOnCollisionIsADraw(t *testing.T) {
	g := newTestGame(t)
	g.playerBody = []snake.Point{{X: 10, Y: 10}}
	g.aiBody = []snake.Point{{X: 12, Y: 10}}
	g.playerSteer = snake.NewSteering(snake.Right)
	g.food = &snake.Point{X: 11, Y: 10}
	g.brain = func(*Game) snake.Direction { return snake.Left }

	g.Tick()

	if !g.IsGameOver() || g.Outcome() != snake.OutcomeDraw {
		t.Fatalf("gameOver=%v outcome=%v; want draw", g.IsGameOver(), g.Outcome())
	}
}

func TestHeadSwapIsADraw(t *testing.T) {
	g := newTestGame(t)
	g.playerBody = []snake.Point{{X: 10, Y: 10}}
	g.aiBody = []snake.Point{{X: 11, Y: 10}}
	g.playerSteer = snake.NewSteering(snake.Right)
	g.brain = func(*Game) snake.Direction { return snake.Left }

	g.Tick()

	if g.Outcome() != snake.OutcomeDraw {
		t.Fatalf("outcome = %v; snakes passed through each other", g.Outcome())
	}
}

func TestPlayerWinsWhenAIHitsWall(t *testing.T) {
	g := newTestGame(t)
	g.playerBody = []snake.Point{{X: 5, Y: 5}}
	g.playerSteer = snake.NewSteering(snake.Down)
	// AI boxed in at the corner with its own body: every move is fatal.
	g.aiBody = []snake.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}, {X: 0, Y: 2}}
	g.aiSteer = snake.NewSteering(snake.Left)
	g.food = &snake.Point{X: 30, Y: 30}

	g.Tick()

	if g.Outcome() != snake.OutcomeWin {
		t.Fatalf("outcome = %v, want win (aiAlive=%v playerAlive=%v)", g.Outcome(), g.aiAlive, g.playerAlive)
	}
}

func TestPlayerLosesWhenHittingAIBody(t *testing.T) {
	g := newTestGame(t)
	g.aiBody = []snake.Point{{X: 20, Y: 20}, {X: 20, Y: 21}, {X: 20, Y: 22}, {X: 20, Y: 23}}
	g.brain = func(*Game) snake.Direction { return snake.Up }
	g.playerBody = []snake.Point{{X: 19, Y: 22}}
	g.playerSteer = snake.NewSteering(snake.Right)
	g.food = &snake.Point{X: 20, Y: 5}

	g.Tick()

	if g.playerAlive {
		t.Fatal("player should die running into the AI's body")
	}
	if g.Outcome() != snake.OutcomeLoss {
		t.Fatalf("outcome = %v, want loss", g.Outcome())
	}
}

func TestAISurvivesManyTicks(t *testing.T) {
	g := newTestGame(t)
	// Keep the player out of the way, circling in a corner.
	turns := []snake.Direction{snake.Down, snake.Right, snake.Up, snake.Left}
	g.playerBody = []snake.Point{{X: 0, Y: 0}}
	g.playerSteer = snake.NewSteering(snake.Right)

	for i := range 40 {
		if i%2 == 0 {
			g.ChangeDirection(turns[(i/2)%len(turns)])
		}
		g.Tick()
		if !g.aiAlive {
			t.Fatalf("AI died on tick %d on an almost empty board", i)
		}
		if !g.playerAlive {
			return // the player wandering off is fine; the AI held up
		}
	}
}
