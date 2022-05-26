package nes

import (
	"image"
	"image/color"
)

// NES PPU generates 256x240 pixels.
const (
	width  = 256
	height = 240
)

// Palatte colors borrowed from "RGB".
// Reference: https://emulation.gametechwiki.com/index.php/Famicom_color_palette
var colors = [64]color.RGBA{
	{0x6D, 0x6D, 0x6D, 255}, {0x00, 0x24, 0x92, 255}, {0x00, 0x00, 0xDB, 255}, {0x6D, 0x49, 0xDB, 255},
	{0x92, 0x00, 0x6D, 255}, {0xB6, 0x00, 0x6D, 255}, {0xB6, 0x24, 0x00, 255}, {0x92, 0x49, 0x00, 255},
	{0x6D, 0x49, 0x00, 255}, {0x24, 0x49, 0x00, 255}, {0x00, 0x6D, 0x24, 255}, {0x00, 0x92, 0x00, 255},
	{0x00, 0x49, 0x49, 255}, {0x00, 0x00, 0x00, 255}, {0x00, 0x00, 0x00, 255}, {0x00, 0x00, 0x00, 255},
	{0xB6, 0xB6, 0xB6, 255}, {0x00, 0x6D, 0xDB, 255}, {0x00, 0x49, 0xFF, 255}, {0x92, 0x00, 0xFF, 255},
	{0xB6, 0x00, 0xFF, 255}, {0xFF, 0x00, 0x92, 255}, {0xFF, 0x00, 0x00, 255}, {0xDB, 0x6D, 0x00, 255},
	{0x92, 0x6D, 0x00, 255}, {0x24, 0x92, 0x00, 255}, {0x00, 0x92, 0x00, 255}, {0x00, 0xB6, 0x6D, 255},
	{0x00, 0x92, 0x92, 255}, {0x24, 0x24, 0x24, 255}, {0x00, 0x00, 0x00, 255}, {0x00, 0x00, 0x00, 255},
	{0xFF, 0xFF, 0xFF, 255}, {0x6D, 0xB6, 0xFF, 255}, {0x92, 0x92, 0xFF, 255}, {0xDB, 0x6D, 0xFF, 255},
	{0xFF, 0x00, 0xFF, 255}, {0xFF, 0x6D, 0xFF, 255}, {0xFF, 0x92, 0x00, 255}, {0xFF, 0xB6, 0x00, 255},
	{0xDB, 0xDB, 0x00, 255}, {0x6D, 0xDB, 0x00, 255}, {0x00, 0xFF, 0x00, 255}, {0x49, 0xFF, 0xDB, 255},
	{0x00, 0xFF, 0xFF, 255}, {0x49, 0x49, 0x49, 255}, {0x00, 0x00, 0x00, 255}, {0x00, 0x00, 0x00, 255},
	{0xFF, 0xFF, 0xFF, 255}, {0xB6, 0xDB, 0xFF, 255}, {0xDB, 0xB6, 0xFF, 255}, {0xFF, 0xB6, 0xFF, 255},
	{0xFF, 0x92, 0xFF, 255}, {0xFF, 0xB6, 0xB6, 255}, {0xFF, 0xDB, 0x92, 255}, {0xFF, 0xFF, 0x49, 255},
	{0xFF, 0xFF, 0x6D, 255}, {0xB6, 0xFF, 0x49, 255}, {0x92, 0xFF, 0x6D, 255}, {0x49, 0xFF, 0xDB, 255},
	{0x92, 0xDB, 0xFF, 255}, {0x92, 0x92, 0x92, 255}, {0x00, 0x00, 0x00, 255}, {0x00, 0x00, 0x00, 255},
}

