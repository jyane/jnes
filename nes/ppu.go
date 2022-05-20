package nes

import (
	"image"
	"image/color"
	"math/rand"

	"github.com/golang/glog"
)

// NES PPU generates 256x240 pixels.
const (
	width  = 256
	height = 240
)

// Palatte colors borrowed from "RGB".
// Reference: https://emulation.gametechwiki.com/index.php/Famicom_color_palette
var palettes = [64]color.RGBA{
	{0x6D, 0x6D, 0x6D, 1}, {0x00, 0x24, 0x92, 1}, {0x00, 0x00, 0xDB, 1}, {0x6D, 0x49, 0xDB, 1},
	{0x92, 0x00, 0x6D, 1}, {0xB6, 0x00, 0x6D, 1}, {0xB6, 0x24, 0x00, 1}, {0x92, 0x49, 0x00, 1},
	{0x6D, 0x49, 0x00, 1}, {0x24, 0x49, 0x00, 1}, {0x00, 0x6D, 0x24, 1}, {0x00, 0x92, 0x00, 1},
	{0x00, 0x49, 0x49, 1}, {0x00, 0x00, 0x00, 1}, {0x00, 0x00, 0x00, 1}, {0x00, 0x00, 0x00, 1},
	{0xB6, 0xB6, 0xB6, 1}, {0x00, 0x6D, 0xDB, 1}, {0x00, 0x49, 0xFF, 1}, {0x92, 0x00, 0xFF, 1},
	{0xB6, 0x00, 0xFF, 1}, {0xFF, 0x00, 0x92, 1}, {0xFF, 0x00, 0x00, 1}, {0xDB, 0x6D, 0x00, 1},
	{0x92, 0x6D, 0x00, 1}, {0x24, 0x92, 0x00, 1}, {0x00, 0x92, 0x00, 1}, {0x00, 0xB6, 0x6D, 1},
	{0x00, 0x92, 0x92, 1}, {0x24, 0x24, 0x24, 1}, {0x00, 0x00, 0x00, 1}, {0x00, 0x00, 0x00, 1},
	{0xFF, 0xFF, 0xFF, 1}, {0x6D, 0xB6, 0xFF, 1}, {0x92, 0x92, 0xFF, 1}, {0xDB, 0x6D, 0xFF, 1},
	{0xFF, 0x00, 0xFF, 1}, {0xFF, 0x6D, 0xFF, 1}, {0xFF, 0x92, 0x00, 1}, {0xFF, 0xB6, 0x00, 1},
	{0xDB, 0xDB, 0x00, 1}, {0x6D, 0xDB, 0x00, 1}, {0x00, 0xFF, 0x00, 1}, {0x49, 0xFF, 0xDB, 1},
	{0x00, 0xFF, 0xFF, 1}, {0x49, 0x49, 0x49, 1}, {0x00, 0x00, 0x00, 1}, {0x00, 0x00, 0x00, 1},
	{0xFF, 0xFF, 0xFF, 1}, {0xB6, 0xDB, 0xFF, 1}, {0xDB, 0xB6, 0xFF, 1}, {0xFF, 0xB6, 0xFF, 1},
	{0xFF, 0x92, 0xFF, 1}, {0xFF, 0xB6, 0xB6, 1}, {0xFF, 0xDB, 0x92, 1}, {0xFF, 0xFF, 0x49, 1},
	{0xFF, 0xFF, 0x6D, 1}, {0xB6, 0xFF, 0x49, 1}, {0x92, 0xFF, 0x6D, 1}, {0x49, 0xFF, 0xDB, 1},
	{0x92, 0xDB, 0xFF, 1}, {0x92, 0x92, 0x92, 1}, {0x00, 0x00, 0x00, 1}, {0x00, 0x00, 0x00, 1},
}

// Registers for PPU.
// Reference:
//   https://www.nesdev.org/wiki/PPU_registers
//   https://www.nesdev.org/wiki/PPU_scrolling
type PPURegisters struct {
	// Current VRAM address (15bit), for PPUADDR $2006
	address uint16
	// writeFlag indicates whether the current access is for high or low, for PPUADDR $2006
	writeFlag bool
	// buffer for PPUDATA
	buffer byte
}

// PPU stands for Picture Processing Unit, renders 256px x 240px image for a screen.
// PPU is 3x faster than CPU and rendering 1 frame requires 341x262=89342 cycles (Each cycles writes a dot).
//
// This PPU implementation includes PPU regsters as well.
// References:
//   https://www.nesdev.org/wiki/PPU
//   https://pgate1.at-ninja.jp/NES_on_FPGA/nes_ppu.htm (In Japanese)
type PPU struct {
	bus       *PPUBus
	registers *PPURegisters

	background *image.RGBA

	// PPU has an internal RAM for palette data.
	paletteRAM [32]byte

	// cycle, scanline indicates which pixel is processing.
	cycle    int
	scanline int

	// temp data: https://www.nesdev.org/wiki/File:Ntsc_timing.png
	nameTableData      byte
	attributeTableData byte
	lowTileData        byte
	highTileData       byte
}

