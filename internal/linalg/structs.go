package linalg

type Matrix struct {
    Rows int
    Cols int
    Data []float64
}

func NewMatrix(rows, cols int) *Matrix {
    return &Matrix{
        Rows: rows,
        Cols: cols,
        Data: make([]float64, rows*cols),
    }
}

func (m *Matrix) Get(i, j int) float64 {
    return m.Data[i*m.Cols+j]
}

func (m *Matrix) Set(i, j int, val float64) {
    m.Data[i*m.Cols+j] = val
}