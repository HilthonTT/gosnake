package ai

import (
	"math/rand"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

const (
	// mistakeBase is the starting probability (level 1) that the AI ignores
	// its optimal path and picks a random safe move.
	mistakeBase = 0.06

	// mistakeFloor is the lowest the mistake probability can drop.
	mistakeFloor = 0.015

	// mistakeDecay is subtracted from mistakeBase per level above 1.
	mistakeDecay = 0.005

	// interceptRange is the Manhattan-distance within which the AI considers
	// cutting off the player.
	interceptRange = 14

	// interceptLookAhead is how many steps ahead of the player the AI
	// predicts when computing an intercept target.
	interceptLookAhead = 3

	// aggressionChance is the probability per tick that the AI enters
	// "aggressive mode" — it drops safety checks and goes straight for the
	// intercept or food regardless of dead-end risk. This is what makes the
	// AI killable: aggressive plays can backfire.
	aggressionChance = 0.20

	// safetyMarginNormal is the multiplier applied to body length when the AI
	// is playing safe.
	safetyMarginNormal = 1.2

	// safetyMarginAggressive is used during aggressive plays — much lower,
	// so the AI commits to risky paths.
	safetyMarginAggressive = 0.4

	// tailChaseLimit caps how many consecutive ticks the AI will tail-chase
	// before forcing a food/intercept attempt (even if unsafe). Without this
	// the AI loops its own tail indefinitely and the game stalls.
	tailChaseLimit = 15

	// wallOffRange — when aggressive and this close to the player the AI
	// tries to move toward the player's head directly (body-block) rather
	// than predicting ahead.
	wallOffRange = 6
)

// aiState tracks per-tick mutable decisions so the caller (Game.Tick) can
// persist it across ticks. Currently tracks consecutive tail-chase ticks.
type aiState struct {
	tailChaseTicks int
}

// newAIState returns a zero-valued state for a fresh game.
func newAIState() *aiState {
	return &aiState{}
}

// nextDirection decides where the AI moves next.
//
// The key design goal is that the AI is strong but beatable. It alternates
// between safe play (flood-fill validated paths) and aggressive play (skipping
// safety checks to go for kills). Aggressive plays can trap the AI in dead
// ends, which is how the player wins.
//
// Priority order:
//  1. Random mistake  — level-scaled probability of a random safe move.
//  2. Aggressive play — with aggressionChance probability, skip safety:
//     a. Wall-off     — if very close, move directly toward the player.
//     b. Intercept    — BFS to predicted player position (no flood-fill).
//     c. Raw food     — BFS to food (no flood-fill).
//  3. Safe intercept  — flood-fill validated path to predicted player pos.
//  4. Safe food chase — flood-fill validated path to food.
//  5. Tail chase      — follow own tail (capped by tailChaseLimit).
//  6. Forced commit   — if tail-chasing too long, BFS to food unsafely.
//  7. Largest space   — pick the most open adjacent cell.
//  8. Current dir     — everything is fatal; crash.
func nextDirection(
	head snake.Point,
	food snake.Point,
	playerHead snake.Point,
	playerDir snake.Direction,
	current snake.Direction,
	occupied map[snake.Point]any,
	aiBody []snake.Point,
	level int,
	state *aiState,
) snake.Direction {
	bodyLen := len(aiBody)

	// 1. Level-scaled random mistake.
	chance := mistakeBase - mistakeDecay*float64(level-1)
	if chance < mistakeFloor {
		chance = mistakeFloor
	}
	if rand.Float64() < chance {
		if dir, ok := randomSafeDirection(head, current, occupied); ok {
			state.tailChaseTicks = 0
			return dir
		}
	}

	aggressive := rand.Float64() < aggressionChance
	dist := manhattanDist(head, playerHead)

	// 2. Aggressive play — skip flood-fill safety, commit to risky paths.
	if aggressive {
		// 2a. Wall-off: very close → move directly toward the player's head.
		if dist <= wallOffRange {
			if dir, ok := bfs(head, playerHead, occupied); ok {
				state.tailChaseTicks = 0
				return dir
			}
		}

		// 2b. Intercept without safety check.
		if dist <= interceptRange {
			target := predictPlayerPos(playerHead, playerDir, interceptLookAhead, occupied)
			if target != playerHead {
				if dir, ok := bfs(head, target, occupied); ok {
					state.tailChaseTicks = 0
					return dir
				}
			}
		}

		// 2c. Chase food without safety check.
		if dir, ok := bfs(head, food, occupied); ok {
			state.tailChaseTicks = 0
			return dir
		}
	}

	minSafe := int(float64(bodyLen) * safetyMarginNormal)

	// 3. Safe intercept.
	if dist <= interceptRange {
		target := predictPlayerPos(playerHead, playerDir, interceptLookAhead, occupied)
		if target != playerHead {
			if dir, ok := safeBFS(head, target, occupied, minSafe); ok {
				state.tailChaseTicks = 0
				return dir
			}
		}
	}

	// 4. Safe food chase.
	if dir, ok := safeBFS(head, food, occupied, minSafe); ok {
		state.tailChaseTicks = 0
		return dir
	}

	// 5. Tail chase (capped).
	if state.tailChaseTicks < tailChaseLimit && bodyLen > 1 {
		tail := aiBody[bodyLen-1]
		if dir, ok := bfs(head, tail, occupied); ok {
			state.tailChaseTicks++
			return dir
		}
	}

	// 6. Forced commit — tail-chase limit exceeded or no tail path.
	//    Use a very low safety threshold so the AI actually moves toward food
	//    instead of looping forever.
	minAggressive := int(float64(bodyLen) * safetyMarginAggressive)
	if dir, ok := safeBFS(head, food, occupied, minAggressive); ok {
		state.tailChaseTicks = 0
		return dir
	}
	// Last resort: raw BFS to food, no safety at all.
	if dir, ok := bfs(head, food, occupied); ok {
		state.tailChaseTicks = 0
		return dir
	}

	// 7. Largest reachable area.
	if dir, ok := largestFloodDir(head, current, occupied); ok {
		return dir
	}

	return current
}

// safeBFS finds the shortest path from src to dst and then verifies the first
// step leaves at least minSafe reachable cells (flood-fill).
func safeBFS(src, dst snake.Point, occupied map[snake.Point]any, minSafe int) (snake.Direction, bool) {
	dir, ok := bfs(src, dst, occupied)
	if !ok {
		return 0, false
	}

	next := step(src, dir)
	simOccupied := copyOccupied(occupied)
	simOccupied[src] = struct{}{}

	reachable := floodFill(next, simOccupied)
	if reachable >= minSafe {
		return dir, true
	}

	return 0, false
}

// largestFloodDir evaluates all four cardinal directions and returns the one
// that leads to the largest connected region of free space.
func largestFloodDir(
	head snake.Point,
	current snake.Direction,
	occupied map[snake.Point]any,
) (snake.Direction, bool) {
	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)
	dirs := []snake.Direction{snake.Up, snake.Down, snake.Left, snake.Right}

	bestDir := current
	bestCount := -1
	found := false

	for _, d := range dirs {
		nb := step(head, d)
		if !bounds.InBounds(nb) {
			continue
		}
		if _, blocked := occupied[nb]; blocked {
			continue
		}

		simOccupied := copyOccupied(occupied)
		simOccupied[head] = struct{}{}

		count := floodFill(nb, simOccupied)
		if count > bestCount {
			bestCount = count
			bestDir = d
			found = true
		}
	}

	return bestDir, found
}

