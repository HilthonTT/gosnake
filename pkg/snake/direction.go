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

// maxQueuedTurns is how many turns may be buffered ahead of the snake, so a
// quick "up, left" between two ticks performs both turns instead of
// dropping the second.
const maxQueuedTurns = 3

// Steering holds a snake's current heading and the turns queued for the
// next ticks. It rejects 180-degree reversals, which would make the snake run
// straight into its own neck.
type Steering struct {
	current Direction
	queue   []Direction
}

// NewSteering returns a Steering facing d.
func NewSteering(d Direction) Steering {
	return Steering{current: d}
}

// Current returns the direction the snake moved on the last tick.
func (s *Steering) Current() Direction { return s.current }

// last is the heading the snake will have once every queued turn is done.
func (s *Steering) last() Direction {
	if n := len(s.queue); n > 0 {
		return s.queue[n-1]
	}
	return s.current
}

// Queue requests a turn. Repeats, reversals of the previous heading and
// turns beyond the buffer size are ignored.
func (s *Steering) Queue(d Direction) {
	s.Turn(d, false)
}

// Turn is Queue with an option to allow reversing, which is safe for a snake
// that is a single cell long and so has no neck to run into.
func (s *Steering) Turn(d Direction, allowReverse bool) {
	last := s.last()
	if d == last || (d == last.Opposite() && !allowReverse) || len(s.queue) >= maxQueuedTurns {
		return
	}
	s.queue = append(s.queue, d)
}

// Advance commits the next queued turn, if any, and returns the heading.
// Call once per tick.
func (s *Steering) Advance() Direction {
	if len(s.queue) > 0 {
		s.current = s.queue[0]
		s.queue = s.queue[1:]
	}
	return s.current
}

// Set forces the heading and clears queued turns (used by the AI, which
// never queues reversals in the first place).
func (s *Steering) Set(d Direction) {
	s.current = d
	s.queue = s.queue[:0]
}
