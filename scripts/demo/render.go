package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Terminal theme the frames are drawn in (GitHub dark).
var (
	termBG = hex("#0D1117")
	termFG = hex("#E6EDF3")
)

type cell struct {
	text   string
	fg, bg color.RGBA
	bold   bool
}

type grid struct {
	w, h  int
	cells [][]cell
}

func newGrid(w, h int) *grid {
	g := &grid{w: w, h: h, cells: make([][]cell, h)}
	for y := range g.cells {
		g.cells[y] = make([]cell, w)
		for x := range g.cells[y] {
			g.cells[y][x] = cell{text: " ", fg: termFG, bg: termBG}
		}
	}
	return g
}

func (g *grid) text() string {
	var sb strings.Builder
	for _, row := range g.cells {
		for _, c := range row {
			sb.WriteString(c.text)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (g *grid) equal(o *grid) bool {
	if o == nil || g.w != o.w || g.h != o.h {
		return false
	}
	for y := range g.cells {
		for x := range g.cells[y] {
			if g.cells[y][x] != o.cells[y][x] {
				return false
			}
		}
	}
	return true
}

// parseANSI lays a rendered view out on a w×h grid, interpreting the SGR
// sequences lipgloss emits. Other escape sequences are skipped.
func parseANSI(s string, w, h int) *grid {
	g := newGrid(w, h)
	var (
		fg, bg           = termFG, termBG
		bold, faint, rev bool
		x, y             int
	)
	for len(s) > 0 {
		if s[0] == '\x1b' {
			n := escapeLen(s)
			if n > 2 && s[1] == '[' && s[n-1] == 'm' {
				applySGR(s[2:n-1], &fg, &bg, &bold, &faint, &rev)
			}
			s = s[n:]
			continue
		}
		if s[0] == '\n' {
			x, y = 0, y+1
			s = s[1:]
			continue
		}
		gr, width := ansi.FirstGraphemeCluster(s, ansi.GraphemeWidth)
		s = s[len(gr):]
		if y >= h {
			continue
		}
		cf, cb := fg, bg
		if rev {
			cf, cb = cb, cf
		}
		if faint {
			cf = blend(cf, cb, 0.5)
		}
		for i := 0; i < max(width, 1); i++ {
			if x < w {
				t := gr
				if i > 0 {
					t = ""
				}
				g.cells[y][x] = cell{text: t, fg: cf, bg: cb, bold: bold}
			}
			x++
		}
	}
	return g
}

func escapeLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == '\a' {
				return i + 1
			}
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
	default:
		return 2
	}
	return len(s)
}

func applySGR(params string, fg, bg *color.RGBA, bold, faint, rev *bool) {
	if params == "" {
		params = "0"
	}
	ps := strings.FieldsFunc(strings.ReplaceAll(params, ":", ";"), func(r rune) bool { return r == ';' })
	num := func(i int) int {
		if i >= len(ps) {
			return 0
		}
		n, _ := strconv.Atoi(ps[i])
		return n
	}
	for i := 0; i < len(ps); i++ {
		switch p := num(i); {
		case p == 0:
			*fg, *bg, *bold, *faint, *rev = termFG, termBG, false, false, false
		case p == 1:
			*bold = true
		case p == 2:
			*faint = true
		case p == 7:
			*rev = true
		case p == 22:
			*bold, *faint = false, false
		case p == 27:
			*rev = false
		case p == 39:
			*fg = termFG
		case p == 49:
			*bg = termBG
		case p == 38 || p == 48:
			var c color.RGBA
			switch num(i + 1) {
			case 2:
				c = color.RGBA{uint8(num(i + 2)), uint8(num(i + 3)), uint8(num(i + 4)), 255}
				i += 4
			case 5:
				c = xterm256(num(i + 2))
				i += 2
			}
			if p == 38 {
				*fg = c
			} else {
				*bg = c
			}
		case p >= 30 && p <= 37:
			*fg = xterm256(p - 30)
		case p >= 90 && p <= 97:
			*fg = xterm256(p - 90 + 8)
		case p >= 40 && p <= 47:
			*bg = xterm256(p - 40)
		case p >= 100 && p <= 107:
			*bg = xterm256(p - 100 + 8)
		}
	}
}

func xterm256(n int) color.RGBA {
	base := []string{"#000000", "#CD3131", "#0DBC79", "#E5E510", "#2472C8", "#BC3FBC", "#11A8CD", "#E5E5E5",
		"#666666", "#F14C4C", "#23D18B", "#F5F543", "#3B8EEA", "#D670D6", "#29B8DB", "#FFFFFF"}
	switch {
	case n < 16:
		return hex(base[n])
	case n < 232:
		n -= 16
		lv := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return color.RGBA{lv(n / 36), lv(n / 6 % 6), lv(n % 6), 255}
	default:
		v := uint8(8 + (n-232)*10)
		return color.RGBA{v, v, v, 255}
	}
}

// renderer draws grids into images.
type renderer struct {
	main, fallback font.Face
	cw, ch, ascent int
}

func newRenderer(mainPath, fallbackPath string, size float64) (*renderer, error) {
	load := func(path string) (font.Face, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		f, err := opentype.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	}
	r := &renderer{}
	var err error
	if r.main, err = load(mainPath); err != nil {
		return nil, err
	}
	if fallbackPath != "" {
		if r.fallback, err = load(fallbackPath); err != nil {
			return nil, err
		}
	}
	adv, _ := r.main.GlyphAdvance('M')
	m := r.main.Metrics()
	r.cw = adv.Round()
	r.ascent = m.Ascent.Round() + 1
	r.ch = m.Ascent.Round() + m.Descent.Round() + 3
	return r, nil
}

func (r *renderer) draw(g *grid) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, g.w*r.cw, g.h*r.ch))
	for y, row := range g.cells {
		for x, c := range row {
			rect := image.Rect(x*r.cw, y*r.ch, (x+1)*r.cw, (y+1)*r.ch)
			fill(img, rect, c.bg)
			r.glyph(img, rect, c)
		}
	}
	return img
}

