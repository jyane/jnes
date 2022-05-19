package nes

import "github.com/golang/glog"

// CPU emulates NES CPU - is custom 6502 made by RICOH.
// References:
//   https://en.wikipedia.org/wiki/MOS_Technology_6502
//   http://www.6502.org/tutorials/6502opcodes.html
//   http://hp.vector.co.jp/authors/VA042397/nes/6502.html (In Japanese)

type addressingMode int

const (
	Implied addressingMode = iota
	Accumulator
	Immediate
	Zeropage
	ZeropageX
	ZeropageY
	Relative
	Absolute
	AbsoluteX
	AbsoluteY
	Indirect
	IndirectX
	IndirectY
)

type status struct {
	C bool // carry
	Z bool // zero
	I bool // IRQ
	D bool // decimal - unused on NES
	B bool // break
	R bool // reserved - unused
	V bool // overflow
	N bool // negative
}

// encode encodes the status to a byte.
func (s *status) encode() byte {
	var res byte
	if s.C {
		res |= (1 << 0)
	}
	if s.Z {
		res |= (1 << 1)
	}
	if s.I {
		res |= (1 << 2)
	}
	if s.D {
		res |= (1 << 3)
	}
	if s.B {
		res |= (1 << 4)
	}
	if s.R {
		res |= (1 << 5)
	}
	if s.V {
		res |= (1 << 6)
	}
	if s.N {
		res |= (1 << 7)
	}
	return res
}

// decodeFrom decodes a byte to the status.
func (s *status) decodeFrom(data byte) {
	s.C = (data>>0)&1 == 1
	s.Z = (data>>1)&1 == 1
	s.I = (data>>2)&1 == 1
	s.D = (data>>3)&1 == 1
	s.B = (data>>4)&1 == 1
	s.R = (data>>5)&1 == 1
	s.V = (data>>6)&1 == 1
	s.N = (data>>7)&1 == 1
}

type CPU struct {
	P            *status // Processor status flag bits
	A            byte    // Accumulator register
	X            byte    // Index register
	Y            byte    // Index register
	PC           uint16  // Program counter
	S            byte    // Stack pointer
	bus          *CPUBus
	instructions []instruction
}

type instruction struct {
	mnemonic string
	mode     addressingMode
	execute  func(addressingMode, uint16)
	size     uint16
	cycles   int
}

