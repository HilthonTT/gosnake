// Package theme is the single palette and set of shared styles used by every
// TUI screen, so the menu, game and leaderboard look like one app.
package theme

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// Palette. Plain text is left unstyled so it follows the terminal's own
// foreground on both light and dark backgrounds.
var (
	Accent    = lipgloss.Color("#4ADE80") // emerald — brand colour, player snake
	AccentAlt = lipgloss.Color("#22D3EE") // cyan — end stop of the logo gradient
	AccentDim = lipgloss.Color("#15803D") // deep green
	Muted     = lipgloss.Color("#A1A1AA") // secondary text
	Subtle    = lipgloss.Color("#71717A") // hints, separators
	Border    = lipgloss.Color("#3F3F46") // rules and empty meter cells
	Highlight = lipgloss.Color("#1E3A2B") // selected row background
	Gold      = lipgloss.Color("#FBBF24")
	Silver    = lipgloss.Color("#CBD5E1")
	Bronze    = lipgloss.Color("#D97706")
	Danger    = lipgloss.Color("#FB7185")
	Warning   = lipgloss.Color("#FB923C")
	Info      = lipgloss.Color("#38BDF8")
	Rival     = lipgloss.Color("#A78BFA") // violet — the AI opponent
	OnAccent  = lipgloss.Color("#052E16") // text drawn on an Accent background
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
	Subtitle = lipgloss.NewStyle().Foreground(Subtle).Italic(true)
	Heading  = lipgloss.NewStyle().Foreground(Muted).Bold(true)
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
	h.ShortSeparator = " · "
	return h
}

// Section renders a small-caps heading followed by a rule, filling width
// columns: "STATS ──────────".
func Section(title string, width int, c color.Color) string {
	head := lipgloss.NewStyle().Foreground(c).Bold(true).Render(strings.ToUpper(title))
	rule := max(width-lipgloss.Width(head)-1, 0)
	return head + " " + lipgloss.NewStyle().Foreground(Border).Render(strings.Repeat("─", rule))
}

// SpaceBetween lays out left and right at opposite ends of width columns.
func SpaceBetween(width int, left, right string) string {
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

// ProgressBar renders a filled/empty bar of width cells for ratio in [0,1],
// with the filled part fading along the brand gradient.
func ProgressBar(width int, ratio float64) string {
	ratio = min(max(ratio, 0), 1)
	filled := int(ratio*float64(width) + 0.5)
	var sb strings.Builder
	if filled > 0 {
		ramp := lipgloss.Blend1D(max(width, 2), AccentDim, Accent)
		for i := range filled {
			sb.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Render("━"))
		}
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(Border).Render(strings.Repeat("━", width-filled)))
	return sb.String()
}

// Meter renders n segments with the first filled ones lit along the brand
// gradient, e.g. "▰▰▰▱▱▱".
func Meter(n, filled int) string {
	ramp := lipgloss.Blend1D(max(n, 2), Accent, AccentAlt)
	var sb strings.Builder
	for i := range n {
		if i < filled {
			sb.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Render("▰"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(Border).Render("▱"))
		}
	}
	return sb.String()
}

// Gradient colours each column of s along the Accent → AccentAlt gradient.
func Gradient(s string) string {
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
