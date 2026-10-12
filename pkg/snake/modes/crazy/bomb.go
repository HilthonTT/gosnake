package crazy

import (
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

// BombState describes which phase of the bomb lifecycle we're in.
type BombState int

const (
	// BombStateWarning: the bomb is blinking at its next position as a visual
	// cue before it becomes lethal. The snake can pass through safely.
	BombStateWarning BombState = iota

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

// newBomb creates a bomb in the warning phase at a random cell not in
// occupied. ok is false when the board is full.
func newBomb(occupied []snake.Point, now time.Time) (*Bomb, bool) {
	p, ok := snake.RandomFreePoint(occupied)
	if !ok {
		return nil, false
	}
	return &Bomb{
		Point:     p,
		State:     BombStateWarning,
		ChangesAt: now.Add(BombWarningDuration),
	}, true
}

// update advances the bomb's state machine. occupied is only evaluated when
// the bomb needs to relocate.
func (b *Bomb) update(now time.Time, occupied func() []snake.Point) {
	if now.Before(b.ChangesAt) {
		return
	}

	switch b.State {
	case BombStateWarning:
		b.State = BombStateActive
		b.ChangesAt = now.Add(BombActiveDuration)

	case BombStateActive:
		// Active period is over — pick a new spot and start warning again.
		if p, ok := snake.RandomFreePoint(occupied()); ok {
			b.Point = p
		}
		b.State = BombStateWarning
		b.ChangesAt = now.Add(BombWarningDuration)
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

// bombCountForLevel returns how many bombs should be present at a given level.
// Starts at 3 for level 1 and grows by 1 per level.
func bombCountForLevel(level int) int {
	return level + 2
}
