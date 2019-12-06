package main

type Status struct {
	N bool // negative flag
	V bool // overflow flag
	R bool // reserved flag
	B bool // break flag
	D bool // decimal flag
	I bool // IRQ flag
	Z bool // zero flag
	C bool // carry flag
}

func newStatus() *Status {
	return &Status{false, false, true, true, false, true, false, false}
}

type CPU struct {
	PC uint16  // program counter
	A  byte    // accumulator register
	X  byte    // index register
	Y  byte    // index register
	S  byte    // stack pointer
	P  *Status // processor status bits

	wram *RAM // wram
}

func NewCPU(wram *RAM) *CPU {
	return &CPU{0x0000, 0x00, 0x00, 0x00, 0x00, newStatus(), wram}
}
