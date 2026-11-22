package main

import (
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

// Board colours the autopilot reads back from the rendered frame. They must
// match internal/tui/views/single.go.
var (
	colFloorA = hex("#101714")
	colFloorB = hex("#131C18")
	colWall   = hex("#26332D")
	colHead   = hex("#BBF7D0")
	colFood   = hex("#FBBF24")
	colAIHead = hex("#DDD6FE")
	colBomb   = hex("#F43F5E")
	colFuse   = hex("#881337")
)

type cellPos struct{ r, c int }

// autopilot plays the game by reading the board from the latest frame:
// BFS to the food, checking it keeps enough room to survive.
type autopilot struct {
	rt       *runtime
	eaten    int
	lastHead cellPos
	lastFood cellPos
	length   int
	bombs    map[cellPos]time.Time
	dir      snake.Direction
	started  bool
}

type board struct {
	cells      [][]byte // '.' free, '#' blocked, 'H' head, 'F' food
	head, food cellPos
	hasHead    bool
	hasFood    bool
}

func (a *autopilot) read() (board, bool) {
	g := a.rt.grid
	x0, y0, ok := -1, -1, false
	for y := 0; y < g.h && !ok; y++ {
		for x := 0; x < g.w; x++ {
			if sameRGB(g.cells[y][x].bg, colWall) {
				x0, y0, ok = x, y, true
				break
			}
		}
	}
	if !ok {
		return board{}, false
	}

	if a.bombs == nil {
		a.bombs = map[cellPos]time.Time{}
	}
	now := time.Now()
	b := board{cells: make([][]byte, snake.DefaultRows)}
	for r := range snake.DefaultRows {
		b.cells[r] = make([]byte, snake.DefaultCols)
		for c := range snake.DefaultCols {
			x, y := x0+2*(c+1), y0+r+1
			if y >= g.h || x >= g.w {
				return board{}, false
			}
			bg := g.cells[y][x].bg
			p := cellPos{r, c}
			switch {
			case sameRGB(bg, colHead):
				b.cells[r][c], b.head, b.hasHead = 'H', p, true
			case sameRGB(bg, colFood):
				b.cells[r][c], b.food, b.hasFood = 'F', p, true
			case sameRGB(bg, colFloorA), sameRGB(bg, colFloorB):
				b.cells[r][c] = '.'
			default:
				b.cells[r][c] = '#'
				if sameRGB(bg, colBomb) || sameRGB(bg, colFuse) {
					a.bombs[p] = now
				}
			}
		}
	}

	// Warning bombs blink, so remember them for a while after they vanish,
	// and keep away from the AI's head to avoid head-on collisions.
	for p, seen := range a.bombs {
		if now.Sub(seen) > 3*time.Second {
			delete(a.bombs, p)
		} else if b.cells[p.r][p.c] == '.' {
			b.cells[p.r][p.c] = '#'
		}
	}
	for r := range b.cells {
		for c := range b.cells[r] {
			x, y := x0+2*(c+1), y0+r+1
			if sameRGB(g.cells[y][x].bg, colAIHead) {
				for _, d := range snake.Directions {
					n := step(cellPos{r, c}, d)
					if inBoard(n) && b.cells[n.r][n.c] == '.' {
						b.cells[n.r][n.c] = '#'
					}
				}
			}
		}
	}
	return b, true
}

func (a *autopilot) over() bool {
	text := a.rt.grid.text()
	return strings.Contains(text, "GAME OVER") || strings.Contains(text, "YOU WIN") || strings.Contains(text, "DRAW")
}

// play steers until done reports true, the game ends or max passes.
func (a *autopilot) play(done func() bool, max time.Duration) {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) && !done() && !a.over() {
		a.rt.next(50 * time.Millisecond)
		b, ok := a.read()
		if !ok || !b.hasHead {
			continue
		}
		if a.started && b.head == a.lastHead {
			continue
		}
		if a.started && b.hasFood && b.food != a.lastFood {
			a.eaten++
		}
		a.started = true
		a.lastHead, a.lastFood = b.head, b.food
		a.length = a.eaten + 1

		d := a.decide(b)
		a.dir = d
		a.rt.key(dirKey(d))
	}
}

// crash steers into the nearest wall and waits for the game to end.
func (a *autopilot) crash(max time.Duration) {
	if b, ok := a.read(); ok && b.hasHead {
		h := b.head
		best, dist := snake.Up, h.r
		for d, n := range map[snake.Direction]int{
			snake.Down:  snake.DefaultRows - 1 - h.r,
			snake.Left:  h.c,
			snake.Right: snake.DefaultCols - 1 - h.c,
		} {
			if n < dist && d != a.dir.Opposite() {
				best, dist = d, n
			}
		}
		if best == a.dir.Opposite() {
			best = snake.Left
		}
		a.rt.key(dirKey(best))
	}
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) && !a.over() {
		a.rt.next(50 * time.Millisecond)
	}
}

func (a *autopilot) decide(b board) snake.Direction {
	need := a.length + 4

	if b.hasFood {
		if d, ok := bfs(b, b.head, b.food); ok {
			n := step(b.head, d)
			if area(b, n) >= need {
				return d
			}
		}
	}

	// No safe path to the food: take the roomiest move.
	best, bestArea := a.dir, -1
	for _, d := range snake.Directions {
		n := step(b.head, d)
		if !free(b, n) {
			continue
		}
		if ar := area(b, n); ar > bestArea {
			best, bestArea = d, ar
		}
	}
	return best
}

func bfs(b board, from, to cellPos) (snake.Direction, bool) {
	first := map[cellPos]snake.Direction{}
	queue := []cellPos{}
	for _, d := range snake.Directions {
		n := step(from, d)
		if free(b, n) {
			first[n] = d
			queue = append(queue, n)
		}
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == to {
			return first[p], true
		}
		for _, d := range snake.Directions {
			n := step(p, d)
			if _, seen := first[n]; seen || !free(b, n) {
				continue
			}
			first[n] = first[p]
			queue = append(queue, n)
		}
	}
	return 0, false
}

// area counts the free cells reachable from p.
func area(b board, p cellPos) int {
	seen := map[cellPos]bool{p: true}
	stack := []cellPos{p}
	for len(stack) > 0 && len(seen) < 400 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range snake.Directions {
			n := step(q, d)
			if !seen[n] && free(b, n) {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	return len(seen)
}

func free(b board, p cellPos) bool {
	return inBoard(p) && (b.cells[p.r][p.c] == '.' || b.cells[p.r][p.c] == 'F')
}

func inBoard(p cellPos) bool {
	return p.r >= 0 && p.r < snake.DefaultRows && p.c >= 0 && p.c < snake.DefaultCols
}

func step(p cellPos, d snake.Direction) cellPos {
	q := snake.Point{X: p.c, Y: p.r}.Step(d)
	return cellPos{q.Y, q.X}
}

func dirKey(d snake.Direction) tea.KeyPressMsg {
	switch d {
	case snake.Up:
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case snake.Down:
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case snake.Left:
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	default:
		return tea.KeyPressMsg{Code: tea.KeyRight}
	}
}

func sameRGB(a, b color.RGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B
}
