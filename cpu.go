package main

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

// NewStatus creates new Status.
func NewStatus() *Status {
	// 0x24 = 36
	return &Status{false, false, true, false, false, true, false, false}
}

// addressing mode
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
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
}

var instructionCycles = [256]int{
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
	PC           uint16            // program counter
	A            byte              // accumulator register
	X            byte              // index register
	Y            byte              // index register
	S            byte              // stack pointer
	P            *Status           // processor status bits
	bus          Bus               // bus
	cycles       int               // current cycles
	instructions [256]func(uint16) // instructions
}

func (cpu *CPU) createTable() {
	cpu.instructions = [256]func(rhs uint16){
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

// setFlags replces all status-flags of the CPU.
func (cpu *CPU) setFlags(f byte) {
	cpu.P.C = (f>>0)&1 == 1
	cpu.P.Z = (f>>1)&1 == 1
	cpu.P.I = (f>>2)&1 == 1
	cpu.P.D = (f>>3)&1 == 1
	cpu.P.B = (f>>4)&1 == 1
	cpu.P.R = (f>>5)&1 == 1
	cpu.P.V = (f>>6)&1 == 1
	cpu.P.N = (f>>7)&1 == 1
}

// Read reads data (1 byte) from bus.
func (cpu *CPU) read(address uint16) byte {
	return cpu.bus.Read(address)
}

// Read16 reads data (2 bytes) from bus.
func (cpu *CPU) read16(address uint16) uint16 {
	low := uint16(cpu.bus.Read(address))
	high := uint16(cpu.bus.Read(address+1)) << 8 // e.g. 11011011 00000000
	return high | low
}

// getRhs gets a right-hand-side value (An operand but not operand) which will be calcurated by instructions.
// http://www.thealmightyguru.com/Games/Hacking/Wiki/index.php/Addressing_Modes
//
// e.g.
// ADC $42 ; If the addressing mode is immediate and the right-hand-side value is $42 = 0x42 returns 0x42.
// ADC     ; If the addressing mode is accumulator returns cpu.A's value.
func (cpu *CPU) getRhs(mode Mode) uint16 {
	switch mode {
	case Implicit:
		return 0
	case Accumulator:
		return uint16(cpu.A)
	case Immediate:
		return uint16(cpu.PC + 1)
	case ZeroPage:
		return uint16(cpu.read(cpu.PC + 1))
	case ZeroPageX:
		return uint16(cpu.read(cpu.PC+1)+cpu.X) & 0x00FF
	case ZeroPageY:
		return uint16(cpu.read(cpu.PC+1)+cpu.Y) & 0x00FF
	case Relative:
		x := uint16(cpu.read(cpu.PC + 1))
		if x < 0x80 {
			return cpu.PC + x
		} else {
			return cpu.PC + x - 0x100
		}
	case Absolute:
		return cpu.read16(cpu.PC + 1)
	case AbsoluteX:
		return cpu.read16(cpu.PC+1) + uint16(cpu.X)
	case AbsoluteY:
		return cpu.read16(cpu.PC+1) + uint16(cpu.Y)
	case Indirect:
	case IndexedIndirect:
	case IndirectIndexed:
	}
	return 0
}

// NewCPU returns new CPU.
func NewCPU(bus Bus) *CPU {
	cpu := &CPU{P: NewStatus(), bus: bus}
	cpu.createTable()
	return cpu
}

// Reset resets CPU state.
func (cpu *CPU) Reset() {
	cpu.PC = cpu.read16(0xFFFC)
	cpu.S = 0xFD
	cpu.setFlags(0x24)
}

// Step performs fetch - decode - execute steps.
func (cpu *CPU) Step() int {
	opcode := cpu.read(cpu.PC)
	rhs := cpu.getRhs(instructionModes[opcode])
	cpu.instructions[opcode](rhs)
	cpu.PC += instructionSizes[opcode]
	cycle := instructionCycles[opcode]
	return cycle
}

// ADC - Add with Carry
func (cpu *CPU) adc(rhs uint16) {
}

// NOP - No Operation
func (cpu *CPU) nop(rhs uint16) {
}