func (c *CPU) createInstructions() []instruction {
	return []instruction{
		{"BRK", Implied, c.brk, 1, 7},     // 0x00
		{"ORA", IndirectX, c.ora, 2, 6},   // 0x01
		{"", Implied, c.nop, 1, 2},        // 0x02
		{"", Implied, c.nop, 1, 2},        // 0x03
		{"", Implied, c.nop, 1, 2},        // 0x04
		{"ORA", Zeropage, c.ora, 2, 3},    // 0x05
		{"ASL", Zeropage, c.asl, 2, 5},    // 0x06
		{"", Implied, c.nop, 1, 2},        // 0x07
		{"PHP", Implied, c.php, 1, 3},     // 0x08
		{"ORA", Immediate, c.ora, 2, 2},   // 0x09
		{"ASL", Accumulator, c.asl, 1, 2}, // 0x0A
		{"", Implied, c.nop, 1, 2},        // 0x0B
		{"", Implied, c.nop, 1, 2},        // 0x0C
		{"ORA", Absolute, c.ora, 3, 4},    // 0x0D
		{"ASL", Absolute, c.asl, 3, 6},    // 0x0E
		{"", Implied, c.nop, 1, 2},        // 0x0F
		{"BPL", Relative, c.bpl, 2, 2},    // 0x10
		{"ORA", IndirectY, c.ora, 2, 5},   // 0x11
		{"", Implied, c.nop, 1, 2},        // 0x12
		{"", Implied, c.nop, 1, 2},        // 0x13
		{"", Implied, c.nop, 1, 2},        // 0x14
		{"ORA", ZeropageX, c.ora, 2, 4},   // 0x15
		{"ASL", ZeropageX, c.asl, 2, 6},   // 0x16
		{"", Implied, c.nop, 1, 2},        // 0x17
		{"CLC", Implied, c.clc, 1, 2},     // 0x18
		{"ORA", AbsoluteY, c.ora, 3, 4},   // 0x19
		{"", Implied, c.nop, 1, 2},        // 0x1A
		{"", Implied, c.nop, 1, 2},        // 0x1B
		{"", Implied, c.nop, 1, 2},        // 0x1C
		{"ORA", AbsoluteX, c.ora, 3, 4},   // 0x1D
		{"ASL", AbsoluteX, c.asl, 3, 7},   // 0x1E
		{"", Implied, c.nop, 1, 2},        // 0x1F
		{"JSR", Absolute, c.jsr, 3, 6},    // 0x20
		{"AND", IndirectX, c.and, 2, 6},   // 0x21
		{"", Implied, c.nop, 1, 2},        // 0x22
		{"", Implied, c.nop, 1, 2},        // 0x23
		{"BIT", Zeropage, c.bit, 2, 3},    // 0x24
		{"AND", Zeropage, c.and, 2, 3},    // 0x25
		{"ROL", Zeropage, c.rol, 2, 5},    // 0x26
		{"", Implied, c.nop, 1, 2},        // 0x27
		{"PLP", Implied, c.plp, 1, 4},     // 0x28
		{"AND", Immediate, c.and, 2, 2},   // 0x29
		{"ROL", Accumulator, c.rol, 1, 2}, // 0x2A
		{"", Implied, c.nop, 1, 2},        // 0x2B
		{"BIT", Absolute, c.bit, 3, 4},    // 0x2C
		{"AND", Absolute, c.and, 3, 4},    // 0x2D
		{"ROL", Absolute, c.rol, 3, 6},    // 0x2E
		{"", Implied, c.nop, 1, 2},        // 0x2F
		{"BMI", Relative, c.bmi, 2, 2},    // 0x30
		{"AND", IndirectY, c.and, 2, 5},   // 0x31
		{"", Implied, c.nop, 1, 2},        // 0x32
		{"", Implied, c.nop, 1, 2},        // 0x33
		{"", Implied, c.nop, 1, 2},        // 0x34
		{"AND", ZeropageX, c.and, 2, 4},   // 0x35
		{"ROL", ZeropageX, c.rol, 2, 6},   // 0x36
		{"", Implied, c.nop, 1, 2},        // 0x37
		{"SEC", Implied, c.sec, 1, 2},     // 0x38
		{"AND", AbsoluteY, c.and, 3, 4},   // 0x39
		{"", Implied, c.nop, 1, 2},        // 0x3A
		{"", Implied, c.nop, 1, 2},        // 0x3B
		{"", Implied, c.nop, 1, 2},        // 0x3C
		{"AND", AbsoluteX, c.and, 3, 4},   // 0x3D
		{"ROL", AbsoluteX, c.rol, 3, 7},   // 0x3E
		{"", Implied, c.nop, 1, 2},        // 0x3F
		{"RTI", Implied, c.rti, 1, 6},     // 0x40
		{"EOR", IndirectX, c.eor, 2, 6},   // 0x41
		{"", Implied, c.nop, 1, 2},        // 0x42
		{"", Implied, c.nop, 1, 2},        // 0x43
		{"", Implied, c.nop, 1, 2},        // 0x44
		{"EOR", Zeropage, c.eor, 2, 3},    // 0x45
		{"LSR", Zeropage, c.lsr, 2, 5},    // 0x46
		{"", Implied, c.nop, 1, 2},        // 0x47
		{"PHA", Implied, c.pha, 1, 3},     // 0x48
		{"EOR", Immediate, c.eor, 2, 2},   // 0x49
		{"LSR", Accumulator, c.lsr, 1, 2}, // 0x4A
		{"", Implied, c.nop, 1, 2},        // 0x4B
		{"JMP", Absolute, c.jmp, 3, 3},    // 0x4C
		{"EOR", Absolute, c.eor, 3, 4},    // 0x4D
		{"LSR", Absolute, c.lsr, 3, 6},    // 0x4E
		{"", Implied, c.nop, 1, 2},        // 0x4F
		{"BVC", Relative, c.bvc, 2, 2},    // 0x50
		{"EOR", IndirectY, c.eor, 2, 5},   // 0x51
		{"", Implied, c.nop, 1, 2},        // 0x52
		{"", Implied, c.nop, 1, 2},        // 0x53
		{"", Implied, c.nop, 1, 2},        // 0x54
		{"EOR", ZeropageX, c.eor, 2, 4},   // 0x55
		{"", ZeropageX, c.nop, 2, 6},      // 0x56
		{"", Implied, c.nop, 1, 2},        // 0x57
		{"CLI", Implied, c.cli, 1, 2},     // 0x58
		{"EOR", AbsoluteY, c.eor, 3, 4},   // 0x59
		{"", Implied, c.nop, 1, 2},        // 0x5A
		{"", Implied, c.nop, 1, 2},        // 0x5B
		{"", Implied, c.nop, 1, 2},        // 0x5C
		{"EOR", AbsoluteX, c.eor, 3, 4},   // 0x5D
		{"LSR", AbsoluteX, c.lsr, 3, 7},   // 0x5E
		{"", Implied, c.nop, 1, 2},        // 0x5F
		{"RTS", Implied, c.rts, 1, 6},     // 0x60
		{"ADC", IndirectX, c.adc, 2, 6},   // 0x61
		{"", Implied, c.nop, 1, 2},        // 0x62
		{"", Implied, c.nop, 1, 2},        // 0x63
		{"", Implied, c.nop, 1, 2},        // 0x64
		{"ADC", Zeropage, c.adc, 2, 3},    // 0x65
		{"ROR", Zeropage, c.ror, 2, 5},    // 0x66
		{"", Implied, c.nop, 1, 2},        // 0x67
		{"PLA", Implied, c.pla, 1, 4},     // 0x68
		{"ADC", Immediate, c.adc, 2, 2},   // 0x69
		{"ROR", Accumulator, c.ror, 1, 2}, // 0x6A
		{"", Implied, c.nop, 1, 2},        // 0x6B
		{"JMP", Indirect, c.jmp, 3, 5},    // 0x6C
		{"ADC", Absolute, c.adc, 3, 4},    // 0x6D
		{"ROR", Absolute, c.ror, 3, 6},    // 0x6E
		{"", Implied, c.nop, 1, 2},        // 0x6F
		{"BVS", Relative, c.bvs, 2, 2},    // 0x70
		{"ADC", IndirectY, c.adc, 2, 5},   // 0x71
		{"", Implied, c.nop, 1, 2},        // 0x72
		{"", Implied, c.nop, 1, 2},        // 0x73
		{"", Implied, c.nop, 1, 2},        // 0x74
		{"ADC", ZeropageX, c.adc, 2, 4},   // 0x75
		{"ROR", ZeropageX, c.ror, 2, 6},   // 0x76
		{"", Implied, c.nop, 1, 2},        // 0x77
		{"SEI", Implied, c.sei, 1, 2},     // 0x78
		{"ADC", AbsoluteY, c.adc, 3, 4},   // 0x79
		{"", Implied, c.nop, 1, 2},        // 0x7A
		{"", Implied, c.nop, 1, 2},        // 0x7B
		{"", Implied, c.nop, 1, 2},        // 0x7C
		{"ADC", AbsoluteX, c.adc, 3, 4},   // 0x7D
		{"ROR", AbsoluteX, c.ror, 3, 7},   // 0x7E
		{"", Implied, c.nop, 1, 2},        // 0x7F
		{"", Implied, c.nop, 1, 2},        // 0x80
		{"STA", IndirectX, c.sta, 2, 6},   // 0x81
		{"", Implied, c.nop, 1, 2},        // 0x82
		{"", Implied, c.nop, 1, 2},        // 0x83
		{"STY", Zeropage, c.sty, 2, 3},    // 0x84
		{"STA", Zeropage, c.sta, 2, 3},    // 0x85
		{"STX", Zeropage, c.stx, 2, 3},    // 0x86
		{"", Implied, c.nop, 1, 2},        // 0x87
		{"DEY", Implied, c.dey, 1, 2},     // 0x88
		{"", Implied, c.nop, 1, 2},        // 0x89
		{"TXA", Implied, c.txa, 1, 2},     // 0x8A
		{"", Implied, c.nop, 1, 2},        // 0x8B
		{"STY", Absolute, c.sty, 3, 4},    // 0x8C
		{"STA", Absolute, c.sta, 3, 4},    // 0x8D
		{"STX", Absolute, c.stx, 3, 4},    // 0x8E
		{"", Implied, c.nop, 1, 2},        // 0x8F
		{"BCC", Relative, c.bcc, 2, 2},    // 0x90
		{"STA", IndirectY, c.sta, 2, 6},   // 0x91
		{"", Implied, c.nop, 1, 2},        // 0x92
		{"", Implied, c.nop, 1, 2},        // 0x93
		{"STY", ZeropageX, c.sty, 2, 4},   // 0x94
		{"STA", ZeropageX, c.sta, 2, 4},   // 0x95
		{"STX", ZeropageY, c.stx, 2, 4},   // 0x96
		{"", Implied, c.nop, 1, 2},        // 0x97
		{"TYA", Implied, c.tya, 1, 2},     // 0x98
		{"STA", AbsoluteY, c.sta, 3, 5},   // 0x99
		{"TSX", Implied, c.tsx, 1, 2},     // 0x9A
		{"", Implied, c.nop, 1, 2},        // 0x9B
		{"", Implied, c.nop, 1, 2},        // 0x9C
		{"STA", AbsoluteX, c.sta, 3, 5},   // 0x9D
		{"", Implied, c.nop, 1, 2},        // 0x9E
		{"", Implied, c.nop, 1, 2},        // 0x9F
		{"LDY", Immediate, c.ldy, 2, 2},   // 0xA0
		{"LDA", IndirectX, c.lda, 2, 6},   // 0xA1
		{"LDX", Immediate, c.ldx, 2, 2},   // 0xA2
		{"", Implied, c.nop, 1, 2},        // 0xA3
		{"LDY", Zeropage, c.ldy, 2, 3},    // 0xA4
		{"LDA", Zeropage, c.lda, 2, 2},    // 0xA5
		{"LDX", Zeropage, c.ldx, 2, 3},    // 0xA6
		{"", Implied, c.nop, 1, 2},        // 0xA7
		{"TAY", Implied, c.tay, 1, 2},     // 0xA8
		{"LDA", Immediate, c.lda, 2, 2},   // 0xA9
		{"TAX", Implied, c.tax, 1, 2},     // 0xAA
		{"", Implied, c.nop, 1, 2},        // 0xAB
		{"LDY", Absolute, c.ldy, 3, 4},    // 0xAC
		{"LDA", Absolute, c.lda, 3, 4},    // 0xAD
		{"LDX", Absolute, c.ldx, 3, 4},    // 0xAE
		{"", Implied, c.nop, 1, 2},        // 0xAF
		{"BCS", Relative, c.bcs, 2, 2},    // 0xB0
		{"LDA", IndirectY, c.lda, 2, 5},   // 0xB1
		{"", Implied, c.nop, 1, 2},        // 0xB2
		{"", Implied, c.nop, 1, 2},        // 0xB3
		{"LDX", ZeropageX, c.ldx, 2, 4},   // 0xB4
		{"LDA", ZeropageX, c.lda, 2, 4},   // 0xB5
		{"LDX", ZeropageY, c.ldx, 2, 4},   // 0xB6
		{"", Implied, c.nop, 1, 2},        // 0xB7
		{"CLV", Implied, c.clv, 1, 2},     // 0xB8
		{"LDA", AbsoluteY, c.lda, 3, 4},   // 0xB9
		{"TSX", Implied, c.tsx, 1, 2},     // 0xBA
		{"", Implied, c.nop, 1, 2},        // 0xBB
		{"LDY", AbsoluteX, c.ldy, 3, 4},   // 0xBC
		{"LDA", AbsoluteX, c.lda, 3, 4},   // 0xBD
		{"LDX", AbsoluteY, c.ldx, 3, 4},   // 0xBE
		{"", Implied, c.nop, 1, 2},        // 0xBF
		{"CPY", Immediate, c.cpy, 2, 2},   // 0xC0
		{"CMP", IndirectX, c.cmp, 2, 6},   // 0xC1
		{"", Implied, c.nop, 1, 2},        // 0xC2
		{"", Implied, c.nop, 1, 2},        // 0xC3
		{"CPY", Zeropage, c.cpy, 2, 3},    // 0xC4
		{"CMP", Zeropage, c.cmp, 2, 3},    // 0xC5
		{"DEC", Zeropage, c.dec, 2, 5},    // 0xC6
		{"", Implied, c.nop, 1, 2},        // 0xC7
		{"INY", Implied, c.iny, 1, 2},     // 0xC8
		{"CMP", Immediate, c.cmp, 2, 2},   // 0xC9
		{"DEX", Implied, c.dex, 1, 2},     // 0xCA
		{"", Implied, c.nop, 1, 2},        // 0xCB
		{"CPY", Absolute, c.cpy, 3, 4},    // 0xCC
		{"CMP", Absolute, c.cmp, 3, 4},    // 0xCD
		{"DEC", Absolute, c.dec, 3, 6},    // 0xCE
		{"", Implied, c.nop, 1, 2},        // 0xCF
		{"BNE", Relative, c.bne, 2, 2},    // 0xD0
		{"CMP", IndirectY, c.cmp, 2, 5},   // 0xD1
		{"", Implied, c.nop, 1, 2},        // 0xD2
		{"", Implied, c.nop, 1, 2},        // 0xD3
		{"", Implied, c.nop, 1, 2},        // 0xD4
		{"CMP", ZeropageX, c.cmp, 2, 4},   // 0xD5
		{"DEC", ZeropageX, c.dec, 2, 6},   // 0xD6
		{"", Implied, c.nop, 1, 2},        // 0xD7
		{"CLD", Implied, c.cld, 1, 2},     // 0xD8
		{"CMP", AbsoluteY, c.cmp, 3, 4},   // 0xD9
		{"", Implied, c.nop, 1, 2},        // 0xDA
		{"", Implied, c.nop, 1, 2},        // 0xDB
		{"", Implied, c.nop, 1, 2},        // 0xDC
		{"CMP", AbsoluteX, c.cmp, 3, 4},   // 0xDD
		{"DEC", AbsoluteX, c.dec, 3, 7},   // 0xDE
		{"", Implied, c.nop, 1, 2},        // 0xDF
		{"CPX", Immediate, c.cpx, 2, 2},   // 0xE0
		{"SBC", IndirectX, c.sbc, 2, 6},   // 0xE1
		{"", Implied, c.nop, 1, 2},        // 0xE2
		{"", Implied, c.nop, 1, 2},        // 0xE3
		{"CPX", Zeropage, c.cpx, 2, 3},    // 0xE4
		{"SBC", Zeropage, c.sbc, 2, 3},    // 0xE5
		{"INC", Zeropage, c.inc, 2, 5},    // 0xE6
		{"", Implied, c.nop, 1, 2},        // 0xE7
		{"INX", Implied, c.inx, 1, 2},     // 0xE8
		{"SBC", Immediate, c.sbc, 2, 2},   // 0xE9
		{"NOP", Implied, c.nop, 1, 2},     // 0xEA
		{"", Implied, c.nop, 1, 2},        // 0xEB
		{"CPX", Absolute, c.cpx, 3, 4},    // 0xEC
		{"SBC", Absolute, c.sbc, 3, 4},    // 0xED
		{"INC", Absolute, c.inc, 3, 6},    // 0xEE
		{"", Implied, c.nop, 1, 2},        // 0xEF
		{"BEQ", Relative, c.beq, 2, 2},    // 0xF0
		{"SBC", IndirectY, c.sbc, 2, 5},   // 0xF1
		{"", Implied, c.nop, 1, 2},        // 0xF2
		{"", Implied, c.nop, 1, 2},        // 0xF3
		{"", Implied, c.nop, 1, 2},        // 0xF4
		{"SBC", ZeropageX, c.sbc, 2, 4},   // 0xF5
		{"INC", ZeropageX, c.inc, 2, 6},   // 0xF6
		{"", Implied, c.nop, 1, 2},        // 0xF7
		{"SED", Implied, c.sed, 1, 2},     // 0xF8
		{"SBC", AbsoluteY, c.sbc, 3, 4},   // 0xF9
		{"", Implied, c.nop, 1, 2},        // 0xFA
		{"", Implied, c.nop, 1, 2},        // 0xFB
		{"", Implied, c.nop, 1, 2},        // 0xFC
		{"SBC", AbsoluteX, c.sbc, 3, 4},   // 0xFD
		{"INC", AbsoluteX, c.inc, 3, 7},   // 0xFE
		{"", Implied, c.nop, 1, 2},        // 0xFF
	}
}