// PPU stands for Picture Processing Unit, renders 256px x 240px image for a screen.
// PPU is 3x faster than CPU and rendering 1 frame requires 341x262=89342 cycles (Each cycles writes a dot).
// This implementation emulates NTSC not PAL or other ways.
//
// This PPU implementation includes PPU regsters as well.
// References:
//   https://www.nesdev.org/wiki/PPU
//   https://pgate1.at-ninja.jp/NES_on_FPGA/nes_ppu.htm (In Japanese)
type PPU struct {
	bus *PPUBus

	background *image.RGBA

	// Registers and temp data for PPU.
	// Reference:
	//   https://www.nesdev.org/wiki/PPU_registers
	//   https://www.nesdev.org/wiki/PPU_scrolling

	// oam
	oamAddress byte
	oamData    [256]byte // PPU has internal memory for Object Attribute Memory.

	// Current VRAM address (15bits), for PPUADDR $2006
	// yyy NN YYYYY XXXXX
	// ||| || ||||| +++++-- coarse X scroll
	// ||| || +++++-------- coarse Y scroll
	// ||| ++-------------- nametable select
	// +++----------------- fine Y scroll
	v uint16
	// Temporary VRAM address (15bits)
	t uint16
	// fine x scroll (3bits)
	x byte
	// w is a shared write toggle.
	w bool
	// buffer for PPUDATA $2007
	buffer byte

	// NMI https://www.nesdev.org/wiki/NMI
	nmiOccurred bool
	oldNMI      bool
	nmiOutput   bool

	// $2000
	nameTableFlag         byte // 0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00
	vramIncrementFlag     byte // 0: add 1, going across; 1: add 32, going down
	spriteTableFlag       byte // 0: $0000; 1: $1000; ignored in 8x16 mode
	backgroundTableFlag   byte // 0: $0000; 1: $1000
	spriteSizeFlag        byte // 0: 8x8 pixels; 1: 8x16 pixels
	masterSlaveSelectFlag byte // 0: read backdrop from EXT pins; 1: output color on EXT pins

	// $2001
	grayScale              bool // unused.
	showLeftBackgroundFlag bool
	showLeftSpriteFlag     bool
	showBackgroundFlag     bool
	showSpriteFlag         bool
	emphasizeRedFlag       bool // I have no idea about these, probably for PAL not NTSC.
	emphasizeGreenFlag     bool // Same above.
	emphasizeBlueFlag      bool // Same above.

	// https://www.nesdev.org/wiki/PPU_sprite_evaluation
	spriteOverflow byte
	spriteZeroHit  byte

	// $2002
	register byte

	// PPU has an internal RAM for palette data.
	paletteRAM [32]byte

	// temp variables for rendering.
	nameTableByte      byte
	attributeTableByte byte
	lowTileByte        byte
	highTileByte       byte
	// PPU fetches data for rendering before 2 "fetch cycles".
	tempTileData [6]byte

	// cycle, scanline indicates which pixel is processing.
	cycle    int
	scanline int
}

