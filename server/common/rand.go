package common

// XORShift es una estructura que contiene el estado del generador.
type XORShift struct {
	seed uint32
}

// NewXORShift inicializa un generador XORShift con una semilla.
func NewXORShift(seed uint32) *XORShift {
	if seed == 0 {
		seed = 1 // Asegurarse de que la semilla no sea 0, ya que generará siempre 0.
	}
	return &XORShift{seed: seed}
}

// Next genera el siguiente número pseudoaleatorio.
func (x *XORShift) Next() uint32 {
	x.seed ^= x.seed << 13
	x.seed ^= x.seed >> 17
	x.seed ^= x.seed << 5
	return x.seed
}