// NewCPU creates a new NES CPU.
func NewCPU(bus *CPUBus) *CPU {
	c := &CPU{
		P: &status{
			C: false,
			Z: false,
			I: false,
			D: false,
			B: true,
			R: true,
			V: false,
			N: false,
		},
		A:   0,
		X:   0,
		Y:   0,
		PC:  0,
		S:   0,
		bus: bus,
	}
	c.instructions = c.createInstructions()
	c.Reset()
	return c
}

// Reset does reset.
func (c *CPU) Reset() {
	glog.Infoln("CPU Reset!")
	c.PC = c.bus.read16(0xFFFC)
	c.S = 0xFD
	c.P.decodeFrom(0x24)
}

// setN sets whether the x is negative or positive.
func (c *CPU) setN(x byte) {
	c.P.N = x&0x80 != 0
}

// setZ sets whether the x is 0 or not.
func (c *CPU) setZ(x byte) {
	c.P.Z = x == 0
}

// push pushes data to stack.
// "With the 6502, the stack is always on page one ($100-$1FF) and works top down."
func (c *CPU) push(x byte) {
	c.bus.write((0x100 | (uint16(c.S) & 0xFF)), x)
	c.S--
}

// pop pops data from stack.
// "With the 6502, the stack is always on page one ($100-$1FF) and works top down."
func (c *CPU) pop() byte {
	c.S++
	return c.bus.read((0x100 | (uint16(c.S) & 0xFF)))
}