// floodFill counts how many cells are reachable from start without crossing
// any point in occupied or leaving the board.
func floodFill(start snake.Point, occupied map[snake.Point]any) int {
	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)

	if !bounds.InBounds(start) {
		return 0
	}
	if _, blocked := occupied[start]; blocked {
		return 0
	}

	visited := make(map[snake.Point]struct{}, 128)
	visited[start] = struct{}{}
	queue := []snake.Point{start}
	count := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		count++

		for _, d := range []snake.Direction{snake.Up, snake.Down, snake.Left, snake.Right} {
			nb := step(cur, d)
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
			queue = append(queue, nb)
		}
	}

	return count
}

// predictPlayerPos walks the player's current direction forward up to n steps,
// stopping at walls or occupied cells.
func predictPlayerPos(
	playerHead snake.Point,
	playerDir snake.Direction,
	n int,
	occupied map[snake.Point]any,
) snake.Point {
	bounds := snake.NewMatrix(snake.DefaultRows, snake.DefaultCols)
	pos := playerHead

	for i := 0; i < n; i++ {
		next := step(pos, playerDir)
		if !bounds.InBounds(next) {
			break
		}
		if _, blocked := occupied[next]; blocked {
			break
		}
		pos = next
	}

	if pos == playerHead {
		return playerHead
	}
	return pos
}

// bfs performs a breadth-first search from src to dst, treating every point
// in occupied as a wall.
func bfs(src, dst snake.Point, occupied map[snake.Point]any) (snake.Direction, bool) {
	type state struct {
		pt       snake.Point
		firstDir snake.Direction
	}

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
func occupiedSet(playerBody, aiBody []snake.Point) map[snake.Point]any {
	set := make(map[snake.Point]any, len(playerBody)+len(aiBody))

	for _, p := range playerBody {
		set[p] = struct{}{}
	}

	for i, p := range aiBody {
		if i == 0 {
			continue
		}
		set[p] = struct{}{}
	}

	return set
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

// randomSafeDirection picks a uniformly random safe direction.
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

// copyOccupied returns a shallow copy of an occupied map.
func copyOccupied(src map[snake.Point]any) map[snake.Point]any {
	dst := make(map[snake.Point]any, len(src)+4)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
