package snake

const (
	DefaultCols = 46
	DefaultRows = 40
)

// Cell values written into a Matrix. Renderers switch on these.
const (
	CellEmpty       byte = 0
	CellHead        byte = 'H'
	CellBody        byte = 'S'
	CellFood        byte = 'F'
	CellBomb        byte = 'B'
	CellBombWarning byte = 'W'
	CellAIHead      byte = 'A'
	CellAIBody      byte = 'Z'
)

type Matrix [][]byte

func NewMatrix(rows, cols int) Matrix {
	m := make(Matrix, rows)
	for i := range m {
		m[i] = make([]byte, cols)
	}
	return m
}

func (m Matrix) Set(p Point, val byte) {
	m[p.Y][p.X] = val
}

func (m Matrix) Get(p Point) byte {
	return m[p.Y][p.X]
}

func (m Matrix) InBounds(p Point) bool {
	return p.Y >= 0 && p.Y < len(m) && p.X >= 0 && p.X < len(m[p.Y])
}

// Clear resets every cell to CellEmpty.
func (m Matrix) Clear() {
	for _, row := range m {
		clear(row)
	}
}

// Clone returns a deep copy of m.
func (m Matrix) Clone() Matrix {
	out := make(Matrix, len(m))
	for i, row := range m {
		out[i] = append([]byte(nil), row...)
	}
	return out
}

// DrawSnake writes body onto m using head for the first segment and bodyCell
// for the rest.
func (m Matrix) DrawSnake(body []Point, head, bodyCell byte) {
	for i, p := range body {
		if !m.InBounds(p) {
			continue
		}
		if i == 0 {
			m.Set(p, head)
		} else {
			m.Set(p, bodyCell)
		}
	}
}
