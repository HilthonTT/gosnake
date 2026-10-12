package crazy

import (
	"testing"
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/single"
)

func TestBombLifecycle(t *testing.T) {
	now := time.Unix(0, 0)
	g, err := newGame(1, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	if len(g.Bombs()) != bombCountForLevel(1) {
		t.Fatalf("bombs = %d, want %d", len(g.Bombs()), bombCountForLevel(1))
	}
	for _, b := range g.Bombs() {
		if !b.IsWarning() {
			t.Fatal("new bombs should start in the warning phase")
		}
		if g.Matrix().Get(b.Point) != snake.CellBombWarning {
			t.Fatal("warning bombs should be drawn")
		}
	}

	now = now.Add(BombWarningDuration)
	g.updateBombs()
	for _, b := range g.Bombs() {
		if !b.IsActive() {
			t.Fatal("bombs should become active after the warning period")
		}
	}

	now = now.Add(BombActiveDuration)
	g.updateBombs()
	if !g.Bombs()[0].IsWarning() {
		t.Fatal("bombs should cycle back to warning")
	}
}

func TestActiveBombIsFatal(t *testing.T) {
	now := time.Unix(0, 0)
	g, err := newGame(1, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	ahead := g.Snake()[0].Step(snake.Right)
	g.bombs[0].Point = ahead
	g.bombs[0].State = BombStateActive
	g.bombs[0].ChangesAt = now.Add(time.Hour)

	g.Tick()

	if !g.IsGameOver() {
		t.Fatal("stepping on an active bomb should end the game")
	}
}

func TestWarningBombIsHarmless(t *testing.T) {
	now := time.Unix(0, 0)
	g, err := newGame(1, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	g.bombs[0].Point = g.Snake()[0].Step(snake.Right)

	g.Tick()

	if g.IsGameOver() {
		t.Fatal("warning bombs should not be lethal")
	}
}

func TestFoodNeverSpawnsOnBombs(t *testing.T) {
	now := time.Unix(0, 0)
	for range 50 {
		g, err := newGame(snake.MaxLevel, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}

		// Feed the snake so food respawns while bombs are on the board.
		ahead := g.Snake()[0].Step(snake.Right)
		for _, b := range g.Bombs() {
			if b.Point == ahead {
				b.Point = snake.Point{X: 0, Y: 0}
			}
		}
		g.Game = feed(t, g, ahead)

		for _, b := range g.Bombs() {
			if f := g.Food(); f != nil && *f == b.Point {
				t.Fatalf("food spawned on bomb at %v", b.Point)
			}
		}
	}
}

// feed places food directly ahead of the snake and ticks once.
func feed(t *testing.T, g *Game, at snake.Point) *single.Game {
	t.Helper()
	before := g.Score()
	g.SetFood(at)
	g.Tick()
	if g.IsGameOver() || g.Score() == before {
		t.Fatalf("expected the snake to eat at %v", at)
	}
	return g.Game
}