// ADC - Add with Carry.
func (c *CPU) adc(mode addressingMode, operand uint16) {
	x := uint16(c.A)
	y := uint16(c.bus.read(operand))
	var carry uint16 = 0
	if c.P.C {
		carry = 1
	}
	res := x + y + carry
	if res > 0xFF {
		c.P.C = true
		c.A = byte(res & 0xFF)
	} else {
		c.P.C = false
		c.A = byte(res)
	}
	c.setN(c.A)
	c.setZ(c.A)
	// checks whether the value overflown by xor.
	if x^y&0x80 != 0 && x^res&0x80 != 0 {
		c.P.V = true
	} else {
		c.P.V = false
	}
}

// AND - And.
func (c *CPU) and(mode addressingMode, operand uint16) {
	c.A = c.A & c.bus.read(operand)
	c.setN(c.A)
	c.setZ(c.A)
}

// ASL - Arithmetic Shift Left.
func (c *CPU) asl(mode addressingMode, operand uint16) {
	if mode == Accumulator {
		c.P.C = (c.A>>7)&1 == 1
		c.A <<= 1
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x := c.bus.read(operand)
		c.P.C = (x>>7)&1 == 1
		x <<= 1
		c.bus.write(operand, x)
		c.setN(x)
		c.setZ(x)
	}
}

