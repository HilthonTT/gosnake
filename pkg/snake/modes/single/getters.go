package single

import "github.com/HilthonTT/gosnake/pkg/snake"

func (g *Game) Matrix() snake.Matrix { return g.matrix }
func (g *Game) IsGameOver() bool     { return g.gameOver }
func (g *Game) IsPaused() bool       { return g.paused }
func (g *Game) Score() int           { return g.scoring.Total() }
func (g *Game) Level() int           { return g.scoring.Level() }
func (g *Game) Snake() []snake.Point { return g.snake }
func (g *Game) Food() *snake.Point   { return g.food }
