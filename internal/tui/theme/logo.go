package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// iconPixels is the app icon — an S-shaped snake heading for a food pellet —
// as pixel art matching docs/assets/logo.svg. Two pixels stack in each
// terminal cell using half blocks, so it is five rows tall.
//
// Pixel codes: '#' body, 'h' head, 'o' eye, 'f' food.
var iconPixels = []string{
	".....#######..",
	"....########..",
	"...##.........",
	"...##.........",
	"....#######...",
	".....#######..",
	"...........##.",
	"...........##.",
	"ff.hoh######..",
	"ff.hhh#####...",
}

var (
	iconHead = lipgloss.Color("#86EFAC")
	iconEye  = lipgloss.Color("#052E16")
	iconTail = lipgloss.Color("#166534")

	icon = buildIcon()
	logo = buildLogo()
)

// LogoWidth is the width of the full logo in columns.
var LogoWidth = lipgloss.Width(logo)

// LogoHeight is the height of the full logo in rows.
var LogoHeight = lipgloss.Height(logo)

// Logo returns the icon with the name and tagline beside it, or a one-line
// wordmark when small is set.
func Logo(small bool) string {
	if small {
		return Wordmark()
	}
	return logo
}

// Icon returns the pixel-art app icon on its own.
func Icon() string { return icon }

// Wordmark is the one-line logo: "gosnake ●".
func Wordmark() string {
	return Gradient("gosnake") + " " + lipgloss.NewStyle().Foreground(Gold).Render("●")
}

func buildLogo() string {
	name := lipgloss.NewStyle().Bold(true).Render(Gradient("gosnake"))
	text := lipgloss.JoinVertical(lipgloss.Left,
		name,
		Subtitle.Render("eat · grow · survive"),
	)
	text = lipgloss.PlaceVertical(lipgloss.Height(icon), lipgloss.Center, text)
	return lipgloss.JoinHorizontal(lipgloss.Center, icon, "   ", text)
}

func buildIcon() string {
	width := len(iconPixels[0])

	// The body fades from the tail (top right) to the head (bottom left).
	steps := width + len(iconPixels)
	ramp := lipgloss.Blend1D(steps, iconTail, Accent, iconHead)

	colorAt := func(r, c int) color.Color {
		if r >= len(iconPixels) {
			return nil
		}
		switch iconPixels[r][c] {
		case '#':
			return ramp[(width-1-c)+r]
		case 'h':
			return iconHead
		case 'o':
			return iconEye
		case 'f':
			return Gold
		}
		return nil
	}

	lines := make([]string, 0, (len(iconPixels)+1)/2)
	for r := 0; r < len(iconPixels); r += 2 {
		var sb strings.Builder
		for c := range width {
			sb.WriteString(halfBlock(colorAt(r, c), colorAt(r+1, c)))
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n")
}

// halfBlock draws two vertically stacked pixels in one cell; nil pixels show
// the terminal background.
func halfBlock(top, bottom color.Color) string {
	switch {
	case top == nil && bottom == nil:
		return " "
	case bottom == nil:
		return lipgloss.NewStyle().Foreground(top).Render("▀")
	case top == nil:
		return lipgloss.NewStyle().Foreground(bottom).Render("▄")
	case top == bottom:
		return lipgloss.NewStyle().Foreground(top).Render("█")
	default:
		return lipgloss.NewStyle().Foreground(top).Background(bottom).Render("▀")
	}
}