// BCC - Branch on Carry Clear.
func (c *CPU) bcc(mode addressingMode, operand uint16) {
	if !c.P.C {
		c.PC = operand
	}
}

// BCS - Branch on Carry Set.
func (c *CPU) bcs(mode addressingMode, operand uint16) {
	if c.P.C {
		c.PC = operand
	}
}

// BEQ - Branch on Equal.
func (c *CPU) beq(mode addressingMode, operand uint16) {
	if c.P.Z {
		c.PC = operand
	}
}

// BIT - test BITS.
func (c *CPU) bit(mode addressingMode, operand uint16) {
	x := c.bus.read(operand)
	c.setN(x)
	c.setZ(c.A & x)
	c.P.V = (x>>6)&1 == 1
}

// BMI - Branch on Minus.
func (c *CPU) bmi(mode addressingMode, operand uint16) {
	if c.P.N {
		c.PC = operand
	}
}

// BNE - Branch on Not Equal.
func (c *CPU) bne(mode addressingMode, operand uint16) {
	if !c.P.Z {
		c.PC = operand
	}
}

// BPL - Branch on Plus.
func (c *CPU) bpl(mode addressingMode, operand uint16) {
	if !c.P.N {
		c.PC = operand
	}
}

// BRK - Break Interrupt.
func (c *CPU) brk(mode addressingMode, operand uint16) {
	c.push(byte(c.PC>>8) & 0xFF)
	c.push(byte(c.PC & 0xFF))
	c.push(c.P.encode())
	c.P.I = true
	c.PC = c.bus.read16(0xFFFE)
}

