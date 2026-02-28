package components

import "github.com/charmbracelet/lipgloss"

// Palette — xterm-256 colour constants used throughout the game.
const (
	colSnakeHead   = lipgloss.Color("82")  // bright lime green
	colSnakeBody   = lipgloss.Color("40")  // medium green
	colFood        = lipgloss.Color("196") // bright red
	colEmpty       = lipgloss.Color("236") // very dark grey
	colBorder      = lipgloss.Color("241") // mid grey
	colAccent      = lipgloss.Color("82")  // matches head — used for titles
	colMuted       = lipgloss.Color("243") // dimmed text
	colScoreValue  = lipgloss.Color("220") // warm yellow for numbers
	colPauseText   = lipgloss.Color("214") // amber for paused state
	colGameOver    = lipgloss.Color("196") // red for game-over state
	colBomb        = lipgloss.Color("196") // bright red  — active/lethal bomb
	colBombWarning = lipgloss.Color("214") // amber       — blinking pre-warning
)

// CellCharacters holds the two-rune wide strings used for each cell type.
// Every entry must be exactly two terminal columns wide so the grid aligns.
type CellCharacters struct {
	Empty       string // empty cell
	Head        string // snake head
	Body        string // snake body
	Food        string // food pellet
	Bomb        string // active (lethal) bomb
	BombWarning string // warning (blinking, not yet lethal) bomb
}

// InfoStyles groups all styles used in the side information panel.
type InfoStyles struct {
	Panel      lipgloss.Style
	Title      lipgloss.Style
	SectionLbl lipgloss.Style
	ValueBig   lipgloss.Style
	Divider    lipgloss.Style
}

// OverlayStyles groups styles for the pause / game-over overlay text.
type OverlayStyles struct {
	Paused   lipgloss.Style
	GameOver lipgloss.Style
}

// GameStyles is the single source of truth for all visual styling.
type GameStyles struct {
	Board           lipgloss.Style
	EmptyCell       lipgloss.Style
	HeadCell        lipgloss.Style
	BodyCell        lipgloss.Style
	FoodCell        lipgloss.Style
	BombCell        lipgloss.Style // active bomb — always visible, lethal
	BombWarningCell lipgloss.Style // warning bomb — rendered on blink "on" frames
	Info            InfoStyles
	Overlay         OverlayStyles
	CellChars       CellCharacters
	Help            lipgloss.Style
}

// CreateGameStyles returns a fully populated GameStyles with the default theme.
func CreateGameStyles() *GameStyles {
	panelWidth := 16

	return &GameStyles{
		// Board
		Board: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0),

		// Cell styles
		EmptyCell: lipgloss.NewStyle().Foreground(colEmpty),
		HeadCell:  lipgloss.NewStyle().Foreground(colSnakeHead).Bold(true),
		BodyCell:  lipgloss.NewStyle().Foreground(colSnakeBody),
		FoodCell:  lipgloss.NewStyle().Foreground(colFood).Bold(true),

		// Active bomb — same red as food but bold + background tint to stand out.
		BombCell: lipgloss.NewStyle().
			Foreground(colBomb).
			Background(lipgloss.Color("52")). // dark red background
			Bold(true),

		// Warning bomb — amber, no background so it reads differently from the
		// live bomb even on the "on" half of the blink cycle.
		BombWarningCell: lipgloss.NewStyle().
			Foreground(colBombWarning).
			Bold(true),

		// Info Panel
		Info: InfoStyles{
			Panel: lipgloss.NewStyle().
				Width(panelWidth).
				Padding(1, 1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colBorder),

			Title: lipgloss.NewStyle().
				Width(panelWidth - 2). // subtract padding
				Align(lipgloss.Center).
				Bold(true).
				Foreground(colAccent),

			SectionLbl: lipgloss.NewStyle().
				Foreground(colMuted).
				Width(panelWidth - 2),

			ValueBig: lipgloss.NewStyle().
				Width(panelWidth - 2).
				Align(lipgloss.Right).
				Bold(true).
				Foreground(colScoreValue),

			Divider: lipgloss.NewStyle().
				Foreground(colBorder).
				Width(panelWidth - 2),
		},

		// Overlays
		Overlay: OverlayStyles{
			Paused: lipgloss.NewStyle().
				Foreground(colPauseText).
				Bold(true).
				Padding(0, 2),

			GameOver: lipgloss.NewStyle().
				Foreground(colGameOver).
				Bold(true).
				Padding(0, 2),
		},

		// Help bar
		Help: lipgloss.NewStyle().Foreground(colMuted),

		// Characters
		CellChars: CellCharacters{
			Empty:       "· ",
			Head:        "██",
			Body:        "▓▓",
			Food:        "◆ ",
			Bomb:        "💣",  // two columns wide in most terminals
			BombWarning: "⚠ ", // warning sign + space = two columns
		},
	}
}
