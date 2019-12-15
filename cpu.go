package main

import "log"

type Status struct {
	C bool // carry flag
	Z bool // zero flag
	I bool // IRQ flag
	D bool // decimal flag
	B bool // break flag
	R bool // reserved flag
	V bool // overflow flag
	N bool // negative flag
}

func newStatus() *Status {
	return &Status{false, false, true, true, false, true, false, false}
}

type Mode int

const (
	_ Mode = iota
	Accumulator
	Immediate
	ZeroPage
	ZeroPageX
	ZeroPageY
	Relative
	Absolute
	AbsoluteX
	AbsoluteY
	Indirect
	IndexedIndirect
	IndirectIndexed
)

var instructionMode = [256]Mode{
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

type CPU struct {
	PC           uint16              // program counter
	A            byte                // accumulator register
	X            byte                // index register
	Y            byte                // index register
	S            byte                // stack pointer
	P            *Status             // processor status bits
	bus          *CPUBus             // bus
	cycles       int                 // current cycles
	instructions [256]func(Mode) int // instructions
}

func (cpu *CPU) createTable() {
	cpu.instructions = [256]func(mode Mode) int{
		cpu.adc,
	}
}

func (cpu *CPU) SetFlags(f byte) {
	cpu.P.C = (f>>0)&1 == 1
	cpu.P.Z = (f>>1)&1 == 1
	cpu.P.I = (f>>2)&1 == 1
	cpu.P.D = (f>>3)&1 == 1
	cpu.P.B = (f>>4)&1 == 1
	cpu.P.R = (f>>5)&1 == 1
	cpu.P.V = (f>>6)&1 == 1
	cpu.P.N = (f>>7)&1 == 1
}

func (cpu *CPU) Reset() {
	log.Printf("CPU reset!\n")
	cpu.PC = cpu.bus.Read16(0xFFFC)
	cpu.S = 0xFD
	cpu.SetFlags(0x24)
}

func NewCPU(bus *CPUBus) *CPU {
	cpu := &CPU{P: newStatus(), bus: bus}
	cpu.createTable()
	cpu.Reset()
	return cpu
}

func (cpu *CPU) Step() int {
	opcode := cpu.bus.Read(cpu.PC)
	return cpu.instructions[opcode](instructionMode[opcode])
}

func (cpu *CPU) adc(mode Mode) int {
	return 42
}
