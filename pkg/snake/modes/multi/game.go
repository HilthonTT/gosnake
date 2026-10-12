// Package multi implements the authoritative state for SSH multiplayer games.
package multi

import "github.com/HilthonTT/gosnake/pkg/snake"

const MaxPlayers = 3

// HeadCells and BodyCells map a player index to its matrix byte value.
// Exported so the TUI layer can build its render switch without magic literals.
var HeadCells = [MaxPlayers]byte{'H', 'A', 'X'}
var BodyCells = [MaxPlayers]byte{'S', 'B', 'Y'}

// PlayerSnake is one player's snake state.
type PlayerSnake struct {
	Index    int
	Name     string
	Points   []snake.Point
	Alive    bool
	Score    int
	steering snake.Steering
}

// Direction returns the direction the snake moved on the last tick.
func (p *PlayerSnake) Direction() snake.Direction { return p.steering.Current() }

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
// At most MaxPlayers names are used.
func NewGame(names []string) *Game {
	n := min(len(names), MaxPlayers)

	// Fixed spread so snakes start far apart and face inward.
	starts := [MaxPlayers]snake.Point{
		{X: snake.DefaultCols / 4, Y: snake.DefaultRows / 2},
		{X: 3 * snake.DefaultCols / 4, Y: snake.DefaultRows / 2},
		{X: snake.DefaultCols / 2, Y: snake.DefaultRows / 4},
	}
	startDirs := [MaxPlayers]snake.Direction{snake.Right, snake.Left, snake.Down}

	players := make([]*PlayerSnake, n)
	var allPts []snake.Point

	for i := range n {
		players[i] = &PlayerSnake{
			Index:    i,
			Name:     names[i],
			Points:   []snake.Point{starts[i]},
			Alive:    true,
			steering: snake.NewSteering(startDirs[i]),
		}
		allPts = append(allPts, starts[i])
	}

	// One food item per player, placed away from all snakes.
	foods := make([]*snake.Point, n)
	for i := range foods {
		foods[i] = snake.NewFood(allPts)
		if foods[i] != nil {
			allPts = append(allPts, *foods[i])
		}
	}

	g := &Game{
		matrix:  snake.NewMatrix(snake.DefaultRows, snake.DefaultCols),
		players: players,
		food:    foods,
		winner:  -1,
	}
	g.render()

	return g
}

// ChangeDirection queues a direction change for a player, rejecting 180° reversals.
// Must only be called from the goroutine that owns the game.
func (g *Game) ChangeDirection(playerIndex int, d snake.Direction) {
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return
	}
	if p := g.players[playerIndex]; p.Alive {
		p.steering.Queue(d)
	}
}

// Tick advances every alive snake by one step and resolves all collisions
// simultaneously. It returns the indices of players that died this tick.
func (g *Game) Tick() []int {
	if g.over {
		return nil
	}

	type move struct {
		p     *PlayerSnake
		next  snake.Point
		body  []snake.Point // body after the move
		food  int           // index of the food eaten, or -1
		alive bool
	}

	// 1. Compute every intended move and the resulting body.
	moves := make([]*move, 0, len(g.players))
	for _, p := range g.players {
		if !p.Alive {
			continue
		}
		next := p.Points[0].Step(p.steering.Advance())
		eaten := g.foodAt(next)
		moves = append(moves, &move{
			p:     p,
			next:  next,
			body:  snake.Advance(p.Points, next, eaten >= 0),
			food:  eaten,
			alive: g.matrix.InBounds(next),
		})
	}

	// 2. Head-to-head collisions (same target cell, or swapping cells) kill both.
	for i, a := range moves {
		for _, b := range moves[i+1:] {
			swapped := a.next == b.p.Points[0] && b.next == a.p.Points[0]
			if a.next == b.next || swapped {
				a.alive, b.alive = false, false
			}
		}
	}

	// 3. Head-to-body collisions against the post-move board. Dead snakes are
	//    removed from the board, so only moving snakes are obstacles.
	for _, mv := range moves {
		if !mv.alive {
			continue
		}
		for _, other := range moves {
			body := other.body
			if other == mv {
				body = body[1:] // skip our own new head
			}
			if snake.HitsBody(body, mv.next, false) {
				mv.alive = false
				break
			}
		}
	}

	// 4. Apply results.
	var died []int
	for _, mv := range moves {
		if !mv.alive {
			mv.p.Alive = false
			died = append(died, mv.p.Index)
			continue
		}
		mv.p.Points = mv.body
		if mv.food >= 0 {
			mv.p.Score += snake.PointsPerFood
			g.foodCount++
			g.food[mv.food] = nil
		}
	}
	for i, f := range g.food {
		if f == nil {
			g.food[i] = snake.NewFood(g.occupied())
		}
	}

	// Game ends when ≤1 snake is alive.
	alive, lastAlive := 0, -1
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

// foodAt returns the index of the food item at p, or -1.
func (g *Game) foodAt(p snake.Point) int {
	for i, f := range g.food {
		if f != nil && *f == p {
			return i
		}
	}
	return -1
}

// occupied returns every cell food must avoid: live snakes and other food.
func (g *Game) occupied() []snake.Point {
	var pts []snake.Point
	for _, p := range g.players {
		if p.Alive {
			pts = append(pts, p.Points...)
		}
	}
	for _, f := range g.food {
		if f != nil {
			pts = append(pts, *f)
		}
	}
	return pts
}

// render clears the matrix and redraws all alive snakes and food items.
func (g *Game) render() {
	g.matrix.Clear()

	for _, f := range g.food {
		if f != nil {
			g.matrix.Set(*f, snake.CellFood)
		}
	}

	for _, p := range g.players {
		if p.Alive {
			g.matrix.DrawSnake(p.Points, HeadCells[p.Index], BodyCells[p.Index])
		}
	}
}

// Level rises by one every five pieces of food eaten, capped at snake.MaxLevel.
func (g *Game) Level() int {
	return min(1+g.foodCount/5, snake.MaxLevel)
}

func (g *Game) IsOver() bool            { return g.over }
func (g *Game) Winner() int             { return g.winner }
func (g *Game) Matrix() snake.Matrix    { return g.matrix }
func (g *Game) Players() []*PlayerSnake { return g.players }
func (g *Game) FoodCount() int          { return g.foodCount }

func (g *Game) Snapshot() map[string]any {
	playerSnaps := make([]map[string]any, len(g.players))
	for i, p := range g.players {
		ps := map[string]any{
			"index":     p.Index,
			"name":      p.Name,
			"score":     p.Score,
			"direction": p.Direction().String(),
			"alive":     p.Alive,
			"length":    len(p.Points),
		}
		if len(p.Points) > 0 {
			ps["head"] = p.Points[0]
		}
		playerSnaps[i] = ps
	}

	foodPts := make([]snake.Point, 0, len(g.food))
	for _, f := range g.food {
		if f != nil {
			foodPts = append(foodPts, *f)
		}
	}

	return map[string]any{
		"players":   playerSnaps,
		"food":      foodPts,
		"foodCount": g.foodCount,
		"level":     g.Level(),
		"over":      g.over,
		"winner":    g.winner,
	}
}
