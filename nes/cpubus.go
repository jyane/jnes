package nes

import (
	"fmt"

	"github.com/golang/glog"
)

type CPUBus struct {
	wram       *RAM
	ppu        *PPU
	cartridge  *Cartridge
	controller *Controller
}

// NewCPUBus creates a new Bus for CPU.
// CPU memory map
// https://www.nesdev.org/wiki/CPU_memory_map
// Address range  Size   Device
// $0000-$07FF    $0800  2KB internal RAM
// $0800-$0FFF    $0800  Mirrors of $0000-$07FF
// $1000-$17FF    $0800
// $1800-$1FFF    $0800
// $2000-$2007    $0008  NES PPU registers
// $2008-$3FFF    $1FF8  Mirrors of $2000-2007 (repeats every 8 bytes)
// $4000-$4017    $0018  NES APU and I/O registers
// $4018-$401F    $0008  APU and I/O functionality that is normally disabled. See CPU Test Mode.
// $4020-$FFFF    $BFE0  Cartridge space: PRG ROM, PRG RAM, and mapper registers (See Note)

func NewCPUBus(wram *RAM, ppu *PPU, cartridge *Cartridge, controller *Controller) *CPUBus {
	return &CPUBus{wram, ppu, cartridge, controller}
}

// writeOAMDMA writes OAMDATA to PPU, this will be called by CPU.
func (b *CPUBus) writeOAMDMA(data [256]byte) {
	b.ppu.primaryOAM = data
}

func (b *CPUBus) readPPURegister(address uint16) (byte, error) {
	addr := 0x2000 | address%8
	switch addr {
	case 0x2002:
		return b.ppu.readPPUSTATUS(), nil
	case 0x2004:
		return b.ppu.readOAMDATA(), nil
	case 0x2007:
		return b.ppu.readPPUDATA()
	default:
		return 0, fmt.Errorf("PPU register $%04x (0x%04x) is not readable.", address, addr)
	}
}

// read reads a byte.
func (b *CPUBus) read(address uint16) (byte, error) {
	switch {
	case address < 0x2000:
		return b.wram.read(address % 0x0800), nil
	case address < 0x4000:
		data, err := b.readPPURegister(address)
		if err != nil {
			return 0, err
		}
		return data, nil
	case address == 0x4016: // 1P
		return b.controller.read(), nil
	case address < 0x4018:
		glog.Infof("Unimplemented CPU bus read: address=0x%04x\n", address)
		return 0, nil
	case address < 0x4020:
		return 0, fmt.Errorf("Reading unused bus address: 0x%04x\n", address)
	case 0x4020 <= address:
		return b.cartridge.readFromCPU(address)
	default:
		return 0, fmt.Errorf("Unknown CPU bus read: 0x%04x", address)
	}
}

//  read16Wrap returns 16 bytes with a known CPU bug.
func (b *CPUBus) read16Wrap(address uint16) (uint16, error) {
	a1 := address
	a2 := (address & 0xFF00) | ((address + 1) & 0xFF)
	l, err := b.read(a1)
	if err != nil {
		return 0, err
	}
	h, err := b.read(a2)
	if err != nil {
		return 0, err
	}
	return uint16(h)<<8 | uint16(l), nil
}

// read16 reads 2 bytes.
func (b *CPUBus) read16(address uint16) (uint16, error) {
	l, err := b.read(address)
	if err != nil {
		return 0, err
	}
	h, err := b.read(address + 1)
	if err != nil {
		return 0, err
	}
	return uint16(h)<<8 | uint16(l), nil
}

// writeToPPURegisters writes data to PPU registers.
func (b *CPUBus) writeToPPURegisters(address uint16, data byte) error {
	addr := 0x2000 | address%8
	switch addr {
	case 0x2000:
		b.ppu.writePPUCTRL(data)
	case 0x2001:
		b.ppu.writePPUMASK(data)
	case 0x2003:
		b.ppu.writePPUADDR(data)
	case 0x2004:
		b.ppu.writeOAMDATA(data)
	case 0x2005:
		b.ppu.writePPUSCROLL(data)
	case 0x2006:
		b.ppu.writePPUADDR(data)
	case 0x2007:
		return b.ppu.writePPUDATA(data)
	default:
		return fmt.Errorf("PPU register $%04x (0x%04x) is not writable.", address, addr)
	}
	return nil
}

// write writes a byte.
// This is supposed to be called from CPU write. Direct calling this function is not allowed,
// because writing data to oamdma is not implemented here (implemented on CPU-side).
func (b *CPUBus) write(address uint16, data byte) error {
	switch {
	case address < 0x2000:
		b.wram.write(address%0x0800, data)
	case address < 0x4000:
		return b.writeToPPURegisters(address, data)
	case address == 0x4014:
		// Implemented on CPU
		return fmt.Errorf("CPU bus write was probably illegally called. (OAMDMA $4014)")
	case address == 0x4016: // 1P
		b.controller.write(data)
	case address < 0x4018:
		glog.Infof("Unimplemented CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	case address < 0x4020:
		return fmt.Errorf("Writing data to unused bus address: 0x%04x\n", address)
	case 0x4020 <= address:
		return b.cartridge.writeFromCPU(address, data)
	default:
		return fmt.Errorf("Unknown CPU bus write: address=0x%04x, data=0x%02x", address, data)
	}
	return nil
}
