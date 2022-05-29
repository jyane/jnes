package nes

import (
	"fmt"

	"github.com/golang/glog"
)

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
	immediate
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
	c bool // carry
	z bool // zero
	i bool // IRQ
	d bool // decimal - unused on NES
	b bool // break
	r bool // reserved - unused
	v bool // overflow
	n bool // negative
}

// encode encodes the status to a byte.
func (s *status) encode() byte {
	var res byte
	if s.c {
		res |= (1 << 0)
	}
	if s.z {
		res |= (1 << 1)
	}
	if s.i {
		res |= (1 << 2)
	}
	if s.d {
		res |= (1 << 3)
	}
	if s.b {
		res |= (1 << 4)
	}
	if s.r {
		res |= (1 << 5)
	}
	if s.v {
		res |= (1 << 6)
	}
	if s.n {
		res |= (1 << 7)
	}
	return res
}

// decodeFrom decodes a byte to the status.
func (s *status) decodeFrom(data byte) {
	s.c = (data>>0)&1 == 1
	s.z = (data>>1)&1 == 1
	s.i = (data>>2)&1 == 1
	s.d = (data>>3)&1 == 1
	s.b = (data>>4)&1 == 1
	s.r = (data>>5)&1 == 1
	s.v = (data>>6)&1 == 1
	s.n = (data>>7)&1 == 1
}

type CPU struct {
	p             *status // Processor status flag bits
	a             byte    // Accumulator register
	x             byte    // Index register
	y             byte    // Index register
	pc            uint16  // Program counter
	s             byte    // Stack pointer
	lastExecution string  // For debug
	stall         uint64  // Stall cycles
	bus           *CPUBus
	// instructions needs references to CPU itself.
	instructions []instruction
	// interrupts
	nmiTriggered bool
}

// mnemonic will be empty if it still not implemented.
type instruction struct {
	mnemonic string
	mode     addressingMode
	// execute returns additional cycles if a page crossing happened on branch instructions.
	execute func(addressingMode, uint16) (int, error)
	size    uint16
	cycles  int
}

