package ai

import (
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
)

func (g *Game) Matrix() snake.Matrix                  { return g.matrix }
func (g *Game) IsGameOver() bool                      { return g.gameOver }
func (g *Game) IsPaused() bool                        { return g.paused }
func (g *Game) Score() int                            { return g.playerScore.Total() }
func (g *Game) Level() int                            { return g.playerScore.Level() }
func (g *Game) Snake() []snake.Point                  { return g.playerBody }
func (g *Game) SnakeLength() int                      { return len(g.playerBody) }
func (g *Game) Food() *snake.Point                    { return g.food }
func (g *Game) GetTickInterval() time.Duration        { return snake.GetTickInterval(g.playerScore.Level()) }
func (g *Game) GetDefaultTickInterval() time.Duration { return snake.GetTickInterval(1) }

func (g *Game) AIScore() int        { return g.aiScore.Total() }
func (g *Game) AISnakeLength() int  { return len(g.aiBody) }
func (g *Game) IsAIAlive() bool     { return g.aiAlive }
func (g *Game) IsPlayerAlive() bool { return g.playerAlive }
