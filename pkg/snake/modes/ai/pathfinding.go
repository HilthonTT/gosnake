package ai

import (
	"math/rand"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

const (
	// mistakeChance is the probability (0.0–1.0) that the AI ignores its
	// optimal path and picks any random safe move instead. Raise this to make
	// the AI easier; lower it to make it harder.
	mistakeChance = 0.05

	// interceptRange is the Manhattan-distance threshold within which the AI
	// switches from chasing food to trying to intercept the player's head.
	// When the player is this many cells away or closer, the AI targets the
	// cell ahead of the player rather than the food.
	interceptRange = 12
)

// nextDirection decides where the AI moves next.
//
// Priority order:
//  1. Random mistake  — with mistakeChance probability, take any safe move.
//  2. Intercept       — if the player is within interceptRange, aim for the
//     cell ahead of the player's head to cut them off.
//  3. Chase food      — BFS shortest path to the food.
//  4. Survival        — any safe adjacent cell if none of the above work.
//  5. Current dir     — every move is fatal; let the game handle the crash.
func nextDirection(
	head snake.Point,
	food snake.Point,
	playerHead snake.Point,
	playerDir snake.Direction,
	current snake.Direction,
	occupied map[snake.Point]any,
) snake.Direction {
	// Make the AI do a mistake
	if rand.Float64() < mistakeChance {
		if dir, ok := randomSafeDirection(head, current, occupied); ok {
			return dir
		}
	}

	if manhattanDist(head, playerHead) <= interceptRange {
		target := interceptTarget(playerHead, playerDir)
		if target != playerHead { // valid intercept point (in bounds)
			if dir, ok := bfs(head, target, occupied); ok {
				return dir
			}
		}
	}

	// Chase the food
	if dir, ok := bfs(head, food, occupied); ok {
		return dir
	}

	// Survival fallback
	if dir, ok := randomSafeDirection(head, current, occupied); ok {
		return dir
	}

	return current // every move is fatal; let the game handle it
}

// bfs performs a breadth-first search from src to dst, treating every point
// in occupied as a wall. It returns the first direction to take and true if a
// path was found, or the zero direction and false if no path exists.
func bfs(src, dst snake.Point, occupied map[snake.Point]any) (snake.Direction, bool) {
	type state struct {
		pt       snake.Point
		firstDir snake.Direction
	}

	// Inline boundary checker using a throwaway matrix.
	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)

	visited := map[snake.Point]struct{}{src: {}}
	queue := []state{}

	for _, dir := range []snake.Direction{snake.Up, snake.Down, snake.Left, snake.Right} {
		nb := step(src, dir)
		if !bounds.InBounds(nb) {
			continue
		}
		if _, blocked := occupied[nb]; blocked {
			continue
		}

		visited[nb] = struct{}{}
		queue = append(queue, state{pt: nb, firstDir: dir})
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.pt == dst {
			return cur.firstDir, true
		}

		for _, dir := range []snake.Direction{snake.Up, snake.Down, snake.Left, snake.Right} {
			nb := step(cur.pt, dir)
			if !bounds.InBounds(nb) {
				continue
			}
			if _, seen := visited[nb]; seen {
				continue
			}
			if _, blocked := occupied[nb]; blocked {
				continue
			}
			visited[nb] = struct{}{}
			queue = append(queue, state{pt: nb, firstDir: cur.firstDir})
		}
	}

	return 0, false
}

// step returns the point one cell in direction d from p.
func step(p snake.Point, d snake.Direction) snake.Point {
	switch d {
	case snake.Up:
		return snake.Point{X: p.X, Y: p.Y - 1}
	case snake.Down:
		return snake.Point{X: p.X, Y: p.Y + 1}
	case snake.Left:
		return snake.Point{X: p.X - 1, Y: p.Y}
	default: // Right
		return snake.Point{X: p.X + 1, Y: p.Y}
	}
}

// occupiedSet builds the map used by the pathfinder from raw point slices.
// It excludes the AI head itself so the search can start from there, and
// excludes the tail tips because those cells will be vacated next tick.
func occupiedSet(playerBody, aiBody []snake.Point) map[snake.Point]any {
	set := make(map[snake.Point]any, len(playerBody)+len(aiBody))

	for _, p := range playerBody {
		set[p] = struct{}{}
	}

	// Skip the AI head (index 0) — that's our starting position.
	for i, p := range aiBody {
		if i == 0 {
			continue
		}
		set[p] = struct{}{}
	}

	return set
}

// interceptTarget returns the cell one step ahead of the player's current
// direction — the spot the AI wants to reach before the player does.
// If that cell is out of bounds it returns playerHead as a sentinel so the
// caller knows to skip the intercept.
func interceptTarget(playerHead snake.Point, playerDir snake.Direction) snake.Point {
	target := step(playerHead, playerDir)
	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)
	if !bounds.InBounds(target) {
		return playerHead
	}
	return target
}

// manhattanDist returns the Manhattan distance between two points.
func manhattanDist(a, b snake.Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// randomSafeDirection picks a uniformly random safe direction from the four
// cardinals, excluding out-of-bounds and blocked cells.
func randomSafeDirection(
	head snake.Point,
	current snake.Direction,
	occupied map[snake.Point]any,
) (snake.Direction, bool) {
	dirs := []snake.Direction{snake.Up, snake.Down, snake.Left, snake.Right}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)
	for _, d := range dirs {
		nb := step(head, d)
		if !bounds.InBounds(nb) {
			continue
		}
		if _, blocked := occupied[nb]; blocked {
			continue
		}
		return d, true
	}
	return current, false
}
