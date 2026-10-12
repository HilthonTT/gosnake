package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Scale is how many terminal cells one board cell occupies.
type Scale int

const (
	// ScaleCompact draws two board rows per terminal row using half blocks,
	// so each board cell is one column wide and half a row tall — roughly
	// square in most fonts.
	ScaleCompact Scale = iota

	// ScaleLarge draws each board cell as two blank columns with a
	// background colour. It renders perfectly in every terminal.
	ScaleLarge
)

// BoardSize returns the rendered size of a rows×cols board, walls included.
func BoardSize(rows, cols int, scale Scale) (width, height int) {
	rows, cols = rows+2, cols+2 // one wall cell on each side
	if scale == ScaleLarge {
		return cols * 2, rows
	}
	return cols, (rows + 1) / 2
}

// PixelFunc returns the colour of the board cell at (row, col), or nil for an
// empty cell.
type PixelFunc func(row, col int) color.Color

// BoardColors are the colours of the parts of the board that aren't pieces.
type BoardColors struct {
	Wall color.Color

	// Empty returns the floor colour at (row, col). Compact boards use
	// Empty(0, 0) everywhere, since a half-block cell can only show two
	// colours.
	Empty PixelFunc
}

// RenderBoard draws a rows×cols grid of pixels surrounded by a wall.
//
// Block glyphs are drawn by the font in some terminals and then don't fill
// their cell, leaving seams. So the renderer only uses background colours
// wherever it can, and uses a half-block glyph only for a cell that holds a
// piece, in the piece's own colour.
func RenderBoard(rows, cols int, scale Scale, colors BoardColors, px PixelFunc) string {
	// at includes the wall; nil means floor.
	at := func(r, c int) color.Color {
		if r == 0 || c == 0 || r == rows+1 || c == cols+1 {
			return colors.Wall
		}
		return px(r-1, c-1)
	}
	totalRows, totalCols := rows+2, cols+2

	var sb strings.Builder

	if scale == ScaleLarge {
		for r := range totalRows {
			if r > 0 {
				sb.WriteByte('\n')
			}
			writeRuns(&sb, totalCols, func(c int) cell {
				bg := at(r, c)
				if bg == nil {
					bg = colors.Empty(r-1, c-1)
				}
				return cell{glyph: "  ", bg: bg}
			})
		}
		return sb.String()
	}

	floor := colors.Empty(0, 0)
	for r := 0; r < totalRows; r += 2 {
		if r > 0 {
			sb.WriteByte('\n')
		}
		writeRuns(&sb, totalCols, func(c int) cell {
			top := at(r, c)
			var bottom color.Color
			if r+1 < totalRows {
				bottom = at(r+1, c)
			}
			switch {
			case top == nil && bottom == nil:
				return cell{glyph: " ", bg: floor}
			case bottom == nil:
				return cell{glyph: "▀", fg: top, bg: floor}
			case top == nil:
				return cell{glyph: "▄", fg: bottom, bg: floor}
			case sameColor(top, bottom):
				return cell{glyph: " ", bg: top}
			default:
				return cell{glyph: "▀", fg: top, bg: bottom}
			}
		})
	}
	return sb.String()
}

type cell struct {
	glyph  string
	fg, bg color.Color
}

func (a cell) same(b cell) bool {
	return a.glyph == b.glyph && sameColor(a.fg, b.fg) && sameColor(a.bg, b.bg)
}

// writeRuns writes n cells, styling consecutive identical cells as a single
// run to keep the output (and the work per frame) small.
func writeRuns(sb *strings.Builder, n int, at func(i int) cell) {
	start := 0
	cur := at(0)
	for i := 1; i <= n; i++ {
		var next cell
		if i < n {
			next = at(i)
			if next.same(cur) {
				continue
			}
		}
		style := lipgloss.NewStyle()
		if cur.fg != nil {
			style = style.Foreground(cur.fg)
		}
		if cur.bg != nil {
			style = style.Background(cur.bg)
		}
		sb.WriteString(style.Render(strings.Repeat(cur.glyph, i-start)))
		start, cur = i, next
	}
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// Overlay draws fg centred on top of bg.
func Overlay(bg, fg string) string {
	bw, bh := lipgloss.Size(bg)
	fw, fh := lipgloss.Size(fg)
	canvas := lipgloss.NewCanvas(bw, bh)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(bg),
		lipgloss.NewLayer(fg).X(max((bw-fw)/2, 0)).Y(max((bh-fh)/2, 0)).Z(1),
	))
	return canvas.Render()
}