// BVC - Branch on Overflow Clear.
func (c *CPU) bvc(mode addressingMode, operand uint16) {
	if !c.P.V {
		c.PC = operand
	}
}

// BVS - Branch on Overflow Set.
func (c *CPU) bvs(mode addressingMode, operand uint16) {
	if c.P.V {
		c.PC = operand
	}
}

// CLC - Clear Carry.
func (c *CPU) clc(mode addressingMode, operand uint16) {
	c.P.C = false
}

// CLD - Clear Decimal.
func (c *CPU) cld(mode addressingMode, operand uint16) {
	// Not implemented on NES
}

// CLI - Clear Interrupt.
func (c *CPU) cli(mode addressingMode, operand uint16) {
	c.P.I = false
}

// CLV - Clear Overflow.
func (c *CPU) clv(mode addressingMode, operand uint16) {
	c.P.V = false
}

// CMP - Compare Accumulator.
func (c *CPU) cmp(mode addressingMode, operand uint16) {
	x := c.A - c.bus.read(operand)
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
}

// CPX - Compare X register.
func (c *CPU) cpx(mode addressingMode, operand uint16) {
	x := c.X - c.bus.read(operand)
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
}

// CPY - Compare Y register.
func (c *CPU) cpy(mode addressingMode, operand uint16) {
	x := c.Y - c.bus.read(operand)
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
}

// DEC - Decrement Memory.
func (c *CPU) dec(mode addressingMode, operand uint16) {
	x := c.bus.read(operand) - 1 // this won't go negative.
	c.bus.write(operand, x)
	c.setN(x)
	c.setZ(x)
}

// DEX - Decrement X Register.
func (c *CPU) dex(mode addressingMode, operand uint16) {
	c.X--
	c.setN(c.X)
	c.setZ(c.X)
}

// DEY - Decrement Y Register.
func (c *CPU) dey(mode addressingMode, operand uint16) {
	c.Y--
	c.setN(c.Y)
	c.setZ(c.Y)
}

