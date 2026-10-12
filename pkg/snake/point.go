package snake

import (
	"math/rand/v2"
	"slices"
)

type Point struct {
	X int
	Y int
}

// Step returns the point one cell away from p in direction d.
func (p Point) Step(d Direction) Point {
	switch d {
	case Up:
		p.Y--
	case Down:
		p.Y++
	case Left:
		p.X--
	case Right:
		p.X++
	}
	return p
}

// InBoard reports whether p lies on the default-sized board.
func InBoard(p Point) bool {
	return p.X >= 0 && p.X < DefaultCols && p.Y >= 0 && p.Y < DefaultRows
}

// NewFood returns a random free cell on the default board, or nil when every
// cell is occupied.
func NewFood(occupied []Point) *Point {
	p, ok := RandomFreePoint(occupied)
	if !ok {
		return nil
	}
	return &p
}

// RandomFreePoint picks a uniformly random cell on the default board that is
// not in occupied. ok is false when the board is full.
func RandomFreePoint(occupied []Point) (p Point, ok bool) {
	taken := make(map[Point]struct{}, len(occupied))
	for _, o := range occupied {
		if InBoard(o) {
			taken[o] = struct{}{}
		}
	}

	free := DefaultCols*DefaultRows - len(taken)
	if free <= 0 {
		return Point{}, false
	}

	// Fast path: rejection sampling works well while the board is mostly empty.
	if free > (DefaultCols*DefaultRows)/4 {
		for {
			p = Point{X: rand.IntN(DefaultCols), Y: rand.IntN(DefaultRows)}
			if _, hit := taken[p]; !hit {
				return p, true
			}
		}
	}

	// Slow path: pick the n-th free cell so a crowded board never spins.
	n := rand.IntN(free)
	for y := range DefaultRows {
		for x := range DefaultCols {
			p = Point{X: x, Y: y}
			if _, hit := taken[p]; hit {
				continue
			}
			if n == 0 {
				return p, true
			}
			n--
		}
	}
	return Point{}, false
}

// Advance returns body moved one step so that next becomes the new head. When
// grow is false the tail cell is dropped, keeping the length constant.
func Advance(body []Point, next Point, grow bool) []Point {
	keep := len(body)
	if !grow && keep > 0 {
		keep--
	}
	moved := make([]Point, 0, keep+1)
	moved = append(moved, next)
	return append(moved, body[:keep]...)
}

// HitsBody reports whether stepping onto p collides with body. When tailMoves
// is true the last segment is ignored, because it vacates its cell on the same
// tick the head arrives — so a snake may safely chase its own tail.
func HitsBody(body []Point, p Point, tailMoves bool) bool {
	if tailMoves && len(body) > 0 {
		body = body[:len(body)-1]
	}
	return slices.Contains(body, p)
}
