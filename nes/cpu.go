package nes

import "fmt"

// CPU emulates NES CPU - is custom 6502 made by RICOH.
// References:
//   https://en.wikipedia.org/wiki/MOS_Technology_6502
//   http://www.6502.org/tutorials/6502opcodes.html
//   http://hp.vector.co.jp/authors/VA042397/nes/6502.html (In Japanese)

const CPUFrequency = 1789773

type addressingMode int

const (
	implied addressingMode = iota
	accumulator
	immdiate
	zeropage
	zeropageX
	zeropageY
	relative
	absolute
	absoluteX
	absoluteY
	indirect
	indirectX
	indirectY
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
	P             *status // Processor status flag bits
	A             byte    // Accumulator register
	X             byte    // Index register
	Y             byte    // Index register
	PC            uint16  // Program counter
	S             byte    // Stack pointer
	lastExecution string  // For debug
	stall         uint64  // Stall cycles
	bus           *CPUBus
	instructions  []instruction
	nmiTriggered  bool
}

type instruction struct {
	mnemonic string
	mode     addressingMode
	execute  func(addressingMode, uint16) error
	size     uint16
	cycles   int
}

func (c *CPU) createInstructions() []instruction {
	return []instruction{
		{"BRK", implied, c.brk, 1, 7},     // 0x00
		{"ORA", indirectX, c.ora, 2, 6},   // 0x01
		{"", implied, c.err, 1, 2},        // 0x02
		{"", implied, c.err, 1, 2},        // 0x03
		{"", implied, c.err, 1, 2},        // 0x04
		{"ORA", zeropage, c.ora, 2, 3},    // 0x05
		{"ASL", zeropage, c.asl, 2, 5},    // 0x06
		{"", implied, c.err, 1, 2},        // 0x07
		{"PHP", implied, c.php, 1, 3},     // 0x08
		{"ORA", immdiate, c.ora, 2, 2},    // 0x09
		{"ASL", accumulator, c.asl, 1, 2}, // 0x0A
		{"", implied, c.err, 1, 2},        // 0x0B
		{"", implied, c.err, 1, 2},        // 0x0C
		{"ORA", absolute, c.ora, 3, 4},    // 0x0D
		{"ASL", absolute, c.asl, 3, 6},    // 0x0E
		{"", implied, c.err, 1, 2},        // 0x0F
		{"BPL", relative, c.bpl, 2, 2},    // 0x10
		{"ORA", indirectY, c.ora, 2, 5},   // 0x11
		{"", implied, c.err, 1, 2},        // 0x12
		{"", implied, c.err, 1, 2},        // 0x13
		{"", implied, c.err, 1, 2},        // 0x14
		{"ORA", zeropageX, c.ora, 2, 4},   // 0x15
		{"ASL", zeropageX, c.asl, 2, 6},   // 0x16
		{"", implied, c.err, 1, 2},        // 0x17
		{"CLC", implied, c.clc, 1, 2},     // 0x18
		{"ORA", absoluteY, c.ora, 3, 4},   // 0x19
		{"", implied, c.err, 1, 2},        // 0x1A
		{"", implied, c.err, 1, 2},        // 0x1B
		{"", implied, c.err, 1, 2},        // 0x1C
		{"ORA", absoluteX, c.ora, 3, 4},   // 0x1D
		{"ASL", absoluteX, c.asl, 3, 7},   // 0x1E
		{"", implied, c.err, 1, 2},        // 0x1F
		{"JSR", absolute, c.jsr, 3, 6},    // 0x20
		{"AND", indirectX, c.and, 2, 6},   // 0x21
		{"", implied, c.err, 1, 2},        // 0x22
		{"", implied, c.err, 1, 2},        // 0x23
		{"BIT", zeropage, c.bit, 2, 3},    // 0x24
		{"AND", zeropage, c.and, 2, 3},    // 0x25
		{"ROL", zeropage, c.rol, 2, 5},    // 0x26
		{"", implied, c.err, 1, 2},        // 0x27
		{"PLP", implied, c.plp, 1, 4},     // 0x28
		{"AND", immdiate, c.and, 2, 2},    // 0x29
		{"ROL", accumulator, c.rol, 1, 2}, // 0x2A
		{"", implied, c.err, 1, 2},        // 0x2B
		{"BIT", absolute, c.bit, 3, 4},    // 0x2C
		{"AND", absolute, c.and, 3, 4},    // 0x2D
		{"ROL", absolute, c.rol, 3, 6},    // 0x2E
		{"", implied, c.err, 1, 2},        // 0x2F
		{"BMI", relative, c.bmi, 2, 2},    // 0x30
		{"AND", indirectY, c.and, 2, 5},   // 0x31
		{"", implied, c.err, 1, 2},        // 0x32
		{"", implied, c.err, 1, 2},        // 0x33
		{"", implied, c.err, 1, 2},        // 0x34
		{"AND", zeropageX, c.and, 2, 4},   // 0x35
		{"ROL", zeropageX, c.rol, 2, 6},   // 0x36
		{"", implied, c.err, 1, 2},        // 0x37
		{"SEC", implied, c.sec, 1, 2},     // 0x38
		{"AND", absoluteY, c.and, 3, 4},   // 0x39
		{"", implied, c.err, 1, 2},        // 0x3A
		{"", implied, c.err, 1, 2},        // 0x3B
		{"", implied, c.err, 1, 2},        // 0x3C
		{"AND", absoluteX, c.and, 3, 4},   // 0x3D
		{"ROL", absoluteX, c.rol, 3, 7},   // 0x3E
		{"", implied, c.err, 1, 2},        // 0x3F
		{"RTI", implied, c.rti, 1, 6},     // 0x40
		{"EOR", indirectX, c.eor, 2, 6},   // 0x41
		{"", implied, c.err, 1, 2},        // 0x42
		{"", implied, c.err, 1, 2},        // 0x43
		{"", implied, c.err, 1, 2},        // 0x44
		{"EOR", zeropage, c.eor, 2, 3},    // 0x45
		{"LSR", zeropage, c.lsr, 2, 5},    // 0x46
		{"", implied, c.err, 1, 2},        // 0x47
		{"PHA", implied, c.pha, 1, 3},     // 0x48
		{"EOR", immdiate, c.eor, 2, 2},    // 0x49
		{"LSR", accumulator, c.lsr, 1, 2}, // 0x4A
		{"", implied, c.err, 1, 2},        // 0x4B
		{"JMP", absolute, c.jmp, 3, 3},    // 0x4C
		{"EOR", absolute, c.eor, 3, 4},    // 0x4D
		{"LSR", absolute, c.lsr, 3, 6},    // 0x4E
		{"", implied, c.err, 1, 2},        // 0x4F
		{"BVC", relative, c.bvc, 2, 2},    // 0x50
		{"EOR", indirectY, c.eor, 2, 5},   // 0x51
		{"", implied, c.err, 1, 2},        // 0x52
		{"", implied, c.err, 1, 2},        // 0x53
		{"", implied, c.err, 1, 2},        // 0x54
		{"EOR", zeropageX, c.eor, 2, 4},   // 0x55
		{"", zeropageX, c.err, 2, 6},      // 0x56
		{"", implied, c.err, 1, 2},        // 0x57
		{"CLI", implied, c.cli, 1, 2},     // 0x58
		{"EOR", absoluteY, c.eor, 3, 4},   // 0x59
		{"", implied, c.err, 1, 2},        // 0x5A
		{"", implied, c.err, 1, 2},        // 0x5B
		{"", implied, c.err, 1, 2},        // 0x5C
		{"EOR", absoluteX, c.eor, 3, 4},   // 0x5D
		{"LSR", absoluteX, c.lsr, 3, 7},   // 0x5E
		{"", implied, c.err, 1, 2},        // 0x5F
		{"RTS", implied, c.rts, 1, 6},     // 0x60
		{"ADC", indirectX, c.adc, 2, 6},   // 0x61
		{"", implied, c.err, 1, 2},        // 0x62
		{"", implied, c.err, 1, 2},        // 0x63
		{"", implied, c.err, 1, 2},        // 0x64
		{"ADC", zeropage, c.adc, 2, 3},    // 0x65
		{"ROR", zeropage, c.ror, 2, 5},    // 0x66
		{"", implied, c.err, 1, 2},        // 0x67
		{"PLA", implied, c.pla, 1, 4},     // 0x68
		{"ADC", immdiate, c.adc, 2, 2},    // 0x69
		{"ROR", accumulator, c.ror, 1, 2}, // 0x6A
		{"", implied, c.err, 1, 2},        // 0x6B
		{"JMP", indirect, c.jmp, 3, 5},    // 0x6C
		{"ADC", absolute, c.adc, 3, 4},    // 0x6D
		{"ROR", absolute, c.ror, 3, 6},    // 0x6E
		{"", implied, c.err, 1, 2},        // 0x6F
		{"BVS", relative, c.bvs, 2, 2},    // 0x70
		{"ADC", indirectY, c.adc, 2, 5},   // 0x71
		{"", implied, c.err, 1, 2},        // 0x72
		{"", implied, c.err, 1, 2},        // 0x73
		{"", implied, c.err, 1, 2},        // 0x74
		{"ADC", zeropageX, c.adc, 2, 4},   // 0x75
		{"ROR", zeropageX, c.ror, 2, 6},   // 0x76
		{"", implied, c.err, 1, 2},        // 0x77
		{"SEI", implied, c.sei, 1, 2},     // 0x78
		{"ADC", absoluteY, c.adc, 3, 4},   // 0x79
		{"", implied, c.err, 1, 2},        // 0x7A
		{"", implied, c.err, 1, 2},        // 0x7B
		{"", implied, c.err, 1, 2},        // 0x7C
		{"ADC", absoluteX, c.adc, 3, 4},   // 0x7D
		{"ROR", absoluteX, c.ror, 3, 7},   // 0x7E
		{"", implied, c.err, 1, 2},        // 0x7F
		{"", implied, c.err, 1, 2},        // 0x80
		{"STA", indirectX, c.sta, 2, 6},   // 0x81
		{"", implied, c.err, 1, 2},        // 0x82
		{"", implied, c.err, 1, 2},        // 0x83
		{"STY", zeropage, c.sty, 2, 3},    // 0x84
		{"STA", zeropage, c.sta, 2, 3},    // 0x85
		{"STX", zeropage, c.stx, 2, 3},    // 0x86
		{"", implied, c.err, 1, 2},        // 0x87
		{"DEY", implied, c.dey, 1, 2},     // 0x88
		{"", implied, c.err, 1, 2},        // 0x89
		{"TXA", implied, c.txa, 1, 2},     // 0x8A
		{"", implied, c.err, 1, 2},        // 0x8B
		{"STY", absolute, c.sty, 3, 4},    // 0x8C
		{"STA", absolute, c.sta, 3, 4},    // 0x8D
		{"STX", absolute, c.stx, 3, 4},    // 0x8E
		{"", implied, c.err, 1, 2},        // 0x8F
		{"BCC", relative, c.bcc, 2, 2},    // 0x90
		{"STA", indirectY, c.sta, 2, 6},   // 0x91
		{"", implied, c.err, 1, 2},        // 0x92
		{"", implied, c.err, 1, 2},        // 0x93
		{"STY", zeropageX, c.sty, 2, 4},   // 0x94
		{"STA", zeropageX, c.sta, 2, 4},   // 0x95
		{"STX", zeropageY, c.stx, 2, 4},   // 0x96
		{"", implied, c.err, 1, 2},        // 0x97
		{"TYA", implied, c.tya, 1, 2},     // 0x98
		{"STA", absoluteY, c.sta, 3, 5},   // 0x99
		{"TSX", implied, c.tsx, 1, 2},     // 0x9A
		{"", implied, c.err, 1, 2},        // 0x9B
		{"", implied, c.err, 1, 2},        // 0x9C
		{"STA", absoluteX, c.sta, 3, 5},   // 0x9D
		{"", implied, c.err, 1, 2},        // 0x9E
		{"", implied, c.err, 1, 2},        // 0x9F
		{"LDY", immdiate, c.ldy, 2, 2},    // 0xA0
		{"LDA", indirectX, c.lda, 2, 6},   // 0xA1
		{"LDX", immdiate, c.ldx, 2, 2},    // 0xA2
		{"", implied, c.err, 1, 2},        // 0xA3
		{"LDY", zeropage, c.ldy, 2, 3},    // 0xA4
		{"LDA", zeropage, c.lda, 2, 2},    // 0xA5
		{"LDX", zeropage, c.ldx, 2, 3},    // 0xA6
		{"", implied, c.err, 1, 2},        // 0xA7
		{"TAY", implied, c.tay, 1, 2},     // 0xA8
		{"LDA", immdiate, c.lda, 2, 2},    // 0xA9
		{"TAX", implied, c.tax, 1, 2},     // 0xAA
		{"", implied, c.err, 1, 2},        // 0xAB
		{"LDY", absolute, c.ldy, 3, 4},    // 0xAC
		{"LDA", absolute, c.lda, 3, 4},    // 0xAD
		{"LDX", absolute, c.ldx, 3, 4},    // 0xAE
		{"", implied, c.err, 1, 2},        // 0xAF
		{"BCS", relative, c.bcs, 2, 2},    // 0xB0
		{"LDA", indirectY, c.lda, 2, 5},   // 0xB1
		{"", implied, c.err, 1, 2},        // 0xB2
		{"", implied, c.err, 1, 2},        // 0xB3
		{"LDX", zeropageX, c.ldx, 2, 4},   // 0xB4
		{"LDA", zeropageX, c.lda, 2, 4},   // 0xB5
		{"LDX", zeropageY, c.ldx, 2, 4},   // 0xB6
		{"", implied, c.err, 1, 2},        // 0xB7
		{"CLV", implied, c.clv, 1, 2},     // 0xB8
		{"LDA", absoluteY, c.lda, 3, 4},   // 0xB9
		{"TSX", implied, c.tsx, 1, 2},     // 0xBA
		{"", implied, c.err, 1, 2},        // 0xBB
		{"LDY", absoluteX, c.ldy, 3, 4},   // 0xBC
		{"LDA", absoluteX, c.lda, 3, 4},   // 0xBD
		{"LDX", absoluteY, c.ldx, 3, 4},   // 0xBE
		{"", implied, c.err, 1, 2},        // 0xBF
		{"CPY", immdiate, c.cpy, 2, 2},    // 0xC0
		{"CMP", indirectX, c.cmp, 2, 6},   // 0xC1
		{"", implied, c.err, 1, 2},        // 0xC2
		{"", implied, c.err, 1, 2},        // 0xC3
		{"CPY", zeropage, c.cpy, 2, 3},    // 0xC4
		{"CMP", zeropage, c.cmp, 2, 3},    // 0xC5
		{"DEC", zeropage, c.dec, 2, 5},    // 0xC6
		{"", implied, c.err, 1, 2},        // 0xC7
		{"INY", implied, c.iny, 1, 2},     // 0xC8
		{"CMP", immdiate, c.cmp, 2, 2},    // 0xC9
		{"DEX", implied, c.dex, 1, 2},     // 0xCA
		{"", implied, c.err, 1, 2},        // 0xCB
		{"CPY", absolute, c.cpy, 3, 4},    // 0xCC
		{"CMP", absolute, c.cmp, 3, 4},    // 0xCD
		{"DEC", absolute, c.dec, 3, 6},    // 0xCE
		{"", implied, c.err, 1, 2},        // 0xCF
		{"BNE", relative, c.bne, 2, 2},    // 0xD0
		{"CMP", indirectY, c.cmp, 2, 5},   // 0xD1
		{"", implied, c.err, 1, 2},        // 0xD2
		{"", implied, c.err, 1, 2},        // 0xD3
		{"", implied, c.err, 1, 2},        // 0xD4
		{"CMP", zeropageX, c.cmp, 2, 4},   // 0xD5
		{"DEC", zeropageX, c.dec, 2, 6},   // 0xD6
		{"", implied, c.err, 1, 2},        // 0xD7
		{"CLD", implied, c.cld, 1, 2},     // 0xD8
		{"CMP", absoluteY, c.cmp, 3, 4},   // 0xD9
		{"", implied, c.err, 1, 2},        // 0xDA
		{"", implied, c.err, 1, 2},        // 0xDB
		{"", implied, c.err, 1, 2},        // 0xDC
		{"CMP", absoluteX, c.cmp, 3, 4},   // 0xDD
		{"DEC", absoluteX, c.dec, 3, 7},   // 0xDE
		{"", implied, c.err, 1, 2},        // 0xDF
		{"CPX", immdiate, c.cpx, 2, 2},    // 0xE0
		{"SBC", indirectX, c.sbc, 2, 6},   // 0xE1
		{"", implied, c.err, 1, 2},        // 0xE2
		{"", implied, c.err, 1, 2},        // 0xE3
		{"CPX", zeropage, c.cpx, 2, 3},    // 0xE4
		{"SBC", zeropage, c.sbc, 2, 3},    // 0xE5
		{"INC", zeropage, c.inc, 2, 5},    // 0xE6
		{"", implied, c.err, 1, 2},        // 0xE7
		{"INX", implied, c.inx, 1, 2},     // 0xE8
		{"SBC", immdiate, c.sbc, 2, 2},    // 0xE9
		{"NOP", implied, c.nop, 1, 2},     // 0xEA
		{"", implied, c.err, 1, 2},        // 0xEB
		{"CPX", absolute, c.cpx, 3, 4},    // 0xEC
		{"SBC", absolute, c.sbc, 3, 4},    // 0xED
		{"INC", absolute, c.inc, 3, 6},    // 0xEE
		{"", implied, c.err, 1, 2},        // 0xEF
		{"BEQ", relative, c.beq, 2, 2},    // 0xF0
		{"SBC", indirectY, c.sbc, 2, 5},   // 0xF1
		{"", implied, c.err, 1, 2},        // 0xF2
		{"", implied, c.err, 1, 2},        // 0xF3
		{"", implied, c.err, 1, 2},        // 0xF4
		{"SBC", zeropageX, c.sbc, 2, 4},   // 0xF5
		{"INC", zeropageX, c.inc, 2, 6},   // 0xF6
		{"", implied, c.err, 1, 2},        // 0xF7
		{"SED", implied, c.sed, 1, 2},     // 0xF8
		{"SBC", absoluteY, c.sbc, 3, 4},   // 0xF9
		{"", implied, c.err, 1, 2},        // 0xFA
		{"", implied, c.err, 1, 2},        // 0xFB
		{"", implied, c.err, 1, 2},        // 0xFC
		{"SBC", absoluteX, c.sbc, 3, 4},   // 0xFD
		{"INC", absoluteX, c.inc, 3, 7},   // 0xFE
		{"", implied, c.err, 1, 2},        // 0xFF
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
func (c *CPU) Reset() error {
	data, err := c.bus.read16(0xFFFC)
	if err != nil {
		return err
	}
	c.PC = data
	c.S = 0xFD
	c.P.decodeFrom(0x24)
	return nil
}

// write is for wrapping c.bus.write, because writing oamdma requires some.
func (c *CPU) write(address uint16, data byte) error {
	// OAMDMA
	if address == 0x4014 {
		oamData := [256]byte{}
		offset := uint16(data) << 8
		for i := 0; i < 256; i++ {
			d, err := c.bus.read(offset + uint16(i))
			if err != nil {
				return err
			}
			oamData[i] = d
		}
		c.bus.writeOAMDMA(oamData)
		// TODO(jyane): this stall value depends on current cycle is even / odd.
		// should be like "if cycles%2 == 0 ..."
		c.stall += 514
		return nil
	} else {
		return c.bus.write(address, data)
	}
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
func (c *CPU) push(x byte) error {
	if err := c.write((0x100 | (uint16(c.S) & 0xFF)), x); err != nil {
		return err
	}
	c.S--
	return nil
}

// pop pops data from stack.
// "With the 6502, the stack is always on page one ($100-$1FF) and works top down."
func (c *CPU) pop() (byte, error) {
	c.S++
	return c.bus.read((0x100 | (uint16(c.S) & 0xFF)))
}

func (c *CPU) err(mode addressingMode, operand uint16) error {
	return fmt.Errorf("Unimplemented CPU instruction was executed.")
}

// ADC - Add with Carry.
func (c *CPU) adc(mode addressingMode, operand uint16) error {
	x := uint16(c.A)
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	y := uint16(data)
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
	return nil
}

// AND - And.
func (c *CPU) and(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.A = c.A & data
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// ASL - Arithmetic Shift Left.
func (c *CPU) asl(mode addressingMode, operand uint16) error {
	if mode == accumulator {
		c.P.C = (c.A>>7)&1 == 1
		c.A <<= 1
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return err
		}
		c.P.C = (x>>7)&1 == 1
		x <<= 1
		if err := c.write(operand, x); err != nil {
			return err
		}
		c.setN(x)
		c.setZ(x)
	}
	return nil
}

// BCC - Branch on Carry Clear.
func (c *CPU) bcc(mode addressingMode, operand uint16) error {
	if !c.P.C {
		c.PC = operand
	}
	return nil
}

// BCS - Branch on Carry Set.
func (c *CPU) bcs(mode addressingMode, operand uint16) error {
	if c.P.C {
		c.PC = operand
	}
	return nil
}

// BEQ - Branch on Equal.
func (c *CPU) beq(mode addressingMode, operand uint16) error {
	if c.P.Z {
		c.PC = operand
	}
	return nil
}

// BIT - test BITS.
func (c *CPU) bit(mode addressingMode, operand uint16) error {
	x, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.setN(x)
	c.setZ(c.A & x)
	c.P.V = (x>>6)&1 == 1
	return nil
}

// BMI - Branch on Minus.
func (c *CPU) bmi(mode addressingMode, operand uint16) error {
	if c.P.N {
		c.PC = operand
	}
	return nil
}

// BNE - Branch on Not Equal.
func (c *CPU) bne(mode addressingMode, operand uint16) error {
	if !c.P.Z {
		c.PC = operand
	}
	return nil
}

// BPL - Branch on Plus.
func (c *CPU) bpl(mode addressingMode, operand uint16) error {
	if !c.P.N {
		c.PC = operand
	}
	return nil
}

// BRK - Break Interrupt.
func (c *CPU) brk(mode addressingMode, operand uint16) error {
	if err := c.push(byte(c.PC>>8) & 0xFF); err != nil {
		return err
	}
	if err := c.push(byte(c.PC & 0xFF)); err != nil {
		return err
	}
	if err := c.push(c.P.encode()); err != nil {
		return err
	}
	c.P.I = true
	data, err := c.bus.read16(0xFFFE)
	if err != nil {
		return err
	}
	c.PC = data
	return nil
}

// BVC - Branch on Overflow Clear.
func (c *CPU) bvc(mode addressingMode, operand uint16) error {
	if !c.P.V {
		c.PC = operand
	}
	return nil
}

// BVS - Branch on Overflow Set.
func (c *CPU) bvs(mode addressingMode, operand uint16) error {
	if c.P.V {
		c.PC = operand
	}
	return nil
}

// CLC - Clear Carry.
func (c *CPU) clc(mode addressingMode, operand uint16) error {
	c.P.C = false
	return nil
}

// CLD - Clear Decimal.
func (c *CPU) cld(mode addressingMode, operand uint16) error {
	// Not implemented on NES
	return nil
}

// CLI - Clear Interrupt.
func (c *CPU) cli(mode addressingMode, operand uint16) error {
	c.P.I = false
	return nil
}

// CLV - Clear Overflow.
func (c *CPU) clv(mode addressingMode, operand uint16) error {
	c.P.V = false
	return nil
}

// CMP - Compare Accumulator.
func (c *CPU) cmp(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	x := c.A - data
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
	return nil
}

// CPX - Compare X register.
func (c *CPU) cpx(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	x := c.X - data
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
	return nil
}

// CPY - Compare Y register.
func (c *CPU) cpy(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	x := c.Y - data
	c.P.C = x >= 0
	c.setN(x)
	c.setZ(x)
	return nil
}

// DEC - Decrement Memory.
func (c *CPU) dec(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	x := data - 1 // this won't go negative.
	if err := c.write(operand, x); err != nil {
		return err
	}
	c.setN(x)
	c.setZ(x)
	return nil
}

// DEX - Decrement X Register.
func (c *CPU) dex(mode addressingMode, operand uint16) error {
	c.X--
	c.setN(c.X)
	c.setZ(c.X)
	return nil
}

// DEY - Decrement Y Register.
func (c *CPU) dey(mode addressingMode, operand uint16) error {
	c.Y--
	c.setN(c.Y)
	c.setZ(c.Y)
	return nil
}

// EOR - Bitwise Exclusive OR.
func (c *CPU) eor(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.A = c.A ^ data
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// INC - Increment Memory.
func (c *CPU) inc(mode addressingMode, operand uint16) error {
	x, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	x++
	if err := c.write(operand, x); err != nil {
		return err
	}
	c.setN(x)
	c.setZ(x)
	return nil
}

// INX - Increment X Register.
func (c *CPU) inx(mode addressingMode, operand uint16) error {
	c.X++
	c.setN(c.X)
	c.setZ(c.X)
	return nil
}

// INY - Increment Y Register.
func (c *CPU) iny(mode addressingMode, operand uint16) error {
	c.Y++
	c.setN(c.Y)
	c.setZ(c.Y)
	return nil
}

// JMP - Jump.
func (c *CPU) jmp(mode addressingMode, operand uint16) error {
	c.PC = operand
	return nil
}

// JSR - Jump to Subroutine.
func (c *CPU) jsr(mode addressingMode, operand uint16) error {
	x := c.PC - 1
	if err := c.push(byte(x>>8) & 0xFF); err != nil {
		return err
	}
	if err := c.push(byte(x & 0xFF)); err != nil {
		return err
	}
	c.PC = operand
	return nil
}

// LDA - Load Accumulator.
func (c *CPU) lda(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.A = data
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// LDX - Load X Register.
func (c *CPU) ldx(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.X = data
	c.setN(c.X)
	c.setZ(c.X)
	return nil
}

// LDY - Load Y Register.
func (c *CPU) ldy(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.Y = data
	c.setN(c.Y)
	c.setZ(c.Y)
	return nil
}

// LSR - Logical Shift Right.
func (c *CPU) lsr(mode addressingMode, operand uint16) error {
	if mode == accumulator {
		c.P.C = c.A&1 == 1
		c.A >>= 1
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return err
		}
		c.P.C = x&1 == 1
		x >>= 1
		if err := c.write(operand, x); err != nil {
			return err
		}
		c.setN(x)
		c.setZ(x)
	}
	return nil
}

// NOP - No Operation.
func (c *CPU) nop(mode addressingMode, operand uint16) error {
	// noop
	return nil
}

// ORA - Bitwise OR with Accumulator.
func (c *CPU) ora(mode addressingMode, operand uint16) error {
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	c.A = c.A | data
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// PHA - Push Accumulator.
func (c *CPU) pha(mode addressingMode, operand uint16) error {
	if err := c.push(c.A); err != nil {
		return err
	}
	return nil
}

// PHP - Push Processor Status.
func (c *CPU) php(mode addressingMode, operand uint16) error {
	if err := c.push(c.P.encode()); err != nil {
		return err
	}
	return nil
}

// PLA - Pull Accumulator.
func (c *CPU) pla(mode addressingMode, operand uint16) error {
	data, err := c.pop()
	if err != nil {
		return err
	}
	c.A = data
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// PLP - Pull Processor Status.
func (c *CPU) plp(mode addressingMode, operand uint16) error {
	data, err := c.pop()
	if err != nil {
		return err
	}
	c.P.decodeFrom(data)
	return nil
}

// ROL - Rotate Left.
func (c *CPU) rol(mode addressingMode, operand uint16) error {
	var carry byte = 0
	if c.P.C {
		carry = 1
	}
	if mode == accumulator {
		c.P.C = (c.A>>7)&1 == 1
		c.A = (c.A << 1) | carry
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return err
		}
		c.P.C = (x>>7)&1 == 1
		x = (x << 1) | carry
		if err := c.write(operand, x); err != nil {
			return err
		}
		c.setN(x)
		c.setZ(x)
	}
	return nil
}

// ROR - Rotate Right.
func (c *CPU) ror(mode addressingMode, operand uint16) error {
	var carry byte = 0
	if c.P.C {
		carry = 1
	}
	if mode == accumulator {
		c.P.C = c.A&1 == 1
		c.A = (c.A >> 1) | (carry << 7)
		c.setN(c.A)
		c.setZ(c.A)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return err
		}
		c.P.C = x&1 == 1
		x = (x >> 1) | (carry << 7)
		if err := c.write(operand, x); err != nil {
			return err
		}
		c.setN(x)
		c.setZ(x)
	}
	return nil
}

// RTS - Return from Subroutine.
func (c *CPU) rts(mode addressingMode, operand uint16) error {
	l, err := c.pop()
	if err != nil {
		return err
	}
	h, err := c.pop()
	if err != nil {
		return err
	}
	c.PC = (uint16(h)<<8 | uint16(l)) + 1
	return nil
}

// RTI - Return from Interrupt.
func (c *CPU) rti(mode addressingMode, operand uint16) error {
	p, err := c.pop()
	if err != nil {
		return err
	}
	c.P.decodeFrom(p)
	l, err := c.pop()
	if err != nil {
		return err
	}
	h, err := c.pop()
	if err != nil {
		return err
	}
	c.PC = uint16(h)<<8 | uint16(l)
	return nil
}

// SBC - Subtract with carry.
func (c *CPU) sbc(mode addressingMode, operand uint16) error {
	x := int16(c.A)
	data, err := c.bus.read(operand)
	if err != nil {
		return err
	}
	y := int16(data)
	var carry int16 = 0
	if c.P.C {
		carry = 1
	}
	res := x - y - (1 - carry)
	if 0 <= res {
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
	return nil
}

// SEC - Set Carry.
func (c *CPU) sec(mode addressingMode, operand uint16) error {
	c.P.C = true
	return nil
}

// SED - Set Carry.
func (c *CPU) sed(mode addressingMode, operand uint16) error {
	// Not implemented on NES.
	return nil
}

// SEI - Set Interrupt.
func (c *CPU) sei(mode addressingMode, operand uint16) error {
	c.P.I = true
	return nil
}

// STA - Store A Register.
func (c *CPU) sta(mode addressingMode, operand uint16) error {
	if err := c.write(operand, c.A); err != nil {
		return err
	}
	return nil
}

// STX - Store X Register.
func (c *CPU) stx(mode addressingMode, operand uint16) error {
	if err := c.write(operand, c.X); err != nil {
		return err
	}
	return nil
}

// STY - Store Y Register.
func (c *CPU) sty(mode addressingMode, operand uint16) error {
	if err := c.write(operand, c.Y); err != nil {
		return err
	}
	return nil
}

// TAX - Transfer A to X.
func (c *CPU) tax(mode addressingMode, operand uint16) error {
	c.X = c.A
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// TAY - Transfer A to Y.
func (c *CPU) tay(mode addressingMode, operand uint16) error {
	c.Y = c.A
	c.setN(c.Y)
	c.setZ(c.Y)
	return nil
}

// TSX - Transfer S to X.
func (c *CPU) tsx(mode addressingMode, operand uint16) error {
	c.X = c.S
	c.setN(c.S)
	c.setZ(c.S)
	return nil
}

// TXA - Transfer X to A.
func (c *CPU) txa(mode addressingMode, operand uint16) error {
	c.A = c.X
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// TXS - Transfer X to S.
func (c *CPU) txs(mode addressingMode, operand uint16) error {
	c.S = c.X
	return nil
}

// TYA - Transfer Y to A.
func (c *CPU) tya(mode addressingMode, operand uint16) error {
	c.A = c.Y
	c.setN(c.A)
	c.setZ(c.A)
	return nil
}

// NMI is non-maskable interrupt, this will be trigered by PPU.
func (c *CPU) nmi() error {
	if err := c.push(byte(c.PC>>8) & 0xFF); err != nil {
		return err
	}
	if err := c.push(byte(c.PC & 0xFF)); err != nil {
		return err
	}
	if err := c.push(c.P.encode()); err != nil {
		return err
	}
	data, err := c.bus.read16(0xFFFA)
	if err != nil {
		return err
	}
	c.PC = data
	c.P.I = true
	return nil
}

// Do performs the instruction cycle - fetch, decode, execute.
func (c *CPU) Do() (int, error) {
	// Running stall cycles.
	if 0 < c.stall {
		c.stall--
		c.lastExecution = fmt.Sprintf("CPU stall, PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x", c.PC, c.A, c.X, c.Y, c.S)
		return 1, nil
	}
	// Non-maskable interrupt.
	if c.nmiTriggered {
		c.nmi()
		c.nmiTriggered = false
		c.lastExecution = fmt.Sprintf("NMI, PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x", c.PC, c.A, c.X, c.Y, c.S)
		return 7, nil
	}
	opcode, err := c.bus.read(c.PC)
	if err != nil {
		return 0, err
	}
	instruction := c.instructions[opcode]
	var operand uint16 = 0
	switch instruction.mode {
	case implied:
		operand = 0
	case accumulator:
		operand = 0
	case immdiate:
		operand = c.PC + 1
	case zeropage:
		data, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		operand = uint16(data)
	case zeropageX:
		data, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(data+c.X) & 0xFF
	case zeropageY:
		data, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(data+c.Y) & 0xFF
	case relative:
		address, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		// Relative will look up a signed value
		// 2 is offset for operand
		if address < 0x80 {
			operand = c.PC + 2 + uint16(address)
		} else {
			operand = c.PC + 2 + uint16(address) - 0x100
		}
	case absolute:
		data, err := c.bus.read16(c.PC + 1)
		if err != nil {
			return 0, err
		}
		operand = data
	case absoluteX:
		data, err := c.bus.read16(c.PC + 1)
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.X)
	case absoluteY:
		data, err := c.bus.read16(c.PC + 1)
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.Y)
	case indirect:
		p, err := c.bus.read16(c.PC + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(p)
		if err != nil {
			return 0, err
		}
		operand = data
	case indirectX:
		p, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(uint16(p) + uint16(c.X))
		if err != nil {
			return 0, err
		}
		operand = data
	case indirectY:
		p, err := c.bus.read(c.PC + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(uint16(p))
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.Y)
	}
	c.PC += instruction.size
	// Save debug string.
	c.lastExecution = fmt.Sprintf("PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x, opcode=0x%02x, mnemonic=%s, operand: 0x%04x",
		c.PC, c.A, c.X, c.Y, c.S, opcode, instruction.mnemonic, operand)
	if err := instruction.execute(instruction.mode, operand); err != nil {
		return 0, err
	}
	return instruction.cycles, nil
}