// glyph draws one cell's text. Block and line characters are drawn as
// rectangles so they tile without seams; everything else uses the font,
// with alpha quantised to a few levels to keep the GIF palette small.
func (r *renderer) glyph(img *image.RGBA, rect image.Rectangle, c cell) {
	w, h := rect.Dx(), rect.Dy()
	sub := func(x0, y0, x1, y1 int) image.Rectangle {
		return image.Rect(rect.Min.X+x0, rect.Min.Y+y0, rect.Min.X+x1, rect.Min.Y+y1)
	}
	switch c.text {
	case "", " ":
		return
	case "█":
		fill(img, rect, c.fg)
		return
	case "▀":
		fill(img, sub(0, 0, w, h/2), c.fg)
		return
	case "▄":
		fill(img, sub(0, h/2, w, h), c.fg)
		return
	case "─":
		fill(img, sub(0, h/2, w, h/2+1), c.fg)
		return
	case "━":
		fill(img, sub(0, h/2-1, w, h/2+1), c.fg)
		return
	}

	runes := []rune(c.text)
	face := r.main
	if _, ok := face.GlyphAdvance(runes[0]); !ok || !hasGlyph(face, runes[0]) {
		if r.fallback != nil {
			face = r.fallback
		}
	}
	dot := fixed.P(rect.Min.X, rect.Min.Y+r.ascent)
	if adv, ok := face.GlyphAdvance(runes[0]); ok {
		// Centre narrower or wider symbols within the cell.
		dot.X += (fixed.I(r.cw) - adv) / 2
	}
	passes := 1
	if c.bold {
		passes = 2
	}
	for p := range passes {
		dr, mask, mp, _, ok := face.Glyph(dot.Add(fixed.P(p, 0)), runes[0])
		if !ok {
			continue
		}
		for py := dr.Min.Y; py < dr.Max.Y; py++ {
			for px := dr.Min.X; px < dr.Max.X; px++ {
				if !(image.Point{px, py}.In(rect)) {
					continue
				}
				_, _, _, a := mask.At(mp.X+px-dr.Min.X, mp.Y+py-dr.Min.Y).RGBA()
				level := float64((a*4+0x7fff)/0xffff) / 4 // 0, ¼, ½, ¾, 1
				if level == 0 {
					continue
				}
				under := img.RGBAAt(px, py)
				img.SetRGBA(px, py, blend(c.fg, under, level))
			}
		}
	}
}