// NewPPU creates a PPU.
func NewPPU(bus *PPUBus) *PPU {
	p := &PPU{
		bus:        bus,
		background: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
	p.Reset()
	return p
}

func (p *PPU) Reset() {
	// TODO(jyane): Configure correct state, I'm not sure where it starts, this may vary.
	// Here just starts from vblank.
	p.cycle = 0
	p.scanline = 241
}

func (p *PPU) Frame() (bool, *image.RGBA) {
	if p.cycle == 257 && p.scanline == 239 {
		return true, p.background
	} else {
		return false, nil
	}
}

// writePPUCTRL writes PPUCTRL ($2000).
func (p *PPU) writePPUCTRL(data byte) {
	p.nameTableFlag = data & 3
	p.vramIncrementFlag = (data >> 2) & 1
	p.spriteTableFlag = (data >> 3) & 1
	p.backgroundTableFlag = (data >> 4) & 1
	p.spriteSizeFlag = (data >> 5) & 1
	p.masterSlaveSelectFlag = (data >> 6) & 1
	p.nmiOutput = (data>>7)&1 == 1
	// t: ...GH.. ........ <- d: ......GH
	p.t = (p.t & 0xF3FF) | ((uint16(data) & 0x03) << 10)
}

// writePPUMASK writes PPUMASK ($2001).
func (p *PPU) writePPUMASK(data byte) {
	p.grayScale = data&1 == 1
	p.showLeftBackgroundFlag = (data>>1)&1 == 1
	p.showLeftSpriteFlag = (data>>2)&1 == 1
	p.showBackgroundFlag = (data>>3)&1 == 1
	p.showSpriteFlag = (data>>4)&1 == 1
	p.emphasizeRedFlag = (data>>5)&1 == 1
	p.emphasizeGreenFlag = (data>>6)&1 == 1
	p.emphasizeBlueFlag = (data>>7)&1 == 1
}

// readPPUSTATUS reads PPUSTATUS ($2002).
func (p *PPU) readPPUSTATUS() byte {
	res := p.register & 0x1F
	res |= p.spriteOverflow << 5
	res |= p.spriteZeroHit << 6
	// Some implementations retrun current NMI, but as per nesdev:
	// "Return old status of NMI_occurred in bit 7, then set NMI_occurred to false."
	// https://www.nesdev.org/wiki/NMI
	if p.oldNMI {
		res |= 1 << 7
	}
	p.updateNMI(false)
	p.w = false
	return res
}

// writeOAMADDR writes OAMADDR ($2003).
func (p *PPU) writeOAMADDR(data byte) {
	p.oamAddress = data
}

// readOAMDATA reads OAMDATA ($2004).
func (p *PPU) readOAMDATA() byte {
	return p.oamData[p.oamAddress]
}

// writeOAMDATA writes OAMDATA ($2004).
func (p *PPU) writeOAMDATA(data byte) {
	p.oamData[p.oamAddress] = data
	p.oamAddress++
}

// writePPUSCROLL writes PPUSCROLL ($2005).
func (p *PPU) writePPUSCROLL(data byte) {
	if !p.w {
		// t: ....... ...ABCDE <- d: ABCDE...
		// x:              FGH <- d: .....FGH
		// w:                  <- 1
		p.t = (p.t & 0xFFE0) | (uint16(data) >> 3)
		p.x = data & 0b111
		p.w = true
	} else {
		// t: FGH..AB CDE..... <- d: ABCDEFGH
		// w:                  <- 0
		// ->
		// t: .FGH .... .... .... <- d: .... .FGH
		p.t = (p.t & 0x8FFF) | ((uint16(data) & 0x07) << 12)
		// t: .... ..AB CDE. .... <- d: ABCD E...
		p.t = (p.t & 0xFC1F) | ((uint16(data) & 0xF8) << 2)
		p.w = false
	}
}

// writePPUADDR writes PPUADDR ($2006).
func (p *PPU) writePPUADDR(data byte) {
	if !p.w {
		// t: ..CD EFGH .... .... <- d: ..CDEFGH
		//    <unused>     <- d: AB......
		// t: Z...... ........ <- 0 (bit Z is cleared)
		// w:                  <- 1
		p.t = (p.t & 0xC0FF) | (uint16(data) << 8)
		p.w = true
	} else {
		// t: ....... ABCDEFGH <- d: ABCDEFGH
		// v: <...all bits...> <- t: <...all bits...>
		// w:                  <- 0
		p.t = (p.t & 0xFF00) | uint16(data)
		p.v = p.t
		p.w = false
	}
}

// writePPUDATA writes PPUDATA ($2007).
func (p *PPU) writePPUDATA(data byte) error {
	// writing to paletteRAM
	if 0x3F00 <= p.v {
		p.paletteRAM[(p.v-0x3F00)%32] = data
	} else {
		if err := p.bus.write(p.v, data); err != nil {
			return err
		}
	}
	if p.vramIncrementFlag == 0 {
		p.v++
	} else {
		p.v += 32
	}
	return nil
}

// readPPUDATA reads PPUDATA ($2007).
func (p *PPU) readPPUDATA() (byte, error) {
	data, err := p.bus.read(p.v)
	if err != nil {
		return 0, err
	}
	// Here buffers data if the address is not paletteRAM, because paletteRAM access is faster than bus access.
	if p.v < 0x3F00 {
		buffered := p.buffer
		p.buffer = data
		data = buffered
	} else {
		buf, err := p.bus.read(p.v)
		if err != nil {
			return 0, err
		}
		p.buffer = buf
	}
	if p.vramIncrementFlag == 0 {
		p.v++
	} else {
		p.v += 32
	}
	return data, nil
}

func (p *PPU) updateNMI(flag bool) {
	p.nmiOccurred = flag
	p.oldNMI = p.nmiOccurred
}

// TODO(jyane): refactor this.
func (p *PPU) getColor(x, y int, v byte) *color.RGBA {
	attributeTableData := p.tempTileData[3]
	var num byte = 0
	if y%16 > 8 {
		num |= 0b10
	}
	if x%16 > 8 {
		num |= 0b01
	}
	var palette byte = 0 // 0, 1, 2 or 3
	palette |= (attributeTableData >> byte(2*num)) & 1
	palette |= 1 << ((attributeTableData >> byte(2*num+1)) & 1)
	paletteData := p.paletteRAM[uint16(palette*4-(4-v))]
	c := colors[paletteData%64]
	return &c
}

// Please see https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) incrementCoarseX() {
	if p.v&0x001F == 31 {
		p.v &= 0xFFE0
		p.v ^= 0x0400
	} else {
		p.v++
	}
}

// Please see https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) copyX() {
	// v: .... .A.. ...B CDEF <- t: .... .A.. ...BCDEF
	p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
}

func (p *PPU) copyY() {
	// v: GHI A.BC DEF. .... <- t: GHIA.BC DEF.....
	p.v = (p.v & 0x841F) | (p.t & 0x7BE0)
}

