// Package crazy implements the classic rules plus timed bombs. It reuses the
// single-player engine through its hooks instead of duplicating it.
package crazy

import (
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/single"
)

// Ensure *Game satisfies the shared controller interface at compile time.
var _ snake.GameController = (*Game)(nil)

// Game is the crazy-mode variant of the snake game. It behaves identically to
// the normal mode except that timed bombs are scattered around the board.
// Bombs cycle through a warning (blinking) phase followed by an active (lethal)
// phase, and their count scales with the current level.
type Game struct {
	*single.Game

	bombs []*Bomb
	now   func() time.Time
}

// NewGame starts a crazy game at the given level (1-10).
func NewGame(level int) (*Game, error) {
	return newGame(level, time.Now)
}

func newGame(level int, now func() time.Time) (*Game, error) {
	g := &Game{now: now}

	base, err := single.NewGameWithHooks(level, single.Hooks{
		BeforeTick: g.updateBombs,
		IsHazard:   g.isActiveBomb,
		Occupied:   g.bombPoints,
		Decorate:   g.drawBombs,
	})
	if err != nil {
		return nil, err
	}
	g.Game = base

	// Spawn the initial set of bombs for the starting level.
	g.syncBombs()
	g.Refresh()

	return g, nil
}

// Bombs returns the bombs currently on the board.
func (g *Game) Bombs() []*Bomb { return g.bombs }

// updateBombs advances every bomb's state machine and adds bombs when the
// level rises.
func (g *Game) updateBombs() {
	for _, b := range g.bombs {
		b.update(g.now(), func() []snake.Point { return g.occupiedExcept(b) })
	}
	g.syncBombs()
}

// syncBombs ensures the bomb slice contains bombCountForLevel(level) entries,
// adding new ones (in warning phase) whenever the level rises.
func (g *Game) syncBombs() {
	want := bombCountForLevel(g.Level())
	for len(g.bombs) < want {
		b, ok := newBomb(g.occupiedExcept(nil), g.now())
		if !ok {
			return
		}
		g.bombs = append(g.bombs, b)
	}
}

// occupiedExcept returns every cell taken by the snake, food and bombs other
// than skip, so bombs never spawn on top of anything.
func (g *Game) occupiedExcept(skip *Bomb) []snake.Point {
	pts := append([]snake.Point(nil), g.Snake()...)
	if f := g.Food(); f != nil {
		pts = append(pts, *f)
	}
	for _, b := range g.bombs {
		if b != skip {
			pts = append(pts, b.Point)
		}
	}
	return pts
}

func (g *Game) bombPoints() []snake.Point {
	pts := make([]snake.Point, len(g.bombs))
	for i, b := range g.bombs {
		pts[i] = b.Point
	}
	return pts
}

func (g *Game) isActiveBomb(p snake.Point) bool {
	for _, b := range g.bombs {
		if b.IsActive() && b.Point == p {
			return true
		}
	}
	return false
}

// drawBombs renders bombs. Cell codes: snake.CellBomb for active (lethal)
// bombs and snake.CellBombWarning for blinking, not-yet-lethal ones.
func (g *Game) drawBombs(m snake.Matrix) {
	for _, b := range g.bombs {
		if b.IsActive() {
			m.Set(b.Point, snake.CellBomb)
		} else {
			m.Set(b.Point, snake.CellBombWarning)
		}
	}
}

func (g *Game) Snapshot() map[string]any {
	snap := g.Game.Snapshot()

	active, warning := 0, 0
	for _, b := range g.bombs {
		if b.IsActive() {
			active++
		} else {
			warning++
		}
	}
	snap["bombCount"] = len(g.bombs)
	snap["activeBombs"] = active
	snap["warningBombs"] = warning

	return snap
}