func hasGlyph(f font.Face, r rune) bool {
	_, _, _, _, ok := f.Glyph(fixed.P(0, 0), r)
	if !ok {
		return false
	}
	// Faces report a .notdef box for missing runes; treat an empty bounds
	// result for a visible rune as missing.
	b, _, ok := f.GlyphBounds(r)
	return ok && b.Max.X > b.Min.X
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func blend(a, b color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*t + float64(y)*(1-t) + 0.5) }
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 255}
}

func hex(s string) color.RGBA {
	v, _ := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

// recorder collects frames with their timestamps.
type recorder struct {
	r      *renderer
	frames []frame
}

type frame struct {
	g  *grid
	at time.Time
}

func (rec *recorder) add(g *grid, at time.Time) {
	if n := len(rec.frames); n > 0 && rec.frames[n-1].g.equal(g) {
		return
	}
	rec.frames = append(rec.frames, frame{g: g, at: at})
}

// writeGIF encodes the frames. Each frame after the first stores only the
// rectangle that changed, with its own palette.
func (rec *recorder) writeGIF(path string) error {
	if len(rec.frames) == 0 {
		return fmt.Errorf("no frames recorded")
	}
	out := &gif.GIF{LoopCount: 0}
	var prev *image.RGBA
	var pendingDelay int

	for i, f := range rec.frames {
		img := rec.r.draw(f.g)

		delay := 300 // hold the last frame for 3s before looping
		if i+1 < len(rec.frames) {
			delay = int(rec.frames[i+1].at.Sub(f.at) / (10 * time.Millisecond))
		}

		bounds := img.Bounds()
		if prev != nil {
			bounds = diffBounds(prev, img)
			if bounds.Empty() {
				// Nothing visible changed: extend the previous frame.
				if n := len(out.Delay); n > 0 {
					out.Delay[n-1] += delay
				}
				continue
			}
		}
		delay += pendingDelay
		pendingDelay = 0
		if delay < 2 {
			// Browsers clamp tiny delays; fold them into the next frame.
			pendingDelay = delay
			delay = 2
		}

		out.Image = append(out.Image, quantize(img, bounds))
		out.Delay = append(out.Delay, delay)
		out.Disposal = append(out.Disposal, gif.DisposalNone)
		prev = img
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, out)
}

func diffBounds(a, b *image.RGBA) image.Rectangle {
	var r image.Rectangle
	bb := b.Bounds()
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		ra := a.Pix[y*a.Stride : y*a.Stride+bb.Dx()*4]
		rb := b.Pix[y*b.Stride : y*b.Stride+bb.Dx()*4]
		if string(ra) == string(rb) {
			continue
		}
		for x := 0; x < bb.Dx(); x++ {
			if ra[x*4] != rb[x*4] || ra[x*4+1] != rb[x*4+1] || ra[x*4+2] != rb[x*4+2] {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

// quantize converts the bounds of img to a paletted image. Frames use few
// colours, so the palette is exact unless more than 256 appear, in which
// case the rarest map to their nearest neighbour.
func quantize(img *image.RGBA, bounds image.Rectangle) *image.Paletted {
	counts := map[color.RGBA]int{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			counts[img.RGBAAt(x, y)]++
		}
	}
	cols := make([]color.RGBA, 0, len(counts))
	for c := range counts {
		cols = append(cols, c)
	}
	sort.Slice(cols, func(i, j int) bool { return counts[cols[i]] > counts[cols[j]] })
	if len(cols) > 256 {
		cols = cols[:256]
	}
	pal := make(color.Palette, len(cols))
	index := make(map[color.RGBA]uint8, len(counts))
	for i, c := range cols {
		pal[i] = c
		index[c] = uint8(i)
	}

	p := image.NewPaletted(bounds, pal)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			i, ok := index[c]
			if !ok {
				i = uint8(pal.Index(c))
				index[c] = i
			}
			p.SetColorIndex(x, y, i)
		}
	}
	return p
}
