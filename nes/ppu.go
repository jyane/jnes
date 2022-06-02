package nes

import (
	"fmt"
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

// OAM data
type sprite struct {
	index int
	y     int

	// 76543210
	// ||||||||
	// |||||||+- Bank ($0000 or $1000) of tiles
	// +++++++-- Tile number of top of sprite (0 to 254; bottom half gets the next tile)
	tile byte

	// This attribute is separeted concept from "attribute" tables.
	// 76543210
	// ||||||||
	// ||||||++- Palette (4 to 7) of sprite
	// |||+++--- Unimplemented (read 0)
	// ||+------ Priority (0: in front of background; 1: behind background)
	// |+------- Flip sprite horizontally
	// +-------- Flip sprite vertically
	attribute byte
	x         int
}

func (s *sprite) bank() uint16 {
	if s.tile&1 == 0 {
		return 0x0000
	} else {
		return 0x1000
	}
}

func (s *sprite) tileByte() byte {
	return s.tile & 0xFE
}

func (s *sprite) priority() byte {
	return s.attribute >> 5 & 1
}

func (s *sprite) horizontalFlip() bool {
	return s.attribute>>6&1 == 1
}

func (s *sprite) verticalFlip() bool {
	return s.attribute>>7&1 == 1
}

// paletteAddress calculates its palette address from `value` which is from the tile.
func (s *sprite) paletteAddress(value byte) uint16 {
	return (0x3F00 | uint16((s.attribute&3)+4)*4) + uint16(value)
}

// PPU has an internal palette RAM
type paletteRAM struct {
	ram [32]byte
}

func (r *paletteRAM) read(address uint16) byte {
	// $3F20-$3FFF	  $00E0	  Mirrors of $3F00-$3F1F
	mirrored := (address-0x3F00)%0x20 + 0x3F00
	switch address {
	case 0x3F10, 0x3F14, 0x3F18, 0x3F1C:
		mirrored = address - 0x10
	case 0x3F04, 0x3F08, 0x3F0C:
		// These addresses are writable, but not readable.
		// failback to 0.
		mirrored = 0x3F00
	}
	mirrored -= 0x3F00
	return r.ram[mirrored]
}

func (r *paletteRAM) write(address uint16, data byte) {
	// $3F20-$3FFF	  $00E0	  Mirrors of $3F00-$3F1F
	mirrored := (address-0x3F00)%0x20 + 0x3F00
	switch address {
	case 0x3F10, 0x3F14, 0x3F18, 0x3F1C:
		mirrored = address - 0x10
	}
	mirrored -= 0x3F00
	r.ram[mirrored] = data
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

	picture *image.RGBA

	// Registers and temp data for PPU.
	// Reference:
	//   https://www.nesdev.org/wiki/PPU_registers
	//   https://www.nesdev.org/wiki/PPU_scrolling

	// oam
	oamAddress   byte
	primaryOAM   [256]byte // PPU has internal memory for Object Attribute Memory.
	secondaryOAM [8]sprite
	secondaryNum int // The number of sprites should be rendered on current line.

	// https://www.nesdev.org/wiki/PPU_sprite_evaluation
	spriteOverflow bool
	spriteZeroHit  bool

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
	grayScale          bool // unused.
	showLeftBackground bool
	showLeftSprite     bool
	showBackground     bool
	showSprite         bool
	emphasizeRed       bool // I have no idea about these, probably for PAL not NTSC.
	emphasizeGreen     bool // Same above.
	emphasizeBlue      bool // Same above.

	// $2002
	register byte

	// PPU has an internal RAM for palette data.
	paletteRAM paletteRAM

	// temp variables for rendering.
	nameTableByte      byte
	attributeTableByte byte
	lowTileByte        byte
	highTileByte       byte
	// PPU fetches data for rendering before 2 "fetch cycles".
	tileDataBuffer [6]byte

	// cycle, scanline indicates which pixel is processing.
	cycle    int
	scanline int
}

// NewPPU creates a PPU.
func NewPPU(bus *PPUBus) *PPU {
	p := &PPU{
		bus:     bus,
		picture: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
	return p
}

func (p *PPU) Reset() {
	// TODO(jyane): Configure correct state, I'm not sure where it starts, this may vary.
	// Here just starts from vblank.
	p.cycle = 0
	p.scanline = 240
}

func (p *PPU) Frame() (bool, *image.RGBA) {
	if p.cycle == 257 && p.scanline == 239 {
		return true, p.picture
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
	p.showLeftBackground = (data>>1)&1 == 1
	p.showLeftSprite = (data>>2)&1 == 1
	p.showBackground = (data>>3)&1 == 1
	p.showSprite = (data>>4)&1 == 1
	p.emphasizeRed = (data>>5)&1 == 1
	p.emphasizeGreen = (data>>6)&1 == 1
	p.emphasizeBlue = (data>>7)&1 == 1
}

// readPPUSTATUS reads PPUSTATUS ($2002).
func (p *PPU) readPPUSTATUS() byte {
	res := p.register & 0x1F
	if p.spriteOverflow {
		res |= 1 << 5
	}
	if p.spriteZeroHit {
		res |= 1 << 6
	}
	// Some implementations return current NMI, but as per nesdev:
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
	return p.primaryOAM[p.oamAddress]
}

// writeOAMDATA writes OAMDATA ($2004).
func (p *PPU) writeOAMDATA(data byte) {
	p.primaryOAM[p.oamAddress] = data
	p.oamAddress++
}

// writePPUSCROLL writes PPUSCROLL ($2005).
func (p *PPU) writePPUSCROLL(data byte) {
	if !p.w {
		// x-scroll
		// t: ....... ...ABCDE <- d: ABCDE...
		// x:              FGH <- d: .....FGH
		// w:                  <- 1
		p.t = (p.t & 0xFFE0) | (uint16(data) >> 3)
		p.x = data & 7
		p.w = true
	} else {
		// y-scroll
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
		p.paletteRAM.write(p.v, data)
	} else {
		if err := p.bus.write(p.v, data); err != nil {
			return fmt.Errorf("Failed to write PPUDATA: %w", err)
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
		return 0, fmt.Errorf("Failed to read PPUDATA: %w", err)
	}
	// Here buffers data if the address is not paletteRAM, because paletteRAM access is faster than bus access.
	if p.v < 0x3F00 {
		buffered := p.buffer
		p.buffer = data
		data = buffered
	} else {
		buf := p.paletteRAM.read(p.v)
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

func (p *PPU) color(value, attributeTableData byte) *color.RGBA {
	x := p.cycle - 1
	y := p.scanline
	num := byte(y&8)>>2 | byte(x&8)>>3
	palette := (attributeTableData >> (num << 1)) & 3
	paletteIndex := p.paletteRAM.read(0x3F00 | uint16((palette<<2)+value))
	return &colors[paletteIndex]
}

// incrementCoarseX increments X, calc from https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) incrementCoarseX() {
	if p.v&0x001F == 31 {
		p.v &= 0xFFE0
		p.v ^= 0x0400
	} else {
		p.v++
	}
}

// copyX copies X, calc from: https://www.nesdev.org/wiki/PPU_scrolling
func (p *PPU) copyX() {
	// v: .... .A.. ...B CDEF <- t: .... .A.. ...BCDEF
	p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
}

func (p *PPU) copyY() {
	// v: GHI A.BC DEF. .... <- t: GHIA.BC DEF.....
	p.v = (p.v & 0x841F) | (p.t & 0x7BE0)
}

// incrementY increments Y, calc from https://www.nesdev.org/wiki/PPU_scrolling#Wrapping_around
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

// evaluateSprite evalutes sprites.
// References:
//   https://www.nesdev.org/wiki/PPU_OAM
//   https://www.nesdev.org/wiki/PPU_sprite_evaluation
func (p *PPU) evaluateSprite() {
	// TODO(jyane): implement sprite size changing.
	spriteCount := 0
	for i := 0; i < 64; i++ {
		y := int(p.primaryOAM[i*4])
		tile := p.primaryOAM[i*4+1]
		attribute := p.primaryOAM[i*4+2]
		x := int(p.primaryOAM[i*4+3])
		if y <= p.scanline+1 && p.scanline+1 < y+8 {
			if spriteCount < 8 {
				p.secondaryOAM[spriteCount] = sprite{
					index:     i,
					y:         y,
					tile:      tile,
					attribute: attribute,
					x:         x,
				}
			}
			spriteCount++
		}
	}
	// NES allows only 8 sprites per line.
	if 8 < spriteCount {
		spriteCount = 8
		p.spriteOverflow = true // I'm not sure...
	}
	p.secondaryNum = spriteCount
}

// TODO(jyane): refactor? returning 3 results is odd.
func (p *PPU) renderSpritePixel() (int, byte, error) {
	if !p.showSprite {
		return 0, 0, nil
	}
	x := p.cycle - 1
	y := p.scanline
	// smaller index num should be prioritized.
	for i := 0; i < p.secondaryNum; i++ {
		sprite := p.secondaryOAM[i]
		// if this sprite should be rendered on current x.
		if sprite.x <= x && x < sprite.x+8 {
			h := y - sprite.y
			if sprite.verticalFlip() {
				h = 7 - h
			}
			address := 0x1000*uint16(p.spriteTableFlag) + uint16(sprite.tile)*16 + uint16(h)
			lowTileByte, err := p.bus.read(address)
			if err != nil {
				return 0, 0, err
			}
			highTileByte, err := p.bus.read(address + 8)
			if err != nil {
				return 0, 0, err
			}
			shift := 7 - (x - sprite.x)
			if sprite.horizontalFlip() {
				shift = x - sprite.x
			}
			lv := (lowTileByte >> shift) & 1
			hv := (highTileByte >> shift) & 1
			return i, lv + hv, nil
		}
	}
	return 0, 0, nil
}

func (p *PPU) renderBackgroundPixel() byte {
	if !p.showBackground {
		return 0
	}
	x := p.cycle - 1
	lowTileByte := p.tileDataBuffer[4]
	highTileByte := p.tileDataBuffer[5]
	lv := lowTileByte >> (7 - (x % 8)) & 1
	hv := highTileByte >> (7 - (x % 8)) & 1
	return lv + hv
}

func (p *PPU) renderPixel() error {
	x := p.cycle - 1 // cycle 0 won't be rendered
	y := p.scanline
	attributeTableByte := p.tileDataBuffer[3]
	bg := p.renderBackgroundPixel()
	i, sp, err := p.renderSpritePixel()
	if err != nil {
		return fmt.Errorf("Failed to render a sprite pixel: %w", err)
	}
	if x < 8 && !p.showLeftBackground {
		bg = 0
	}
	if x < 8 && !p.showLeftSprite {
		sp = 0
	}
	// BG pixel | Sprite pixel | Priority | Output
	// 0        | 0            | X        | BG($3F00)
	// 0        | 1-3          | X        | Sprite
	// 1-3      | 0            | X        | BG
	// 1-3      | 1-3          | 0        | Sprite
	// 1-3      | 1-3          | 1        | BG
	bgOpaque := bg != 0
	spOpaque := sp != 0
	sprite := p.secondaryOAM[i]
	color := &color.RGBA{}
	if !spOpaque && !bgOpaque {
		// both pixels are transparent, fallback to 0x3F00 color.
		color = &colors[p.paletteRAM.read(0x3F00)]
	} else if spOpaque && !bgOpaque {
		color = &colors[p.paletteRAM.read(sprite.paletteAddress(sp))]
	} else if !spOpaque && bgOpaque {
		color = p.color(bg, attributeTableByte)
	} else {
		// both pixles are opaque.
		// checking the priority.
		if sprite.priority() == 1 {
			// behind background.
			color = p.color(bg, attributeTableByte)
		} else {
			// in front of background.
			color = &colors[p.paletteRAM.read(sprite.paletteAddress(sp))]
		}
		// "when an opaque pixel of sprite 0 overlaps an opaque pixel of the background, this is a sprite zero hit"
		if sprite.index == 0 && x < 255 {
			p.spriteZeroHit = true
		}
	}
	p.picture.SetRGBA(x, y, *color)
	return nil
}

// Step emulates a cycle of PPU and each cycles renders a pixel for NTSC.
// Reference:
//   https://www.nesdev.org/wiki/PPU_rendering
//   https://www.nesdev.org/wiki/File:Ntsc_timing.png
func (p *PPU) Step() (bool, error) {
	// tick.
	p.cycle++
	if p.cycle == 341 {
		p.cycle = 0
		p.scanline++
		if p.scanline == 262 {
			p.scanline = 0
		}
	}
	// logic starts here.
	if p.showBackground {
		if 1 <= p.cycle && p.cycle <= 256 && p.scanline <= 239 {
			if err := p.renderPixel(); err != nil {
				return false, fmt.Errorf("Failed to render a pixel: %w", err)
			}
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
					p.tileDataBuffer[3] = p.tileDataBuffer[0] // attributeTableByte
					p.tileDataBuffer[4] = p.tileDataBuffer[1] // lowTileByte
					p.tileDataBuffer[5] = p.tileDataBuffer[2] // highTileByte
					p.tileDataBuffer[0] = p.attributeTableByte
					p.tileDataBuffer[1] = p.lowTileByte
					p.tileDataBuffer[2] = p.highTileByte
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
	// set vblank
	if p.scanline == 241 && p.cycle == 1 {
		p.updateNMI(true)
	}
	// clear vblank
	if p.scanline == 261 && p.cycle == 1 {
		p.spriteOverflow = false
		p.spriteZeroHit = false
		p.updateNMI(false)
	}
	// Actual sprite evaluation will happen on each cycles, here just computes all logic by 1.
	// Because sprite evaluation is independent logic.
	if p.cycle == 257 {
		if p.scanline < 240 {
			p.evaluateSprite()
		} else {
			p.secondaryNum = 0
		}
	}
	// Here makes sure that only 1 NMI happens per frame.
	if p.nmiOutput && p.nmiOccurred && p.scanline == 241 && p.cycle == 1 {
		return true, nil
	} else {
		return false, nil
	}
}
