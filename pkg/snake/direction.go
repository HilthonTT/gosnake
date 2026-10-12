package snake

// Direction represents the snake's cardinal movement direction.
type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

// Directions lists every cardinal direction in a stable order.
var Directions = [...]Direction{Up, Down, Left, Right}

// Opposite returns the direction pointing the other way.
func (d Direction) Opposite() Direction {
	switch d {
	case Up:
		return Down
	case Down:
		return Up
	case Left:
		return Right
	default:
		return Left
	}
}

func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	case Left:
		return "left"
	case Right:
		return "right"
	default:
		return "unknown"
	}
}

// Steering holds a snake's current heading and the heading queued for the
// next tick. It rejects 180-degree reversals, which would make the snake run
// straight into its own neck.
type Steering struct {
	current Direction
	next    Direction
}

// NewSteering returns a Steering facing d.
func NewSteering(d Direction) Steering {
	return Steering{current: d, next: d}
}

// Current returns the direction the snake moved on the last tick.
func (s *Steering) Current() Direction { return s.current }

// Queue requests a turn for the next tick. Reversals are ignored.
func (s *Steering) Queue(d Direction) {
	if d == s.current.Opposite() {
		return
	}
	s.next = d
}

// Advance commits the queued direction and returns it. Call once per tick.
func (s *Steering) Advance() Direction {
	s.current = s.next
	return s.current
}

// Set forces both the current and queued direction (used by the AI, which
// never queues reversals in the first place).
func (s *Steering) Set(d Direction) {
	s.current = d
	s.next = d
}
