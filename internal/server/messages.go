package server

import "github.com/HilthonTT/gosnake/pkg/snake"

// NoteMsg is a plain-text notification broadcast to all players in a room.
type NoteMsg string

// DirectionMsg is sent by a player's SharedMultiGame to the room's sync
// channel to queue a direction change.
type DirectionMsg struct {
	PlayerIndex int
	Direction   int // maps to multi.Direction; avoids circular import
}

// PlayerSnapshot is an immutable copy of one player's state for broadcast.
// Using a value type (not a pointer) makes the GameStateMsg fully self-contained.
type PlayerSnapshot struct {
	Index  int
	Name   string
	Score  int
	Alive  bool
	Length int
}

// GameStateMsg is broadcast by the room to every connected session after each
// game tick. It carries a deep-copied matrix and player snapshots so each
// player's Bubble Tea program can render independently without touching shared
// mutable state.
type GameStateMsg struct {
	Matrix  snake.Matrix
	Players []PlayerSnapshot
	Level   int
	Over    bool
	Winner  int   // -1 = draw, 0-2 = winning player index
	Died    []int // player indices that died this tick (for death overlay)
}

// RestartMsg is broadcast when any player triggers a restart.
type RestartMsg struct{}
