// Package theme is the single palette and set of shared styles used by every
// TUI screen, so the menu, game and leaderboard look like one app.
package theme

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// Palette. Plain text is left unstyled so it follows the terminal's own
// foreground on both light and dark backgrounds.
var (
	Accent    = lipgloss.Color("#A3E635") // lime — brand colour, player snake
	AccentAlt = lipgloss.Color("#2DD4BF") // teal — end stop of the logo gradient
	AccentDim = lipgloss.Color("#4D7C0F") // deep green
	Muted     = lipgloss.Color("#9CA3AF") // secondary text
	Subtle    = lipgloss.Color("#6B7280") // hints, separators
	Border    = lipgloss.Color("#3F4A3F") // box borders
	Highlight = lipgloss.Color("#26361C") // selected row background
	Gold      = lipgloss.Color("#FACC15")
	Silver    = lipgloss.Color("#CBD5E1")
	Bronze    = lipgloss.Color("#D97706")
	Danger    = lipgloss.Color("#F43F5E")
	Warning   = lipgloss.Color("#F59E0B")
	Info      = lipgloss.Color("#38BDF8")
	OnAccent  = lipgloss.Color("#0B1206") // text drawn on an Accent background
)

// Panel frames a block with horizontal rules only. Vertical box-drawing
// lines render dashed in terminals that draw them from the font (e.g. VS
// Code without GPU acceleration), so panels avoid them.
var Panel = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), true, false).
	BorderForeground(Border)

// Shared styles.
var (
	Title    = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	Label    = lipgloss.NewStyle().Foreground(Muted)
	Hint     = lipgloss.NewStyle().Foreground(Subtle)
	Value    = lipgloss.NewStyle().Bold(true)
	Score    = lipgloss.NewStyle().Foreground(Gold).Bold(true)
	Key      = lipgloss.NewStyle().Foreground(Accent)
	Error    = lipgloss.NewStyle().Foreground(Danger).Bold(true)
	Badge    = lipgloss.NewStyle().Foreground(OnAccent).Background(Accent).Bold(true).Padding(0, 1)
	Card     = Panel.Padding(1, 2)
	Subtitle = lipgloss.NewStyle().Foreground(AccentDim).Italic(true)
)

// NewHelp returns a help model styled to match the palette.
func NewHelp() help.Model {
	h := help.New()
	sep := lipgloss.NewStyle().Foreground(Border)
	h.Styles = help.Styles{
		ShortKey:       Key,
		ShortDesc:      Hint,
		ShortSeparator: sep,
		FullKey:        Key,
		FullDesc:       Hint,
		FullSeparator:  sep,
		Ellipsis:       sep,
	}
	h.ShortSeparator = " • "
	return h
}

// logoPixels spells GOSNAKE in a 4×5 pixel font. Each '#' is drawn as a
// background-coloured cell rather than a block glyph, so it renders solid in
// every terminal regardless of font or line height.
var logoPixels = []string{
	".###  .##.  .###  #..#  .##.  #..#  ####",
	"#...  #..#  #...  ##.#  #..#  #.#.  #...",
	"#.##  #..#  .##.  #.##  ####  ##..  ###.",
	"#..#  #..#  ...#  #..#  #..#  #.#.  #...",
	".###  .##.  ###.  #..#  #..#  #..#  ####",
}

var (
	logoLarge = pixelArt(logoPixels)
	logoSmall = gradient("G O S N A K E")
)

// LogoWidth is the width of the large logo in columns.
var LogoWidth = lipgloss.Width(logoLarge)

// Logo returns the gradient logo, or a one-line wordmark when small is set.
func Logo(small bool) string {
	if small {
		return logoSmall
	}
	return logoLarge
}

// pixelArt draws each '#' as a two-column cell along the Accent → AccentAlt
// gradient; every other rune becomes blank space.
func pixelArt(rows []string) string {
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}
	ramp := lipgloss.Blend1D(max(width, 2), Accent, AccentAlt)

	lines := make([]string, len(rows))
	for i, row := range rows {
		var sb strings.Builder
		for col, r := range row {
			if r == '#' {
				sb.WriteString(lipgloss.NewStyle().Background(ramp[col]).Render("  "))
			} else {
				sb.WriteString("  ")
			}
		}
		lines[i] = sb.String()
	}
	return strings.Join(lines, "\n")
}

// gradient colours each column of s along the Accent → AccentAlt gradient.
func gradient(s string) string {
	ramp := lipgloss.Blend1D(max(lipgloss.Width(s), 2), Accent, AccentAlt)

	var sb strings.Builder
	for col, r := range []rune(s) {
		if r == ' ' {
			sb.WriteRune(r)
			continue
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(ramp[col]).Bold(true).Render(string(r)))
	}
	return sb.String()
}

// SpaceBetween lays out left and right at opposite ends of width columns.
func SpaceBetween(width int, left, right string) string {
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

// ProgressBar renders a filled/empty bar of width cells for ratio in [0,1].
func ProgressBar(width int, ratio float64) string {
	ratio = min(max(ratio, 0), 1)
	filled := int(ratio*float64(width) + 0.5)
	return lipgloss.NewStyle().Foreground(Accent).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(Border).Render(strings.Repeat("━", width-filled))
}

// TooSmall is shown in place of a screen that does not fit the terminal.
func TooSmall(width, height, needW, needH int) string {
	msg := lipgloss.JoinVertical(lipgloss.Center,
		Error.Render("Terminal too small"),
		"",
		Label.Render("Resize to at least"),
		Value.Render(sizeStr(needW, needH)),
		Hint.Render("currently "+sizeStr(width, height)),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
}

func sizeStr(w, h int) string {
	return strconv.Itoa(w) + " × " + strconv.Itoa(h)
}
