package multi

import (
	"slices"
	"testing"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

func TestNewGameCapsPlayers(t *testing.T) {
	g := NewGame([]string{"a", "b", "c", "d"})
	if len(g.Players()) != MaxPlayers {
		t.Fatalf("players = %d, want %d", len(g.Players()), MaxPlayers)
	}
}

func TestHeadToHeadKillsBoth(t *testing.T) {
	g := NewGame([]string{"a", "b"})
	a, b := g.players[0], g.players[1]
	a.Points = []snake.Point{{X: 10, Y: 10}}
	a.steering = snake.NewSteering(snake.Right)
	b.Points = []snake.Point{{X: 12, Y: 10}}
	b.steering = snake.NewSteering(snake.Left)

	died := g.Tick()

	slices.Sort(died)
	if !slices.Equal(died, []int{0, 1}) {
		t.Fatalf("died = %v, want both", died)
	}
	if !g.IsOver() || g.Winner() != -1 {
		t.Fatalf("over=%v winner=%d; want draw", g.IsOver(), g.Winner())
	}
}

func TestDeadSnakesAreNotObstacles(t *testing.T) {
	g := NewGame([]string{"a", "b", "c"})
	a, b, c := g.players[0], g.players[1], g.players[2]

	// c is dead; its old body lies right in front of a.
	c.Alive = false
	c.Points = []snake.Point{{X: 6, Y: 5}, {X: 7, Y: 5}}

	a.Points = []snake.Point{{X: 5, Y: 5}}
	a.steering = snake.NewSteering(snake.Right)
	b.Points = []snake.Point{{X: 30, Y: 30}}
	b.steering = snake.NewSteering(snake.Up)
	g.food = []*snake.Point{{X: 40, Y: 0}}

	died := g.Tick()

	if len(died) != 0 || !a.Alive {
		t.Fatalf("died = %v; a dead snake's invisible body killed a player", died)
	}
}

func TestLastSnakeStandingWins(t *testing.T) {
	g := NewGame([]string{"a", "b"})
	a, b := g.players[0], g.players[1]
	a.Points = []snake.Point{{X: 0, Y: 0}}
	a.steering = snake.NewSteering(snake.Up) // straight into the wall
	b.Points = []snake.Point{{X: 20, Y: 20}}
	b.steering = snake.NewSteering(snake.Up)

	g.Tick()

	if !g.IsOver() || g.Winner() != 1 {
		t.Fatalf("over=%v winner=%d; want player 1", g.IsOver(), g.Winner())
	}
}

func TestEatingScoresAndRespawnsFood(t *testing.T) {
	g := NewGame([]string{"a", "b"})
	a := g.players[0]
	target := a.Points[0].Step(a.Direction())
	g.food[0] = &target

	g.Tick()

	if a.Score != snake.PointsPerFood || len(a.Points) != 2 || g.FoodCount() != 1 {
		t.Fatalf("score=%d len=%d foodCount=%d", a.Score, len(a.Points), g.FoodCount())
	}
	if g.food[0] == nil || *g.food[0] == target {
		t.Fatal("food should respawn")
	}
}