// EOR - Bitwise Exclusive OR.
func (c *CPU) eor(mode addressingMode, operand uint16) {
	c.A = c.A ^ c.bus.read(operand)
	c.setN(c.A)
	c.setZ(c.A)
}

// INC - Increment Memory.
func (c *CPU) inc(mode addressingMode, operand uint16) {
	x := c.bus.read(operand)
	x--
	c.bus.write(operand, x)
	c.setN(x)
	c.setZ(x)
}

// INX - Increment X Register.
func (c *CPU) inx(mode addressingMode, operand uint16) {
	c.X++
	c.setN(c.X)
	c.setZ(c.X)
}

// INY - Increment Y Register.
func (c *CPU) iny(mode addressingMode, operand uint16) {
	c.Y++
	c.setN(c.Y)
	c.setZ(c.Y)
}

// JMP - Jump.
func (c *CPU) jmp(mode addressingMode, operand uint16) {
	c.PC = operand
}

// JSR - Jump to Subroutine.
func (c *CPU) jsr(mode addressingMode, operand uint16) {
	x := c.PC - 1
	c.push(byte(x>>8) & 0xFF)
	c.push(byte(x & 0xFF))
	c.PC = operand
}

// LDA - Load Accumulator.
func (c *CPU) lda(mode addressingMode, operand uint16) {
	c.A = c.bus.read(operand)
	c.setN(c.A)
	c.setZ(c.A)
}

// LDX - Load X Register.
func (c *CPU) ldx(mode addressingMode, operand uint16) {
	c.X = c.bus.read(operand)
	c.setN(c.X)
	c.setZ(c.X)
}

// LDY - Load Y Register.
func (c *CPU) ldy(mode addressingMode, operand uint16) {
	c.Y = c.bus.read(operand)
	c.setN(c.Y)
	c.setZ(c.Y)
}

// LSR - Logical Shift Right.
func (c *CPU) lsr(mode addressingMode, operand uint16) {
	if mode == Accumulator {
		c.P.C = c.A&1 == 1
		c.A >>= 1
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x := c.bus.read(operand)
		c.P.C = x&1 == 1
		x >>= 1
		c.bus.write(operand, x)
		c.setN(x)
		c.setZ(x)
	}
}

// NOP - No Operation.
func (c *CPU) nop(mode addressingMode, operand uint16) {
	// noop
}

// ORA - Bitwise OR with Accumulator.
func (c *CPU) ora(mode addressingMode, operand uint16) {
	c.A = c.A | c.bus.read(operand)
	c.setN(c.A)
	c.setZ(c.A)
}

// PHA - Push Accumulator.
func (c *CPU) pha(mode addressingMode, operand uint16) {
	c.push(c.A)
}

// PHP - Push Processor Status.
func (c *CPU) php(mode addressingMode, operand uint16) {
	c.push(c.P.encode())
}

// PLA - Pull Accumulator.
func (c *CPU) pla(mode addressingMode, operand uint16) {
	c.A = c.pop()
	c.setN(c.A)
	c.setZ(c.A)
}

// PLP - Pull Processor Status.
func (c *CPU) plp(mode addressingMode, operand uint16) {
	c.P.decodeFrom(c.pop())
}

// ROL - Rotate Left.
func (c *CPU) rol(mode addressingMode, operand uint16) {
	var carry byte = 0
	if c.P.C {
		carry = 1
	}
	if mode == Accumulator {
		c.P.C = (c.A>>7)&1 == 1
		c.A = (c.A << 1) | carry
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x := c.bus.read(operand)
		c.P.C = (x>>7)&1 == 1
		x = (x << 1) | carry
		c.bus.write(operand, x)
		c.setN(x)
		c.setZ(x)
	}
}

// ROR - Rotate Right.
func (c *CPU) ror(mode addressingMode, operand uint16) {
	var carry byte = 0
	if c.P.C {
		carry = 1
	}
	if mode == Accumulator {
		c.P.C = c.A&1 == 1
		c.A = (c.A >> 1) | (carry << 7)
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x := c.bus.read(operand)
		c.P.C = x&1 == 1
		x = (x >> 1) | (carry << 7)
		c.bus.write(operand, x)
		c.setN(x)
		c.setZ(x)
	}
}

// RTS - Return from Subroutine.
func (c *CPU) rts(mode addressingMode, operand uint16) {
	l := uint16(c.pop())
	h := uint16(c.pop()) >> 8
	c.PC = h | l
}

// RTI - Return from Interrupt.
func (c *CPU) rti(mode addressingMode, operand uint16) {
	c.P.decodeFrom(c.pop())
	l := uint16(c.pop())
	h := uint16(c.pop()) >> 8
	c.PC = h | l
}