func (c *CPU) createInstructions() []instruction {
	return []instruction{
		{"BRK", implied, c.brk, 1, 7},     // 0x00
		{"ORA", indirectX, c.ora, 2, 6},   // 0x01
		{},                                // 0x02, STP
		{"SLO", indirectX, c.slo, 2, 8},   // 0x03
		{"NOP", zeropage, c.nop, 2, 3},    // 0x04
		{"ORA", zeropage, c.ora, 2, 3},    // 0x05
		{"ASL", zeropage, c.asl, 2, 5},    // 0x06
		{"SLO", zeropage, c.slo, 2, 5},    // 0x07
		{"PHP", implied, c.php, 1, 3},     // 0x08
		{"ORA", immediate, c.ora, 2, 2},   // 0x09
		{"ASL", accumulator, c.asl, 1, 2}, // 0x0A
		{},                                // 0x0B, ANC
		{"NOP", absolute, c.nop, 3, 4},    // 0x0C
		{"ORA", absolute, c.ora, 3, 4},    // 0x0D
		{"ASL", absolute, c.asl, 3, 6},    // 0x0E
		{"SLO", absolute, c.slo, 3, 6},    // 0x0F
		{"BPL", relative, c.bpl, 2, 2},    // 0x10
		{"ORA", indirectY, c.ora, 2, 5},   // 0x11
		{},                                // 0x12, STP
		{"SLO", indirectY, c.slo, 2, 7},   // 0x13
		{"NOP", zeropageX, c.nop, 2, 4},   // 0x14
		{"ORA", zeropageX, c.ora, 2, 4},   // 0x15
		{"ASL", zeropageX, c.asl, 2, 6},   // 0x16
		{"SLO", zeropageX, c.slo, 2, 6},   // 0x17
		{"CLC", implied, c.clc, 1, 2},     // 0x18
		{"ORA", absoluteY, c.ora, 3, 4},   // 0x19
		{"NOP", implied, c.nop, 1, 2},     // 0x1A
		{"SLO", absoluteY, c.slo, 3, 6},   // 0x1B
		{"NOP", absoluteX, c.nop, 3, 4},   // 0x1C
		{"ORA", absoluteX, c.ora, 3, 4},   // 0x1D
		{"ASL", absoluteX, c.asl, 3, 7},   // 0x1E
		{"SLO", absoluteX, c.slo, 3, 6},   // 0x1F
		{"JSR", absolute, c.jsr, 3, 6},    // 0x20
		{"AND", indirectX, c.and, 2, 6},   // 0x21
		{},                                // 0x22, STP
		{"RLA", indirectX, c.rla, 2, 8},   // 0x23
		{"BIT", zeropage, c.bit, 2, 3},    // 0x24
		{"AND", zeropage, c.and, 2, 3},    // 0x25
		{"ROL", zeropage, c.rol, 2, 5},    // 0x26
		{"RLA", zeropage, c.rla, 2, 5},    // 0x27
		{"PLP", implied, c.plp, 1, 4},     // 0x28
		{"AND", immediate, c.and, 2, 2},   // 0x29
		{"ROL", accumulator, c.rol, 1, 2}, // 0x2A
		{},                                // 0x2B, ANC
		{"BIT", absolute, c.bit, 3, 4},    // 0x2C
		{"AND", absolute, c.and, 3, 4},    // 0x2D
		{"ROL", absolute, c.rol, 3, 6},    // 0x2E
		{"RLA", absolute, c.rla, 3, 6},    // 0x2F
		{"BMI", relative, c.bmi, 2, 2},    // 0x30
		{"AND", indirectY, c.and, 2, 5},   // 0x31
		{},                                // 0x32, STP
		{"RLA", indirectY, c.rla, 2, 7},   // 0x33
		{"NOP", zeropage, c.nop, 2, 4},    // 0x34
		{"AND", zeropageX, c.and, 2, 4},   // 0x35
		{"ROL", zeropageX, c.rol, 2, 6},   // 0x36
		{"RLA", zeropageX, c.rla, 2, 6},   // 0x37
		{"SEC", implied, c.sec, 1, 2},     // 0x38
		{"AND", absoluteY, c.and, 3, 4},   // 0x39
		{"NOP", implied, c.nop, 1, 2},     // 0x3A
		{"RLA", absoluteY, c.rla, 3, 6},   // 0x3B
		{"NOP", absoluteX, c.nop, 3, 4},   // 0x3C
		{"AND", absoluteX, c.and, 3, 4},   // 0x3D
		{"ROL", absoluteX, c.rol, 3, 7},   // 0x3E
		{"RLA", absoluteX, c.rla, 3, 6},   // 0x3F
		{"RTI", implied, c.rti, 1, 6},     // 0x40
		{"EOR", indirectX, c.eor, 2, 6},   // 0x41
		{},                                // 0x42, STP
		{"SRE", indirectX, c.sre, 2, 8},   // 0x43
		{"NOP", zeropage, c.nop, 2, 3},    // 0x44
		{"EOR", zeropage, c.eor, 2, 3},    // 0x45
		{"LSR", zeropage, c.lsr, 2, 5},    // 0x46
		{"SRE", zeropage, c.sre, 2, 5},    // 0x47
		{"PHA", implied, c.pha, 1, 3},     // 0x48
		{"EOR", immediate, c.eor, 2, 2},   // 0x49
		{"LSR", accumulator, c.lsr, 1, 2}, // 0x4A
		{},                                // 0x4B, ALR
		{"JMP", absolute, c.jmp, 3, 3},    // 0x4C
		{"EOR", absolute, c.eor, 3, 4},    // 0x4D
		{"LSR", absolute, c.lsr, 3, 6},    // 0x4E
		{"SRE", absolute, c.sre, 3, 6},    // 0x4F
		{"BVC", relative, c.bvc, 2, 2},    // 0x50
		{"EOR", indirectY, c.eor, 2, 5},   // 0x51
		{},                                // 0x52, STP
		{"SRE", indirectY, c.sre, 2, 7},   // 0x53
		{"NOP", zeropage, c.nop, 2, 4},    // 0x54
		{"EOR", zeropageX, c.eor, 2, 4},   // 0x55
		{"LSR", zeropageX, c.lsr, 2, 6},   // 0x56
		{"SRE", zeropageX, c.sre, 2, 6},   // 0x57
		{"CLI", implied, c.cli, 1, 2},     // 0x58
		{"EOR", absoluteY, c.eor, 3, 4},   // 0x59
		{"NOP", implied, c.nop, 1, 2},     // 0x5A
		{"SRE", absoluteY, c.sre, 3, 6},   // 0x5B
		{"NOP", absoluteX, c.nop, 3, 4},   // 0x5C
		{"EOR", absoluteX, c.eor, 3, 4},   // 0x5D
		{"LSR", absoluteX, c.lsr, 3, 7},   // 0x5E
		{"SRE", absoluteX, c.sre, 3, 6},   // 0x5F
		{"RTS", implied, c.rts, 1, 6},     // 0x60
		{"ADC", indirectX, c.adc, 2, 6},   // 0x61
		{},                                // 0x62, STP
		{"RRA", indirectX, c.rra, 2, 8},   // 0x63
		{"NOP", zeropage, c.nop, 2, 3},    // 0x64
		{"ADC", zeropage, c.adc, 2, 3},    // 0x65
		{"ROR", zeropage, c.ror, 2, 5},    // 0x66
		{"RRA", zeropage, c.rra, 2, 5},    // 0x67
		{"PLA", implied, c.pla, 1, 4},     // 0x68
		{"ADC", immediate, c.adc, 2, 2},   // 0x69
		{"ROR", accumulator, c.ror, 1, 2}, // 0x6A
		{},                                // 0x6B, ARR
		{"JMP", indirect, c.jmp, 3, 5},    // 0x6C
		{"ADC", absolute, c.adc, 3, 4},    // 0x6D
		{"ROR", absolute, c.ror, 3, 6},    // 0x6E
		{"RRA", absolute, c.rra, 3, 6},    // 0x6F
		{"BVS", relative, c.bvs, 2, 2},    // 0x70
		{"ADC", indirectY, c.adc, 2, 5},   // 0x71
		{},                                // 0x72, STP
		{"RRA", indirectY, c.rra, 2, 7},   // 0x73
		{"NOP", zeropage, c.nop, 2, 4},    // 0x74
		{"ADC", zeropageX, c.adc, 2, 4},   // 0x75
		{"ROR", zeropageX, c.ror, 2, 6},   // 0x76
		{"RRA", zeropageX, c.rra, 2, 6},   // 0x77
		{"SEI", implied, c.sei, 1, 2},     // 0x78
		{"ADC", absoluteY, c.adc, 3, 4},   // 0x79
		{"NOP", implied, c.nop, 1, 2},     // 0x7A
		{"RRA", absoluteY, c.rra, 3, 6},   // 0x7B
		{"NOP", absoluteX, c.nop, 3, 4},   // 0x7C
		{"ADC", absoluteX, c.adc, 3, 4},   // 0x7D
		{"ROR", absoluteX, c.ror, 3, 7},   // 0x7E
		{"RRA", absoluteX, c.rra, 3, 6},   // 0x7F
		{"NOP", immediate, c.nop, 2, 2},   // 0x80
		{"STA", indirectX, c.sta, 2, 6},   // 0x81
		{"NOP", immediate, c.nop, 2, 2},   // 0x82
		{"SAX", indirectX, c.sax, 2, 6},   // 0x83
		{"STY", zeropage, c.sty, 2, 3},    // 0x84
		{"STA", zeropage, c.sta, 2, 3},    // 0x85
		{"STX", zeropage, c.stx, 2, 3},    // 0x86
		{"SAX", zeropage, c.sax, 2, 3},    // 0x87
		{"DEY", implied, c.dey, 1, 2},     // 0x88
		{"NOP", immediate, c.nop, 2, 2},   // 0x89
		{"TXA", implied, c.txa, 1, 2},     // 0x8A
		{},                                // 0x8B, XAA
		{"STY", absolute, c.sty, 3, 4},    // 0x8C
		{"STA", absolute, c.sta, 3, 4},    // 0x8D
		{"STX", absolute, c.stx, 3, 4},    // 0x8E
		{"SAX", absolute, c.sax, 3, 4},    // 0x8F
		{"BCC", relative, c.bcc, 2, 2},    // 0x90
		{"STA", indirectY, c.sta, 2, 6},   // 0x91
		{},                                // 0x92, STP
		{},                                // 0x93, AHX
		{"STY", zeropageX, c.sty, 2, 4},   // 0x94
		{"STA", zeropageX, c.sta, 2, 4},   // 0x95
		{"STX", zeropageY, c.stx, 2, 4},   // 0x96
		{"SAX", zeropageY, c.sax, 2, 4},   // 0x97
		{"TYA", implied, c.tya, 1, 2},     // 0x98
		{"STA", absoluteY, c.sta, 3, 5},   // 0x99
		{"TXS", implied, c.txs, 1, 2},     // 0x9A
		{},                                // 0x9B, TAS
		{},                                // 0x9C, SHY
		{"STA", absoluteX, c.sta, 3, 5},   // 0x9D
		{},                                // 0x9E, SHX
		{},                                // 0x9F, AHX
		{"LDY", immediate, c.ldy, 2, 2},   // 0xA0
		{"LDA", indirectX, c.lda, 2, 6},   // 0xA1
		{"LDX", immediate, c.ldx, 2, 2},   // 0xA2
		{"LAX", indirectX, c.lax, 2, 6},   // 0xA3
		{"LDY", zeropage, c.ldy, 2, 3},    // 0xA4
		{"LDA", zeropage, c.lda, 2, 3},    // 0xA5
		{"LDX", zeropage, c.ldx, 2, 3},    // 0xA6
		{"LAX", zeropage, c.lax, 2, 3},    // 0xA7
		{"TAY", implied, c.tay, 1, 2},     // 0xA8
		{"LDA", immediate, c.lda, 2, 2},   // 0xA9
		{"TAX", implied, c.tax, 1, 2},     // 0xAA
		{},                                // 0xAB, LAX
		{"LDY", absolute, c.ldy, 3, 4},    // 0xAC
		{"LDA", absolute, c.lda, 3, 4},    // 0xAD
		{"LDX", absolute, c.ldx, 3, 4},    // 0xAE
		{"LAX", absolute, c.lax, 3, 4},    // 0xAF
		{"BCS", relative, c.bcs, 2, 2},    // 0xB0
		{"LDA", indirectY, c.lda, 2, 5},   // 0xB1
		{},                                // 0xB2, STP
		{"LAX", indirectY, c.lax, 2, 5},   // 0xB3
		{"LDY", zeropageX, c.ldy, 2, 4},   // 0xB4
		{"LDA", zeropageX, c.lda, 2, 4},   // 0xB5
		{"LDX", zeropageY, c.ldx, 2, 4},   // 0xB6
		{"LAX", zeropageY, c.lax, 2, 4},   // 0xB7
		{"CLV", implied, c.clv, 1, 2},     // 0xB8
		{"LDA", absoluteY, c.lda, 3, 4},   // 0xB9
		{"TSX", implied, c.tsx, 1, 2},     // 0xBA
		{},                                // 0xBB, LAS
		{"LDY", absoluteX, c.ldy, 3, 4},   // 0xBC
		{"LDA", absoluteX, c.lda, 3, 4},   // 0xBD
		{"LDX", absoluteY, c.ldx, 3, 4},   // 0xBE
		{"LAX", absoluteY, c.lax, 3, 4},   // 0xBF
		{"CPY", immediate, c.cpy, 2, 2},   // 0xC0
		{"CMP", indirectX, c.cmp, 2, 6},   // 0xC1
		{"NOP", immediate, c.nop, 2, 2},   // 0xC2
		{"DCP", indirectX, c.dcp, 2, 8},   // 0xC3
		{"CPY", zeropage, c.cpy, 2, 3},    // 0xC4
		{"CMP", zeropage, c.cmp, 2, 3},    // 0xC5
		{"DEC", zeropage, c.dec, 2, 5},    // 0xC6
		{"DCP", zeropage, c.dcp, 2, 5},    // 0xC7
		{"INY", implied, c.iny, 1, 2},     // 0xC8
		{"CMP", immediate, c.cmp, 2, 2},   // 0xC9
		{"DEX", implied, c.dex, 1, 2},     // 0xCA
		{},                                // 0xCB, AXS
		{"CPY", absolute, c.cpy, 3, 4},    // 0xCC
		{"CMP", absolute, c.cmp, 3, 4},    // 0xCD
		{"DEC", absolute, c.dec, 3, 6},    // 0xCE
		{"DCP", absolute, c.dcp, 3, 6},    // 0xCF
		{"BNE", relative, c.bne, 2, 2},    // 0xD0
		{"CMP", indirectY, c.cmp, 2, 5},   // 0xD1
		{},                                // 0xD2, STP
		{"DCP", indirectY, c.dcp, 2, 7},   // 0xD3
		{"NOP", zeropage, c.nop, 2, 4},    // 0xD4
		{"CMP", zeropageX, c.cmp, 2, 4},   // 0xD5
		{"DEC", zeropageX, c.dec, 2, 6},   // 0xD6
		{"DCP", zeropageX, c.dcp, 2, 6},   // 0xD7
		{"CLD", implied, c.cld, 1, 2},     // 0xD8
		{"CMP", absoluteY, c.cmp, 3, 4},   // 0xD9
		{"NOP", implied, c.nop, 1, 2},     // 0xDA
		{"DCP", absoluteY, c.dcp, 3, 6},   // 0xDB
		{"NOP", absoluteX, c.nop, 3, 4},   // 0xDC
		{"CMP", absoluteX, c.cmp, 3, 4},   // 0xDD
		{"DEC", absoluteX, c.dec, 3, 7},   // 0xDE
		{"DCP", absoluteX, c.dcp, 3, 6},   // 0xDF
		{"CPX", immediate, c.cpx, 2, 2},   // 0xE0
		{"SBC", indirectX, c.sbc, 2, 6},   // 0xE1
		{"NOP", immediate, c.nop, 2, 2},   // 0xE2
		{"ISC", indirectX, c.isc, 2, 8},   // 0xE3
		{"CPX", zeropage, c.cpx, 2, 3},    // 0xE4
		{"SBC", zeropage, c.sbc, 2, 3},    // 0xE5
		{"INC", zeropage, c.inc, 2, 5},    // 0xE6
		{"ISC", zeropage, c.isc, 2, 5},    // 0xE7
		{"INX", implied, c.inx, 1, 2},     // 0xE8
		{"SBC", immediate, c.sbc, 2, 2},   // 0xE9
		{"NOP", implied, c.nop, 1, 2},     // 0xEA
		{"SBC", immediate, c.sbc, 2, 2},   // 0xEB
		{"CPX", absolute, c.cpx, 3, 4},    // 0xEC
		{"SBC", absolute, c.sbc, 3, 4},    // 0xED
		{"INC", absolute, c.inc, 3, 6},    // 0xEE
		{"ISC", absolute, c.isc, 3, 6},    // 0xEF
		{"BEQ", relative, c.beq, 2, 2},    // 0xF0
		{"SBC", indirectY, c.sbc, 2, 5},   // 0xF1
		{},                                // 0xF2, STP
		{"ISC", indirectY, c.isc, 2, 7},   // 0xF3
		{"NOP", zeropage, c.nop, 2, 4},    // 0xF4
		{"SBC", zeropageX, c.sbc, 2, 4},   // 0xF5
		{"INC", zeropageX, c.inc, 2, 6},   // 0xF6
		{"ISC", zeropageX, c.isc, 2, 6},   // 0xF7
		{"SED", implied, c.sed, 1, 2},     // 0xF8
		{"SBC", absoluteY, c.sbc, 3, 4},   // 0xF9
		{"NOP", implied, c.nop, 1, 2},     // 0xFA
		{"ISC", absoluteY, c.isc, 3, 6},   // 0xFB
		{"NOP", absoluteX, c.nop, 3, 4},   // 0xFC
		{"SBC", absoluteX, c.sbc, 3, 4},   // 0xFD
		{"INC", absoluteX, c.inc, 3, 7},   // 0xFE
		{"ISC", absoluteX, c.isc, 3, 6},   // 0xFF
	}
}

