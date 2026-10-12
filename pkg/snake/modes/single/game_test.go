package single

import (
	"testing"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

func newTestGame(t *testing.T, level int) *Game {
	t.Helper()
	g, err := NewGame(level)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestNewGameHonoursStartingLevel(t *testing.T) {
	g := newTestGame(t, 4)
	if g.Level() != 4 {
		t.Fatalf("level = %d, want 4", g.Level())
	}
	if g.GetTickInterval() != snake.GetTickInterval(4) {
		t.Fatal("tick interval should match the starting level")
	}
	if _, err := NewGame(snake.MaxLevel + 1); err == nil {
		t.Fatal("invalid level should be rejected")
	}
}

func TestEatingGrowsAndScores(t *testing.T) {
	g := newTestGame(t, 1)
	head := g.body[0]
	food := head.Step(snake.Right)
	g.food = &food

	g.Tick()

	if g.SnakeLength() != 2 || g.Score() != snake.PointsPerFood {
		t.Fatalf("after eating: len=%d score=%d", g.SnakeLength(), g.Score())
	}
	if g.Food() == nil || *g.Food() == food {
		t.Fatal("food should respawn elsewhere")
	}
	if g.Matrix().Get(g.body[0]) != snake.CellHead {
		t.Fatal("head should be rendered")
	}
}

func TestWallEndsGame(t *testing.T) {
	g := newTestGame(t, 1)
	g.body = []snake.Point{{X: snake.DefaultCols - 1, Y: 0}}
	g.food = nil

	g.Tick()

	if !g.IsGameOver() {
		t.Fatal("running into the wall should end the game")
	}
}

func TestSnakeCanFollowItsOwnTail(t *testing.T) {
	g := newTestGame(t, 1)
	// A 2x2 loop: moving up from (1,1) lands on (1,0), which is the tail.
	g.body = []snake.Point{{X: 1, Y: 1}, {X: 0, Y: 1}, {X: 0, Y: 0}, {X: 1, Y: 0}}
	g.steering = snake.NewSteering(snake.Right)
	g.ChangeDirection(snake.Up)
	g.food = &snake.Point{X: 20, Y: 20}

	g.Tick()

	if g.IsGameOver() {
		t.Fatal("moving into the cell the tail vacates should be safe")
	}
}

func TestSelfCollisionEndsGame(t *testing.T) {
	g := newTestGame(t, 1)
	g.body = []snake.Point{{X: 1, Y: 1}, {X: 0, Y: 1}, {X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}}
	g.steering = snake.NewSteering(snake.Right)
	g.ChangeDirection(snake.Up)
	g.food = &snake.Point{X: 20, Y: 20}

	g.Tick()

	if !g.IsGameOver() {
		t.Fatal("running into the body should end the game")
	}
}

func TestPauseFreezesGame(t *testing.T) {
	g := newTestGame(t, 1)
	head := g.body[0]

	g.TogglePause()
	g.Tick()
	g.ChangeDirection(snake.Up)

	if g.body[0] != head {
		t.Fatal("paused game should not move")
	}
	if g.steering.Advance() != snake.Right {
		t.Fatal("paused game should ignore input")
	}
}

func TestHooks(t *testing.T) {
	hazard := snake.Point{X: snake.DefaultCols/2 + 1, Y: snake.DefaultRows / 2}
	ticks := 0

	g, err := NewGameWithHooks(1, Hooks{
		BeforeTick: func() { ticks++ },
		IsHazard:   func(p snake.Point) bool { return p == hazard },
		Decorate:   func(m snake.Matrix) { m.Set(hazard, snake.CellBomb) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.Matrix().Get(hazard) != snake.CellBomb {
		t.Fatal("Decorate should draw onto the matrix")
	}

	g.Tick()

	if ticks != 1 || !g.IsGameOver() {
		t.Fatalf("ticks=%d gameOver=%v; hazard should be fatal", ticks, g.IsGameOver())
	}
}
