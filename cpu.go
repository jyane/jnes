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
	Implicit
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

var instructionModes = [256]Mode{
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Immediate, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
	Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit, Implicit,
}

var instructionSizes = [256]uint16{
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
}

type CPU struct {
	PC           uint16              // program counter
	A            byte                // accumulator register
	X            byte                // index register
	Y            byte                // index register
	S            byte                // stack pointer
	P            *Status             // processor status bits
	Bus                              // bus
	cycles       int                 // current cycles
	instructions [256]func(Mode) int // instructions
}

func (cpu *CPU) createTable() {
	cpu.instructions = [256]func(mode Mode) int{
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.adc, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
		cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop, cpu.nop,
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
	cpu.PC = cpu.bus.Read16(0xFFFC)
	cpu.S = 0xFD
	cpu.SetFlags(0x24)
}

func NewCPU(bus *CPUBus) *CPU {
	cpu := &CPU{P: newStatus(), bus: bus}
	cpu.createTable()
	return cpu
}

// Step performs fetch - decode - execute steps
func (cpu *CPU) Step() int {
	log.Println(cpu.PC)
	opcode := cpu.bus.Read(cpu.PC)
	log.Println(opcode)
	cycle := cpu.instructions[opcode](instructionModes[opcode])
	cpu.PC += instructionSizes[opcode]
	return cycle
}

// ADC - Add with Carry
func (cpu *CPU) adc(mode Mode) int {
	switch mode {
	case Immediate:
		return 2
	}
	return 0
}

// NOP - No Operation
func (cpu *CPU) nop(mode Mode) int {
	return 2
}
