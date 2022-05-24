package nes

import "github.com/golang/glog"

type PPUBus struct {
	vram      *RAM
	cartridge *Cartridge
}

// NewPPUBus creates a new Bus for PPU
func NewPPUBus(vram *RAM, cartridge *Cartridge) *PPUBus {
	return &PPUBus{vram, cartridge}
}

// horizontal, vertical
var offsets = []uint16{0x0400, 0x0800}

func (b *PPUBus) mirrorAddress(address uint16) uint16 {
	mode := b.cartridge.getTableMirrorMode()
	if 0x2000 <= address && address <= 0x23FF { // first screen
		return address - 0x2000
	} else {
		return address - 0x2000 - offsets[mode]
	}
}

// read reads data.
// Address        Size	  Description
// -------------------------------------
// $0000-$0FFF	  $1000	  Pattern table 0
// $1000-$1FFF	  $1000	  Pattern table 1
// $2000-$23FF	  $0400	  Nametable 0
// $2400-$27FF	  $0400	  Nametable 1
// $2800-$2BFF	  $0400	  Nametable 2
// $2C00-$2FFF	  $0400	  Nametable 3
// $3000-$3EFF	  $0F00	  Mirrors of $2000-$2EFF
// $3F00-$3F1F	  $0020	  Palette RAM indexes
// $3F20-$3FFF	  $00E0	  Mirrors of $3F00-$3F1F
// Reference: https://www.nesdev.org/wiki/PPU_memory_map
func (b *PPUBus) read(address uint16) byte {
	switch {
	case address < 0x2000:
		return b.cartridge.chrROM[address]
	case address < 0x3000:
		return b.vram.read(b.mirrorAddress(address) % 2048)
	case address < 0x3F00:
		// Mirror
		return b.vram.read((b.mirrorAddress(address) - 0x1000) % 2048)
	default:
		glog.Fatalf("Unknown PPU bus read: 0x%04x\n", address)
	}
	return 0
}

// write writes data.
// Reference: https://www.nesdev.org/wiki/PPU_memory_map
func (b *PPUBus) write(address uint16, data byte) {
	switch {
	case address < 0x2000:
		return
	case address < 0x3000:
		b.vram.write(b.mirrorAddress(address)%2048, data)
	case address < 0x3F00:
		// Mirror
		b.vram.write((b.mirrorAddress(address)-0x1000)%2048, data)
	default:
		glog.Fatalf("Unknown PPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	}
}