// SBC - Subtract with carry.
func (c *CPU) sbc(mode addressingMode, operand uint16) {
	x := int16(c.A)
	y := int16(c.bus.read(operand))
	var carry int16 = 0
	if c.P.C {
		carry = 1
	}
	res := x - y - (1 - carry)
	if res < 0 {
		c.P.C = true
		c.A = 0
	} else {
		c.P.C = false
		c.A = byte(res)
	}
	c.setN(c.A)
	c.setZ(c.A)
	// checks whether the value overflown by xor.
	if x^y&0x80 != 0 && x^res&0x80 != 0 {
		c.P.V = true
	} else {
		c.P.V = false
	}
}

// SEC - Set Carry.
func (c *CPU) sec(mode addressingMode, operand uint16) {
	c.P.C = true
}

// SED - Set Carry.
func (c *CPU) sed(mode addressingMode, operand uint16) {
	// Not implemented on NES.
}

// SEI - Set Interrupt.
func (c *CPU) sei(mode addressingMode, operand uint16) {
	c.P.I = true
}

// STA - Store A Register.
func (c *CPU) sta(mode addressingMode, operand uint16) {
	c.bus.write(operand, c.A)
}

// STX - Store X Register.
func (c *CPU) stx(mode addressingMode, operand uint16) {
	c.bus.write(operand, c.X)
}

// STY - Store Y Register.
func (c *CPU) sty(mode addressingMode, operand uint16) {
	c.bus.write(operand, c.Y)
}

// TAX - Transfer A to X.
func (c *CPU) tax(mode addressingMode, operand uint16) {
	c.X = c.A
	c.setN(c.A)
	c.setZ(c.A)
}

// TAY - Transfer A to Y.
func (c *CPU) tay(mode addressingMode, operand uint16) {
	c.Y = c.A
	c.setN(c.A)
	c.setZ(c.A)
}

// TSX - Transfer S to X.
func (c *CPU) tsx(mode addressingMode, operand uint16) {
	c.X = c.S
	c.setN(c.S)
	c.setZ(c.S)
}

// TXA - Transfer X to A.
func (c *CPU) txa(mode addressingMode, operand uint16) {
	c.A = c.X
	c.setN(c.X)
	c.setZ(c.X)
}

// TXS - Transfer X to S.
func (c *CPU) txs(mode addressingMode, operand uint16) {
	c.S = c.X
	c.setN(c.X)
	c.setZ(c.X)
}

// TYA - Transfer Y to A.
func (c *CPU) tya(mode addressingMode, operand uint16) {
	c.A = c.Y
	c.setN(c.Y)
	c.setZ(c.Y)
}

// Do performs the instruction cycle - fetch, decode, execute.
func (c *CPU) Do() int {
	opcode := c.bus.read(c.PC)
	instruction := c.instructions[opcode]
	var operand uint16 = 0
	switch instruction.mode {
	case Implied:
		operand = 0
	case Accumulator:
		operand = 0
	case Immediate:
		operand = c.PC + 1
	case Zeropage:
		operand = uint16(c.bus.read(c.PC + 1))
	case ZeropageX:
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(c.bus.read(c.PC+1)+c.X) & 0xFF
	case ZeropageY:
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(c.bus.read(c.PC+1)+c.Y) & 0xFF
	case Relative:
		address := c.bus.read(c.PC + 1)
		// Relative will look up a signed value
		// 2 is offset for operand
		if address < 0x80 {
			operand = c.PC + 2 + uint16(address)
		} else {
			operand = c.PC + 2 + uint16(address) - 0x100
		}
	case Absolute:
		operand = c.bus.read16(c.PC + 1)
	case AbsoluteX:
		operand = c.bus.read16(c.PC+1) + uint16(c.X)
	case AbsoluteY:
		operand = c.bus.read16(c.PC+1) + uint16(c.Y)
	case Indirect:
		operand = c.bus.read16(c.bus.read16(c.PC + 1))
	case IndirectX:
		operand = uint16(c.bus.read(c.PC+1)) + uint16(c.X)
	case IndirectY:
		operand = uint16(c.bus.read(c.PC+1)) + uint16(c.Y)
	}
	c.PC += instruction.size
	glog.V(1).Infof("PC: 0x%04x, A: 0x%02x, X: 0x%02x, Y: 0x%02x, S: 0x%02x opcode: 0x%02x, mnemonic: %s, operand: 0x%04x\n",
		c.PC, c.A, c.X, c.Y, c.S, opcode, instruction.mnemonic, operand)
	instruction.execute(instruction.mode, operand)
	return instruction.cycles
}
