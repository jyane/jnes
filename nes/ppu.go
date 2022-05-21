package nes

import (
	"image"
	"image/color"

	"github.com/golang/glog"
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
//
// This PPU implementation includes PPU regsters as well.
// References:
//   https://www.nesdev.org/wiki/PPU
//   https://pgate1.at-ninja.jp/NES_on_FPGA/nes_ppu.htm (In Japanese)
type PPU struct {
	bus *PPUBus

	background *image.RGBA

	// Registers for PPU.
	// Reference:
	//   https://www.nesdev.org/wiki/PPU_registers
	//   https://www.nesdev.org/wiki/PPU_scrolling

	oamAddress byte
	oamData    [256]byte // PPU has internal memory for Object Attribute Memory.

	// Current VRAM address (15bits), for PPUADDR $2006
	v uint16
	// Temporary VRAM address (15bits)
	t uint16
	// fine x scroll (3bits)
	x byte
	// w is a shared write toggle.
	w bool
	// buffer for PPUDATA $2007
	buffer byte

	// PPU has an internal RAM for palette data.
	paletteRAM [32]byte

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
	// TODO(jyane): Configure a correct state, I'm not sure where it starts, this may vary.
	// Here just starts from an invisible line.
	p.cycle = 0
	p.scanline = 241
}

// writePPUCTRL writes PPUCTRL ($2000).
func (p *PPU) writePPUCTRL(data byte) {
	// TODO(jyane): impelent
	// t: ...GH.. ........ <- d: ......GH
	p.t = (p.t & 0xF3FF) | ((uint16(data) & 0x03) << 10)
}

// writePPUMASK writes PPUMASK ($2001).
func (p *PPU) writePPUMASK(data byte) {
	// TODO(jyane): impelent
}

// readPPUSTATUS reads PPUSTATUS ($2002).
func (p *PPU) readPPUSTATUS() byte {
	// TODO(jyane): impelent
	p.w = false
	return 0
}

// writeOAMADDR writes OAMADDR ($2003).
func (p *PPU) writeOAMADDR(data byte) {
	p.oamAddress = data
}

// readOAMDATA reads OAMDATA ($2004).
func (p *PPU) readOAMDATA() byte {
	// TODO(jyane): impelent
	return 0
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
func (p *PPU) writePPUDATA(data byte) {
	// writing to paletteRAM
	if 0x3F00 <= p.v {
		p.paletteRAM[(p.v-0x3F00)%32] = data
	} else {
		p.bus.write(p.v, data)
	}
	p.v++
}

// readPPUDATA reads PPUDATA ($2007).
func (p *PPU) readPPUDATA() byte {
	data := p.bus.read(p.v)
	// Here buffers if the address is not paletteRAM.
	if p.v < 0x3F00 {
		buffered := p.buffer
		p.buffer = data
		data = buffered
	} else {
		p.buffer = p.bus.read(p.v)
	}
	p.v++
	return data
}

// writeOAMDMA writes OAMDMA ($4014).
func (p *PPU) writeOAMDMA(value byte) {
	glog.Infoln("writeOAMDMA called, not implemented.")
}

func (p *PPU) getColor(x, y int, v byte) *color.RGBA {
	attributeTileY := y / 16
	attributeTileX := x / 16
	attributeTableData := p.bus.read(0x23C0 + uint16(attributeTileY)*15 + uint16(attributeTileX))
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
	c := colors[paletteData]
	return &c
}

func (p *PPU) renderFrame() {
	glog.Infoln("rendering frame...")
	// Looking up NameTable
	for y := 0; y <= 240; y++ {
		for x := 0; x <= 256; x++ {
			tileY := y / 8
			tileX := x / 8
			nameTableAddress := 0x2000 + tileY*32 + tileX
			sprite := p.bus.read(uint16(nameTableAddress))
			lowTileAddress := uint16(sprite) * 16
			highTileAddress := uint16(sprite)*16 + 8
			var v byte
			for i := 0; i < 8; i++ {
				yy := y % 8
				lv := (p.bus.read(uint16(lowTileAddress + uint16(yy)))) >> (8 - (x % 8)) & 1
				hv := (p.bus.read(uint16(highTileAddress + uint16(yy)))) >> (8 - (x % 8)) & 1
				v = lv + hv
			}
			p.background.SetRGBA(x, y, *p.getColor(x, y, v))
		}
	}
	glog.Infoln("done")
}

// Do emulates a cycle of PPU and each cycles renders a pixel for NTSC,
// so PPU renders a pixel (right to left, top to bottom) respectively.
// PPU renders 256x240 pixels but it actually processes 341x261 area.
// Reference:
//   https://www.nesdev.org/wiki/PPU_rendering
//   https://www.nesdev.org/wiki/File:Ntsc_timing.png
func (p *PPU) Do() (bool, *image.RGBA) {
	// tick
	p.cycle++
	if p.cycle == 341 { // rendered a line
		p.cycle = 0
		p.scanline++
		if p.scanline == 261 { // rendered a frame
			p.scanline = 0
			p.renderFrame()
			return true, p.background
		}
	}
	return false, nil
}
