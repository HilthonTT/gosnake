package snake

import (
	"math/rand"
)

type Point struct {
	X int
	Y int
}

func NewFood(snake []Point) *Point {
	p := &Point{}

	for {
		p.X = rand.Intn(DefaultCols)
		p.Y = rand.Intn(DefaultRows)
		if !hasExistingPoint(snake, p) {
			break
		}
	}

	return p
}

func hasExistingPoint(snake []Point, point *Point) bool {
	if point == nil {
		return false
	}
	for _, p := range snake {
		if p.X == point.X && p.Y == point.Y {
			return true
		}
	}
	return false
}
