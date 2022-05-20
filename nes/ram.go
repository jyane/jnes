package nes

type RAM struct {
	data [2048]byte
}

// NewRAM creates a RAM for both PPU and CPU.
func NewRAM() *RAM {
	return &RAM{}
}

// read reads data
func (r *RAM) read(address uint16) byte {
	return r.data[address]
}

// write writes data
func (r *RAM) write(address uint16, x byte) {
	r.data[address] = x
}
