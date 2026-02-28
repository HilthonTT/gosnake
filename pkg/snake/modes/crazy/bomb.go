package crazy

import (
	"math/rand"
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

// BombState describes which phase of the bomb lifecycle we're in
type BombState int

const (
	// BombStateWarning: the bomb is blinking at its next position as a visual
	// cue before it becomes lethal. The snake can pass through safely.
	BombStateWarning = iota

	// BombStateActive: the bomb is live. Touching it kills the snake.
	BombStateActive
)

const (
	// How long a bomb stays lethal before cycling back to a warning.
	BombActiveDuration = 10 * time.Second

	// How long the blinking warning lasts before the bomb goes active.
	BombWarningDuration = 2 * time.Second

	// How fast the cell blinks during the warning phase (toggle every N ms).
	BombBlinkPeriodMs = 250
)

// Bomb represents a single timed hazard on the board.
type Bomb struct {
	Point     snake.Point
	State     BombState
	ChangesAt time.Time // wall-clock deadline for the next state transition
}

// newBomb creates a bomb in the warning phase at a random unoccupied position.
// occupied should contain all points that must not overlap (snake, food, other
// bombs currently active)
func newBomb(occupied []snake.Point) *Bomb {
	p := randomFreePoint(occupied)
	return &Bomb{
		Point:     p,
		State:     BombStateWarning,
		ChangesAt: time.Now().Add(BombWarningDuration),
	}
}

func (b *Bomb) update(occupied []snake.Point) {
	if time.Now().Before(b.ChangesAt) {
		return
	}

	switch b.State {
	case BombStateWarning:
		b.State = BombStateActive
		b.ChangesAt = time.Now().Add(BombActiveDuration)

	case BombStateActive:
		// Active period is over — pick a new spot and start warning again.
		b.Point = randomFreePoint(occupied)
		b.State = BombStateWarning
		b.ChangesAt = time.Now().Add(BombWarningDuration)
	}
}

// IsActive returns true while the bomb is lethal.
func (b *Bomb) IsActive() bool {
	return b.State == BombStateActive
}

// IsWarning returns true while the bomb is in its pre-appearance blink phase.
func (b *Bomb) IsWarning() bool {
	return b.State == BombStateWarning
}

// ShouldRenderWarning returns true on the "on" half of the blink cycle so
// that the caller can alternately show/hide the cell to produce a flash effect.
// This is purely cosmetic — it doesn't affect game logic.
func (b *Bomb) ShouldRenderWarning() bool {
	return (time.Now().UnixMilli()/BombBlinkPeriodMs)%2 == 0
}

// bombCountForLevel returns how many bombs should be active at a given level.
// Starts at 3 for level 1 and grows by 1 per level.
func bombCountForLevel(level int) int {
	return level + 2 // level 1 -> 3, level 2 -> 4, …
}

// randomFreePoint picks a random board cell that does not appear in occupied.
func randomFreePoint(occupied []snake.Point) snake.Point {
	for {
		p := snake.Point{
			X: rand.Intn(snake.DefaultCols),
			Y: rand.Intn(snake.DefaultRows),
		}
		if !pointIn(occupied, p) {
			return p
		}
	}
}

// pointIn is a small linear search helper.
func pointIn(pts []snake.Point, p snake.Point) bool {
	for _, q := range pts {
		if q == p {
			return true
		}
	}
	return false
}
