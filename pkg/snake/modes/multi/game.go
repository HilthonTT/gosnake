package multi

import "github.com/HilthonTT/gosnake/pkg/snake"

const MaxPlayers = 3

// HeadCells and BodyCells map a player index to its matrix byte value.
// Exported so the TUI layer can build its render switch without magic literals.
var HeadCells = [MaxPlayers]byte{'H', 'A', 'X'}
var BodyCells = [MaxPlayers]byte{'S', 'B', 'Y'}

// Direction mirrors the single-player type so callers don't import that package.
type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

// PlayerSnake is one player's snake state.
type PlayerSnake struct {
	Index     int
	Name      string
	Points    []snake.Point
	Direction Direction
	nextDir   Direction
	Alive     bool
	Score     int
}

// Game is the authoritative multiplayer game state.
// It is driven entirely by the server's tick goroutine; no Bubble Tea dependency.
type Game struct {
	matrix    snake.Matrix
	players   []*PlayerSnake
	food      []*snake.Point
	over      bool
	winner    int // -1 = draw, 0-2 = winning player index
	foodCount int // total food eaten globally; drives level calculation
}

// NewGame initialises a fresh game for the given player names.
// len(names) must be between 2 and MaxPlayers.
func NewGame(names []string) *Game {
	n := len(names)
	if n > MaxPlayers {
		n = MaxPlayers
	}

	// Fixed spread so snakes start far apart and face inward.
	starts := [MaxPlayers]snake.Point{
		{X: snake.DefaultCols / 4, Y: snake.DefaultRows / 2},
		{X: 3 * snake.DefaultCols / 4, Y: snake.DefaultRows / 2},
		{X: snake.DefaultCols / 2, Y: snake.DefaultRows / 4},
	}
	startDirs := [MaxPlayers]Direction{Right, Left, Down}

	players := make([]*PlayerSnake, n)
	var allPts []snake.Point

	for i := 0; i < n; i++ {
		name := "Player"
		if i < len(names) {
			name = names[i]
		}
		players[i] = &PlayerSnake{
			Index:     i,
			Name:      name,
			Points:    []snake.Point{starts[i]},
			Direction: startDirs[i],
			nextDir:   startDirs[i],
			Alive:     true,
		}
		allPts = append(allPts, starts[i])
	}

	// One food item per player, placed away from all snakes.
	foods := make([]*snake.Point, n)
	for i := range foods {
		f := snake.NewFood(allPts)
		foods[i] = f
		allPts = append(allPts, *f)
	}

	g := &Game{
		matrix:  snake.NewMatrix(snake.DefaultRows, snake.DefaultCols),
		players: players,
		food:    foods,
		over:    false,
		winner:  -1,
	}
	g.render()

	return g
}

// ChangeDirection queues a direction change for a player, rejecting 180° reversals.
// Safe to call from any goroutine that owns the room lock.
func (g *Game) ChangeDirection(playerIndex int, d Direction) {
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return
	}
	p := g.players[playerIndex]
	if !p.Alive {
		return
	}
	cur := p.Direction
	if (d == Up && cur == Down) || (d == Down && cur == Up) ||
		(d == Left && cur == Right) || (d == Right && cur == Left) {
		return
	}
	p.nextDir = d
}

// Tick advances every alive snake by one step and resolves all collisions.
// It returns the indices of players that died this tick.
func (g *Game) Tick() []int {
	if g.over {
		return nil
	}

	type move struct {
		p    *PlayerSnake
		next snake.Point
	}

	// Compute intended next positions.
	pending := make([]move, 0, len(g.players))
	for _, p := range g.players {
		if !p.Alive {
			continue
		}

		p.Direction = p.nextDir
		h := p.Points[0]
		n := h

		switch p.Direction {
		case Up:
			n.Y--
		case Down:
			n.Y++
		case Left:
			n.X--
		case Right:
			n.X++
		}
		pending = append(pending, move{p, n})
	}

	// 1. Wall collisions.
	for i := range pending {
		if !g.matrix.InBounds(pending[i].next) {
			pending[i].p.Alive = false
		}
	}

	// 2. Head-to-head collisions (both snakes die).
	for i := 0; i < len(pending); i++ {
		for j := i + 1; j < len(pending); j++ {
			if !pending[i].p.Alive || !pending[j].p.Alive {
				continue
			}

			if pending[i].next == pending[j].next {
				pending[i].p.Alive = false
				pending[j].p.Alive = false
			}
		}
	}

	// 3. Head-to-body collisions (mover dies; body owner survives).
	for _, mv := range pending {
		if !mv.p.Alive {
			continue
		}
	outer:
		for _, other := range g.players {
			for si, seg := range other.Points {
				// The mover's own head is where we're coming from; skip it.
				if other == mv.p && si == 0 {
					continue
				}
				if seg == mv.next {
					mv.p.Alive = false
					break outer
				}
			}
		}
	}

	// Collect deaths before moving so callers can notify clients.
	var died []int
	for _, mv := range pending {
		if !mv.p.Alive {
			died = append(died, mv.p.Index)
		}
	}

	// Move surviving snakes and handle food consumption.
	for _, mv := range pending {
		if !mv.p.Alive {
			continue
		}

		ateIdx := -1
		for fi, f := range g.food {
			if f != nil && mv.next.X == f.X && mv.next.Y == f.Y {
				ateIdx = fi
				break
			}
		}

		mv.p.Points = append([]snake.Point{mv.next}, mv.p.Points...)
		if ateIdx >= 0 {
			mv.p.Score += 10
			g.foodCount++
			g.food[ateIdx] = snake.NewFood(g.allSnakePts())
		} else {
			mv.p.Points = mv.p.Points[:len(mv.p.Points)-1]
		}
	}

	// Check win condition: game ends when ≤1 snake is alive.
	alive := 0
	lastAlive := -1
	for _, p := range g.players {
		if p.Alive {
			alive++
			lastAlive = p.Index
		}
	}
	if alive <= 1 {
		g.over = true
		g.winner = lastAlive // -1 if all died simultaneously this tick
	}

	g.render()
	return died
}

// allSnakePts returns every occupied cell across all snakes (for food placement).
func (g *Game) allSnakePts() []snake.Point {
	var pts []snake.Point
	for _, p := range g.players {
		pts = append(pts, p.Points...)
	}
	return pts
}

// render clears the matrix and redraws all alive snakes and food items.
func (g *Game) render() {
	for y := range g.matrix {
		for x := range g.matrix[y] {
			g.matrix[y][x] = 0
		}
	}

	for _, f := range g.food {
		if f != nil {
			g.matrix.Set(*f, 'F')
		}
	}

	for _, p := range g.players {
		if !p.Alive {
			continue
		}
		for i, pt := range p.Points {
			if i == 0 {
				g.matrix.Set(pt, HeadCells[p.Index])
			} else {
				g.matrix.Set(pt, BodyCells[p.Index])
			}
		}
	}
}