// NewPPU creates a PPU.
func NewPPU(bus *PPUBus) *PPU {
	p := &PPU{
		bus:        bus,
		registers:  &PPURegisters{},
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

// read reads data from bus or internal palette RAM.
// All reads in PPU should call this method.
// Reference: https://www.nesdev.org/wiki/PPU_memory_map
func (p *PPU) read(address uint16) byte {
	switch {
	case address < 0x3F00:
		return p.bus.read(address)
	case address < 0x4000:
		return p.paletteRAM[(address-0x4000)%32]
	default:
		glog.Infof("Unknown PPU bus reference: 0x%04x\n", address)
	}
	return 0
}

// write writes data to bus or internal palette RAM.
// All writes in PPU should call this method.
// Reference: https://www.nesdev.org/wiki/PPU_memory_map
func (p *PPU) write(address uint16, x byte) {
	switch {
	case address < 0x3F00:
		p.bus.write(address, x)
	case address < 0x4000:
		p.paletteRAM[(address-0x4000)%32] = x
	default:
		glog.Infof("Unknown PPU bus reference: 0x%04x\n", address)
	}
}

// writePPUADDR writes PPUADDR ($2007).
func (p *PPU) writePPUADDR(data byte) {
	if p.registers.writeFlag { // low
		p.registers.writeFlag = false
		p.registers.address += uint16(data)
	} else { // high
		p.registers.address = uint16(data) << 8
		p.registers.writeFlag = true
	}
}

// readPPUDATA reads PPUDATA ($2006).
func (p *PPU) readPPUDATA() byte {
	data := p.read(p.registers.address)
	// buffer if the address is not paletteRAM.
	if p.registers.address < 0x3F00 {
		buffered := p.registers.buffer
		p.registers.buffer = data
		data = buffered
	} else {
		p.registers.buffer = p.read(p.registers.address)
	}
	p.registers.address++
	return data
}

// writePPUDATA writes PPUDATA ($2006).
func (p *PPU) writePPUDATA(data byte) {
	p.write(p.registers.address, data)
	p.registers.address++
}

func (p *PPU) fetchHighTileData() {
	address := 0x2000 + uint16(p.nameTableData)
	p.highTileData = p.read(address + 8)
}

func (p *PPU) fetchLowTileData() {
	address := 0x2000 + uint16(p.nameTableData)
	p.highTileData = p.read(address)
}

// https://www.nesdev.org/wiki/PPU_scrolling#Tile_and_attribute_fetching
func (p *PPU) fetchAttributeData() {
	v := p.registers.address
	address := 0x23C0 | (v & 0xC000) | ((v >> 4) & 0x38) | ((v >> 2) & 0x07)
	p.attributeTableData = p.read(address)
}

// https://www.nesdev.org/wiki/PPU_scrolling#Tile_and_attribute_fetching
func (p *PPU) fetchNameTableData() {
	address := 0x2000 | (p.registers.address & 0xFFF)
	glog.Infof("address=0x%04x, p.registers.address=0x%04x\n", address, p.registers.address)
	p.nameTableData = p.read(address)
	glog.Infof("p.nameTableData=0x%04x\n", p.nameTableData)
}

func (p *PPU) renderPixel(x, y int) {
	p.background.Set(x, y, palettes[rand.Intn(64)])
}

// Do emulates a cycle of PPU and each cycles renders a pixel for Analog TV screen,
// so PPU renders a pixel (right to left, top to bottom) respectively.
// PPU renders 256x240 pixels but it actually processes 341x261.
// Reference:
//   https://www.nesdev.org/wiki/PPU_rendering
//   https://www.nesdev.org/wiki/File:Ntsc_timing.png
func (p *PPU) Do() (bool, *image.RGBA) {
	// Please see the timing.png.
	if (1 <= p.cycle && p.cycle <= 256 || 321 <= p.cycle && p.cycle <= 340) && (p.scanline <= 240 || p.scanline == 261) {
		switch p.cycle % 8 {
		case 1:
			p.fetchNameTableData()
		case 3:
			p.fetchAttributeData()
		case 5:
			p.fetchLowTileData()
		case 7:
			p.fetchHighTileData()
		}
	}
	// PPU is processing visible area.
	if 1 <= p.cycle && p.cycle <= 256 && p.scanline <= 240 {
		p.renderPixel(p.cycle-1, p.scanline)
	}
	// tick
	p.cycle++
	if p.cycle == 341 { // rendered a line
		p.cycle = 0
		p.scanline++
		if p.scanline == 261 { // rendered a frame
			p.scanline = 0
			return true, p.background
		}
	}
	return false, nil
}
