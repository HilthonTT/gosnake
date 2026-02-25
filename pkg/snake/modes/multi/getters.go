package multi

import "github.com/HilthonTT/gosnake/pkg/snake"

func (g *Game) Level() int {
	l := 1 + g.foodCount/5
	if l > 10 {
		return 10
	}
	return l
}

func (g *Game) IsOver() bool {
	return g.over
}

func (g *Game) Winner() int {
	return g.winner
}

func (g *Game) Matrix() snake.Matrix {
	return g.matrix
}

func (g *Game) Players() []*PlayerSnake {
	return g.players
}

func (g *Game) FoodCount() int {
	return g.foodCount
}
