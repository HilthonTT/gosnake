package snake

const (
	DefaultCols = 46
	DefaultRows = 40
	Cell        = 32
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
	return p.X >= 0 && p.X < len(m[0]) && p.Y >= 0 && p.Y < len(m)
}