// NewCPU creates a new NES CPU.
func NewCPU(bus *CPUBus) *CPU {
	c := &CPU{
		p: &status{
			b: true,
			r: true,
		},
		bus: bus,
	}
	c.instructions = c.createInstructions()
	return c
}

// Reset does Reset.
func (c *CPU) Reset() error {
	data, err := c.bus.read16(0xFFFC)
	if err != nil {
		return fmt.Errorf("Failed to reset CPU: %w", err)
	}
	c.pc = data
	c.s = 0xFD
	c.p.decodeFrom(0x24)
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

// TODO(jyane): implement read to keep symmetry?

// setN sets whether the x is negative or positive.
func (c *CPU) setN(x byte) {
	c.p.n = x&0x80 != 0
}

// setZ sets whether the x is 0 or not.
func (c *CPU) setZ(x byte) {
	c.p.z = x == 0
}

// push pushes data to stack.
// "With the 6502, the stack is always on page one ($100-$1FF) and works top down."
func (c *CPU) push(x byte) error {
	if err := c.write((0x100 | (uint16(c.s) & 0xFF)), x); err != nil {
		return err
	}
	c.s--
	return nil
}

// pop pops data from stack.
// "With the 6502, the stack is always on page one ($100-$1FF) and works top down."
func (c *CPU) pop() (byte, error) {
	c.s++
	return c.bus.read((0x100 | (uint16(c.s) & 0xFF)))
}

// pageCrossed checks whether a and b is on the same page (0x??00 - 0x??FF).
func (c *CPU) pageCrossed(a, b uint16) bool {
	// That means if a and b don't have the same ?? - if the higher 8 bits are the same or not.
	return a&0xFF00 != b&0xFF00
}

// ADC - Add with Carry.
func (c *CPU) adc(mode addressingMode, operand uint16) (int, error) {
	x := uint16(c.a)
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	y := uint16(data)
	var carry uint16 = 0
	if c.p.c {
		carry = 1
	}
	res := x + y + carry
	if res > 0xFF {
		c.p.c = true
		c.a = byte(res & 0xFF)
	} else {
		c.p.c = false
		c.a = byte(res)
	}
	c.setN(c.a)
	c.setZ(c.a)
	// checks whether the value overflown by xor.
	if (x^y)&0x80 == 0 && (x^res)&0x80 != 0 {
		c.p.v = true
	} else {
		c.p.v = false
	}
	return 0, nil
}

// AND - And.
func (c *CPU) and(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.a = c.a & data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// ASL - Arithmetic Shift Left.
func (c *CPU) asl(mode addressingMode, operand uint16) (int, error) {
	if mode == accumulator {
		c.p.c = (c.a>>7)&1 == 1
		c.a <<= 1
		c.setN(c.a)
		c.setZ(c.a)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return 0, err
		}
		c.p.c = (x>>7)&1 == 1
		x <<= 1
		if err := c.write(operand, x); err != nil {
			return 0, err
		}
		c.setN(x)
		c.setZ(x)
	}
	return 0, nil
}

// BCC - Branch on Carry Clear.
func (c *CPU) bcc(mode addressingMode, operand uint16) (int, error) {
	if !c.p.c {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BCS - Branch on Carry Set.
func (c *CPU) bcs(mode addressingMode, operand uint16) (int, error) {
	if c.p.c {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BEQ - Branch on Equal.
func (c *CPU) beq(mode addressingMode, operand uint16) (int, error) {
	if c.p.z {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BIT - test BITS.
func (c *CPU) bit(mode addressingMode, operand uint16) (int, error) {
	x, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.setN(x)
	c.setZ(c.a & x)
	c.p.v = (x>>6)&1 == 1
	return 0, nil
}

// BMI - Branch on Minus.
func (c *CPU) bmi(mode addressingMode, operand uint16) (int, error) {
	if c.p.n {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BNE - Branch on Not Equal.
func (c *CPU) bne(mode addressingMode, operand uint16) (int, error) {
	if !c.p.z {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BPL - Branch on Plus.
func (c *CPU) bpl(mode addressingMode, operand uint16) (int, error) {
	if !c.p.n {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BRK - Break Interrupt.
func (c *CPU) brk(mode addressingMode, operand uint16) (int, error) {
	if err := c.push(byte(c.pc>>8) & 0xFF); err != nil {
		return 0, err
	}
	if err := c.push(byte(c.pc & 0xFF)); err != nil {
		return 0, err
	}
	if err := c.push(c.p.encode()); err != nil {
		return 0, err
	}
	c.p.i = true
	data, err := c.bus.read16(0xFFFE)
	if err != nil {
		return 0, err
	}
	c.pc = data
	return 0, nil
}

// BVC - Branch on Overflow Clear.
func (c *CPU) bvc(mode addressingMode, operand uint16) (int, error) {
	if !c.p.v {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// BVS - Branch on Overflow Set.
func (c *CPU) bvs(mode addressingMode, operand uint16) (int, error) {
	if c.p.v {
		cycles := 1
		c.pc = operand
		if c.pageCrossed(c.pc-1, operand) {
			cycles++
		}
		return cycles, nil
	}
	return 0, nil
}

// CLC - Clear Carry.
func (c *CPU) clc(mode addressingMode, operand uint16) (int, error) {
	c.p.c = false
	return 0, nil
}

// CLD - Clear Decimal.
func (c *CPU) cld(mode addressingMode, operand uint16) (int, error) {
	c.p.d = false
	return 0, nil
}

// CLI - Clear Interrupt.
func (c *CPU) cli(mode addressingMode, operand uint16) (int, error) {
	c.p.i = false
	return 0, nil
}

// CLV - Clear Overflow.
func (c *CPU) clv(mode addressingMode, operand uint16) (int, error) {
	c.p.v = false
	return 0, nil
}

// CMP - Compare Accumulator.
func (c *CPU) cmp(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	x := c.a - data
	c.p.c = c.a >= data
	c.setN(x)
	c.setZ(x)
	return 0, nil
}

// CPX - Compare X register.
func (c *CPU) cpx(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	x := c.x - data
	c.p.c = c.x >= data
	c.setN(x)
	c.setZ(x)
	return 0, nil
}

// CPY - Compare Y register.
func (c *CPU) cpy(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	x := c.y - data
	c.p.c = c.y >= data
	c.setN(x)
	c.setZ(x)
	return 0, nil
}

// DEC - Decrement Memory.
func (c *CPU) dec(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	x := data - 1 // this won't go negative.
	if err := c.write(operand, x); err != nil {
		return 0, err
	}
	c.setN(x)
	c.setZ(x)
	return 0, nil
}

// DEX - Decrement X Register.
func (c *CPU) dex(mode addressingMode, operand uint16) (int, error) {
	c.x--
	c.setN(c.x)
	c.setZ(c.x)
	return 0, nil
}

// DEY - Decrement Y Register.
func (c *CPU) dey(mode addressingMode, operand uint16) (int, error) {
	c.y--
	c.setN(c.y)
	c.setZ(c.y)
	return 0, nil
}

// EOR - Bitwise Exclusive OR.
func (c *CPU) eor(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.a = c.a ^ data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// INC - Increment Memory.
func (c *CPU) inc(mode addressingMode, operand uint16) (int, error) {
	x, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	x++
	if err := c.write(operand, x); err != nil {
		return 0, err
	}
	c.setN(x)
	c.setZ(x)
	return 0, nil
}

// INX - Increment X Register.
func (c *CPU) inx(mode addressingMode, operand uint16) (int, error) {
	c.x++
	c.setN(c.x)
	c.setZ(c.x)
	return 0, nil
}

// INY - Increment Y Register.
func (c *CPU) iny(mode addressingMode, operand uint16) (int, error) {
	c.y++
	c.setN(c.y)
	c.setZ(c.y)
	return 0, nil
}

// JMP - Jump.
func (c *CPU) jmp(mode addressingMode, operand uint16) (int, error) {
	c.pc = operand
	return 0, nil
}

// JSR - Jump to Subroutine.
func (c *CPU) jsr(mode addressingMode, operand uint16) (int, error) {
	x := c.pc - 1
	if err := c.push(byte(x>>8) & 0xFF); err != nil {
		return 0, err
	}
	if err := c.push(byte(x & 0xFF)); err != nil {
		return 0, err
	}
	c.pc = operand
	return 0, nil
}

// LDA - Load Accumulator.
func (c *CPU) lda(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.a = data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// LDX - Load X Register.
func (c *CPU) ldx(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.x = data
	c.setN(c.x)
	c.setZ(c.x)
	return 0, nil
}

// LDY - Load Y Register.
func (c *CPU) ldy(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.y = data
	c.setN(c.y)
	c.setZ(c.y)
	return 0, nil
}

// LSR - Logical Shift Right.
func (c *CPU) lsr(mode addressingMode, operand uint16) (int, error) {
	if mode == accumulator {
		c.p.c = c.a&1 == 1
		c.a >>= 1
		c.setN(c.a)
		c.setZ(c.a)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return 0, err
		}
		c.p.c = x&1 == 1
		x >>= 1
		if err := c.write(operand, x); err != nil {
			return 0, err
		}
		c.setN(x)
		c.setZ(x)
	}
	return 0, nil
}

// NOP - No Operation.
func (c *CPU) nop(mode addressingMode, operand uint16) (int, error) {
	if mode != implied {
		glog.Infof("Unofficial opcode execution: NOP(not $EA), operand: 0x%04x\n", operand)
	}
	// noop
	return 0, nil
}

// ORA - Bitwise OR with Accumulator.
func (c *CPU) ora(mode addressingMode, operand uint16) (int, error) {
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.a = c.a | data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// PHA - Push Accumulator.
func (c *CPU) pha(mode addressingMode, operand uint16) (int, error) {
	if err := c.push(c.a); err != nil {
		return 0, err
	}
	return 0, nil
}

// PHP - Push Processor Status.
func (c *CPU) php(mode addressingMode, operand uint16) (int, error) {
	if err := c.push(c.p.encode() | 0x10); err != nil {
		return 0, err
	}
	return 0, nil
}

// PLA - Pull Accumulator.
func (c *CPU) pla(mode addressingMode, operand uint16) (int, error) {
	data, err := c.pop()
	if err != nil {
		return 0, err
	}
	c.a = data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// PLP - Pull Processor Status.
func (c *CPU) plp(mode addressingMode, operand uint16) (int, error) {
	data, err := c.pop()
	if err != nil {
		return 0, err
	}
	c.p.decodeFrom(data&0xEF | 0x20)
	return 0, nil
}

// ROL - Rotate Left.
func (c *CPU) rol(mode addressingMode, operand uint16) (int, error) {
	var carry byte = 0
	if c.p.c {
		carry = 1
	}
	if mode == accumulator {
		c.p.c = (c.a>>7)&1 == 1
		c.a = (c.a << 1) | carry
		c.setN(c.a)
		c.setZ(c.a)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return 0, err
		}
		c.p.c = (x>>7)&1 == 1
		x = (x << 1) | carry
		if err := c.write(operand, x); err != nil {
			return 0, err
		}
		c.setN(x)
		c.setZ(x)
	}
	return 0, nil
}

// ROR - Rotate Right.
func (c *CPU) ror(mode addressingMode, operand uint16) (int, error) {
	var carry byte = 0
	if c.p.c {
		carry = 1
	}
	if mode == accumulator {
		c.p.c = c.a&1 == 1
		c.a = (c.a >> 1) | (carry << 7)
		c.setN(c.a)
		c.setZ(c.a)
	} else {
		x, err := c.bus.read(operand)
		if err != nil {
			return 0, err
		}
		c.p.c = x&1 == 1
		x = (x >> 1) | (carry << 7)
		if err := c.write(operand, x); err != nil {
			return 0, err
		}
		c.setN(x)
		c.setZ(x)
	}
	return 0, nil
}

// RTS - Return from Subroutine.
func (c *CPU) rts(mode addressingMode, operand uint16) (int, error) {
	l, err := c.pop()
	if err != nil {
		return 0, err
	}
	h, err := c.pop()
	if err != nil {
		return 0, err
	}
	c.pc = (uint16(h)<<8 | uint16(l)) + 1
	return 0, nil
}

// RTI - Return from Interrupt.
func (c *CPU) rti(mode addressingMode, operand uint16) (int, error) {
	p, err := c.pop()
	if err != nil {
		return 0, err
	}
	c.p.decodeFrom(p&0xEF | 0x20)
	l, err := c.pop()
	if err != nil {
		return 0, err
	}
	h, err := c.pop()
	if err != nil {
		return 0, err
	}
	c.pc = uint16(h)<<8 | uint16(l)
	return 0, nil
}

// SBC - Subtract with carry.
func (c *CPU) sbc(mode addressingMode, operand uint16) (int, error) {
	x := int16(c.a)
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	y := int16(data)
	carry := int16(0)
	if c.p.c {
		carry = 1
	}
	res := x - y - (1 - carry)
	if 0 <= res {
		c.p.c = true
	} else {
		c.p.c = false
	}
	c.a = byte(res)
	c.setN(c.a)
	c.setZ(c.a)
	// checks whether the value overflown by xor.
	if (x^y)&0x80 != 0 && (x^res)&0x80 != 0 {
		c.p.v = true
	} else {
		c.p.v = false
	}
	return 0, nil
}

// SEC - Set Carry.
func (c *CPU) sec(mode addressingMode, operand uint16) (int, error) {
	c.p.c = true
	return 0, nil
}

// SED - Set Carry.
func (c *CPU) sed(mode addressingMode, operand uint16) (int, error) {
	c.p.d = true
	return 0, nil
}

// SEI - Set Interrupt.
func (c *CPU) sei(mode addressingMode, operand uint16) (int, error) {
	c.p.i = true
	return 0, nil
}

// STA - Store A Register.
func (c *CPU) sta(mode addressingMode, operand uint16) (int, error) {
	if err := c.write(operand, c.a); err != nil {
		return 0, err
	}
	return 0, nil
}

// STX - Store X Register.
func (c *CPU) stx(mode addressingMode, operand uint16) (int, error) {
	if err := c.write(operand, c.x); err != nil {
		return 0, err
	}
	return 0, nil
}

// STY - Store Y Register.
func (c *CPU) sty(mode addressingMode, operand uint16) (int, error) {
	if err := c.write(operand, c.y); err != nil {
		return 0, err
	}
	return 0, nil
}

// TAX - Transfer A to X.
func (c *CPU) tax(mode addressingMode, operand uint16) (int, error) {
	c.x = c.a
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// TAY - Transfer A to Y.
func (c *CPU) tay(mode addressingMode, operand uint16) (int, error) {
	c.y = c.a
	c.setN(c.y)
	c.setZ(c.y)
	return 0, nil
}

// TSX - Transfer S to X.
func (c *CPU) tsx(mode addressingMode, operand uint16) (int, error) {
	c.x = c.s
	c.setN(c.s)
	c.setZ(c.s)
	return 0, nil
}

// TXA - Transfer X to A.
func (c *CPU) txa(mode addressingMode, operand uint16) (int, error) {
	c.a = c.x
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// TXS - Transfer X to S.
func (c *CPU) txs(mode addressingMode, operand uint16) (int, error) {
	c.s = c.x
	return 0, nil
}

// TYA - Transfer Y to A.
func (c *CPU) tya(mode addressingMode, operand uint16) (int, error) {
	c.a = c.y
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// NMI is non-maskable interrupt, this will be trigered by PPU.
func (c *CPU) nmi() error {
	if err := c.push(byte(c.pc>>8) & 0xFF); err != nil {
		return err
	}
	if err := c.push(byte(c.pc & 0xFF)); err != nil {
		return err
	}
	if err := c.push(c.p.encode()); err != nil {
		return err
	}
	data, err := c.bus.read16(0xFFFA)
	if err != nil {
		return err
	}
	c.pc = data
	c.p.i = true
	return nil
}

// Step performs the instruction cycle - fetch, decode, execute, and returns the number of consumed cycles.
func (c *CPU) Step() (int, error) {
	// Running stall cycles.
	if 0 < c.stall {
		c.stall--
		c.lastExecution = fmt.Sprintf("CPU stall, PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x", c.pc, c.a, c.x, c.y, c.s)
		// 514 (OAMDMA) is large, if this returns 514 cycles, may cause sync problems.
		// So here returns every single cycles to keep the sync with PPU.
		return 1, nil
	}
	// Non-maskable interrupt.
	didNMI := false
	if c.nmiTriggered {
		c.nmi()
		c.nmiTriggered = false
		didNMI = true
		c.lastExecution = fmt.Sprintf("NMI, PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x", c.pc, c.a, c.x, c.y, c.s)
	}
	opcode, err := c.bus.read(c.pc)
	if err != nil {
		return 0, fmt.Errorf("Failed to fetch opcode(0x%04x): %w", opcode, err)
	}
	instruction := c.instructions[opcode]
	operand := uint16(0)
	additionalCycle := false
	switch instruction.mode {
	case implied:
		operand = 0
	case accumulator:
		operand = 0
	case immediate:
		operand = c.pc + 1
	case zeropage:
		data, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		operand = uint16(data)
	case zeropageX:
		data, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(data+c.x) & 0xFF
	case zeropageY:
		data, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		// If the address exceeds 0xFF (page crossed), back to 0x00
		operand = uint16(data+c.y) & 0xFF
	case relative:
		address, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		// Relative will look up a signed value
		// 2 is offset for operand
		if address < 0x80 {
			operand = c.pc + 2 + uint16(address)
		} else {
			operand = c.pc + 2 + uint16(address) - 0x100
		}
	case absolute:
		data, err := c.bus.read16(c.pc + 1)
		if err != nil {
			return 0, err
		}
		operand = data
	case absoluteX:
		data, err := c.bus.read16(c.pc + 1)
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.x)
		additionalCycle = c.pageCrossed(operand-uint16(c.x), operand)
	case absoluteY:
		data, err := c.bus.read16(c.pc + 1)
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.y)
		additionalCycle = c.pageCrossed(operand-uint16(c.y), operand)
	case indirect:
		p, err := c.bus.read16(c.pc + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(p)
		if err != nil {
			return 0, err
		}
		operand = data
	case indirectX:
		p, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(uint16(p + c.x))
		if err != nil {
			return 0, err
		}
		operand = data
	case indirectY:
		p, err := c.bus.read(c.pc + 1)
		if err != nil {
			return 0, err
		}
		data, err := c.bus.read16Wrap(uint16(p))
		if err != nil {
			return 0, err
		}
		operand = data + uint16(c.y)
		additionalCycle = c.pageCrossed(operand-uint16(c.y), operand)
	}
	mnemonic := instruction.mnemonic
	if mnemonic == "" {
		return 0, fmt.Errorf("Tried to execute unimplemented instruction: opcode=0x%02x", opcode)
	}
	// Save debug string.
	c.lastExecution = fmt.Sprintf("PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x, P=0x%02x, opcode=0x%02x, mnemonic=%s, operand: 0x%04x",
		c.pc, c.a, c.x, c.y, c.s, c.p.encode(), opcode, mnemonic, operand)
	c.pc += instruction.size
	branchCycles, err := instruction.execute(instruction.mode, operand)
	if err != nil {
		return 0, fmt.Errorf("Failed to execute an instruction(%s): %w", c.lastExecution, err)
	}
	// Adding some cycles if needed.
	cycles := instruction.cycles
	cycles += branchCycles
	if didNMI {
		cycles += 7
	}
	// STA shouldn't be affected the page crossing.
	if additionalCycle && mnemonic != "STA" {
		cycles += 1
	}
	return cycles, nil
}

// Unofficial opcodes - only a few games depend these opcodes.
// Note: These implementations depend on existing opcode implementations.

// LAX - ?
func (c *CPU) lax(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: LAX, operand: 0x%04x\n", operand)
	data, err := c.bus.read(operand)
	if err != nil {
		return 0, err
	}
	c.a = data
	c.x = data
	c.setN(c.a)
	c.setZ(c.a)
	return 0, nil
}

// SAX - ?
func (c *CPU) sax(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: SAX, operand: 0x%04x\n", operand)
	x := c.a & c.x
	if err := c.write(operand, x); err != nil {
		return 0, err
	}
	return 0, nil
}

// DCP - ?
func (c *CPU) dcp(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: DCP, operand: 0x%04x\n", operand)
	c.dec(mode, operand)
	c.cmp(mode, operand)
	return 0, nil
}

// ISC - ?
func (c *CPU) isc(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: ISC, operand: 0x%04x\n", operand)
	c.inc(mode, operand)
	c.sbc(mode, operand)
	return 0, nil
}

// SLO - ?
func (c *CPU) slo(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: SLO, operand: 0x%04x\n", operand)
	c.asl(mode, operand)
	c.ora(mode, operand)
	return 0, nil
}

// RLA - ?
func (c *CPU) rla(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: RLA, operand: 0x%04x\n", operand)
	c.rol(mode, operand)
	c.and(mode, operand)
	return 0, nil
}

// SRE - ?
func (c *CPU) sre(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: SRE, operand: 0x%04x\n", operand)
	c.lsr(mode, operand)
	c.eor(mode, operand)
	return 0, nil
}

// RRA - ?
func (c *CPU) rra(mode addressingMode, operand uint16) (int, error) {
	glog.Infof("Unofficial opcode execution: SRE, operand: 0x%04x\n", operand)
	c.ror(mode, operand)
	c.adc(mode, operand)
	return 0, nil
}