// Please see https://www.nesdev.org/wiki/PPU_scrolling#Wrapping_around
func (p *PPU) incrementY() {
	if (p.v & 0x7000) != 0x7000 {
		p.v += 0x1000
	} else {
		p.v &= 0x8FFF
		y := (p.v & 0x03E0) >> 5
		if y == 29 {
			y = 0
			p.v ^= 0x0800
		} else if y == 31 {
			y = 0
		} else {
			y++
		}
		p.v = (p.v & 0xFC1F) | (y << 5)
	}
}

func (p *PPU) fetchLowTileByte() error {
	fineY := (p.v >> 12) & 0b111
	address := 0x1000*uint16(p.backgroundTableFlag) + uint16(p.nameTableByte)*16 + fineY
	data, err := p.bus.read(address)
	if err != nil {
		return err
	}
	p.lowTileByte = data
	return nil
}

func (p *PPU) fetchHighTileByte() error {
	fineY := (p.v >> 12) & 0b111
	address := 0x1000*uint16(p.backgroundTableFlag) + uint16(p.nameTableByte)*16 + fineY + 8
	data, err := p.bus.read(address)
	if err != nil {
		return err
	}
	p.highTileByte = data
	return nil
}

// Address calc from https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) fetchAttributeTableByte() error {
	address := 0x23C0 | (p.v & 0x0C00) | ((p.v >> 4) & 0x38) | ((p.v >> 2) & 0x07)
	data, err := p.bus.read(address)
	if err != nil {
		return err
	}
	p.attributeTableByte = data
	return nil
}

// Address calc from https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) fetchNameTableByte() error {
	data, err := p.bus.read(0x2000 | (p.v & 0x0FFF))
	if err != nil {
		return err
	}
	p.nameTableByte = data
	return nil
}

func (p *PPU) renderPixel() {
	x := p.cycle - 1 // cycle 0 won't be rendered
	y := p.scanline
	lv := p.tempTileData[4] >> (8 - (x % 8)) & 1 // low tile
	hv := p.tempTileData[5] >> (8 - (x % 8)) & 1 // high tile
	p.background.SetRGBA(x, y, *p.getColor(x, y, lv+hv))
}

// Do emulates a cycle of PPU and each cycles renders a pixel for NTSC.
// Reference:
//   https://www.nesdev.org/wiki/PPU_rendering
//   https://www.nesdev.org/wiki/File:Ntsc_timing.png
func (p *PPU) Do() (bool, error) {
	if p.showBackgroundFlag {
		if 1 <= p.cycle && p.cycle <= 256 && p.scanline <= 239 {
			p.renderPixel()
		}
		if p.scanline == 261 && 280 <= p.cycle && p.cycle <= 304 {
			p.copyY()
		}
		if p.scanline < 240 || p.scanline == 261 {
			if 1 <= p.cycle && p.cycle <= 256 && p.cycle%8 == 0 {
				p.incrementCoarseX()
			}
			if p.cycle == 328 || p.cycle == 336 {
				p.incrementCoarseX()
			}
			if p.cycle == 256 {
				p.incrementY()
			}
			if p.cycle == 257 {
				p.copyX()
			}
			if (0 < p.cycle && p.cycle <= 257) || 320 < p.cycle {
				switch p.cycle % 8 {
				case 0:
					// PPU fetches tile data for current cycle "2 fetch cycles before".
					// Here stores current cycle data for after the next.
					p.tempTileData[3] = p.tempTileData[0] // attributeTableByte
					p.tempTileData[4] = p.tempTileData[1] // lowTileByte
					p.tempTileData[5] = p.tempTileData[2] // highTileByte
					p.tempTileData[0] = p.attributeTableByte
					p.tempTileData[1] = p.lowTileByte
					p.tempTileData[2] = p.highTileByte
				case 1:
					if err := p.fetchNameTableByte(); err != nil {
						return false, err
					}
				case 3:
					if err := p.fetchAttributeTableByte(); err != nil {
						return false, err
					}
				case 5:
					if err := p.fetchLowTileByte(); err != nil {
						return false, err
					}
				case 7:
					if err := p.fetchHighTileByte(); err != nil {
						return false, err
					}
				}
			}
		}
	}
	// TODO(jyane): NMI, I'm not sure whether this logic is correct or not.
	// set vblank
	if p.scanline == 241 && p.cycle == 1 {
		p.updateNMI(true)
	}
	// clear vblank
	if p.scanline == 261 && p.cycle == 1 {
		p.updateNMI(false)
	}
	// tick
	p.cycle++
	if p.cycle == 341 {
		p.cycle = 0
		p.scanline++
		if p.scanline == 262 {
			p.scanline = 0
		}
	}
	if p.nmiOutput && p.nmiOccurred {
		return true, nil
	} else {
		return false, nil
	}
}
