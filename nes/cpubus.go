package nes

import "github.com/golang/glog"

type CPUBus struct {
	wram       *RAM
	ppu        *PPU
	cartridge  *Cartridge
	controller *Controller
}

// NewCPUBus creates a new Bus for CPU.
// CPU memory map
// 0x0000 - 0x07FF	WRAM
// 0x0800 - 0x1FFF	WRAM Mirror
// 0x2000 - 0x2007	PPU Registers
// 0x2008 - 0x3FFF	PPU Registers Mirror
// 0x4000 - 0x401F	I/O Port
// 0x4020 - 0x5FFF	Extended RAM
// 0x6000 - 0x7FFF	Battery Backup RAM
// 0x8000 - 0xBFFF	ProgramROM Low
// 0xC000 - 0xFFFF	ProgramROM High
func NewCPUBus(wram *RAM, ppu *PPU, cartridge *Cartridge, controller *Controller) *CPUBus {
	return &CPUBus{wram, ppu, cartridge, controller}
}

func (b *CPUBus) readPPURegister(address uint16) byte {
	switch address {
	case 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005:
		glog.Infof("Unimplemented CPU bus read: 0x%04x\n", address)
	case 0x2007:
		return b.ppu.readPPUDATA()
	default:
		glog.Fatalf("Unknown CPU bus read: 0x%04x\n", address)
	}
	return 0
}

// read reads a byte.
func (b *CPUBus) read(address uint16) byte {
	switch {
	case address < 0x0800:
		return b.wram.read(address)
	case address < 0x2000:
		return b.wram.read(address - 0x0800)
	case address < 0x2008:
		return b.readPPURegister(address)
	case address == 0x4016: // 1P
		return b.controller.read()
	case 0x8000 <= address:
		return b.cartridge.prgROM[address-0x8000]
	default:
		glog.Fatalf("Unknown CPU bus read: 0x%04x\n", address)
	}
	return 0
}

// read16 reads 2 bytes.
func (b *CPUBus) read16(address uint16) uint16 {
	l := uint16(b.read(address))
	h := uint16(b.read(address+1)) << 8
	return h | l
}

// writeToPPURegisters writes data to PPU registers.
func (b *CPUBus) writeToPPURegisters(address uint16, data byte) {
	switch address {
	case 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005:
		glog.Infof("Unimplemented CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	case 0x2006:
		b.ppu.writePPUADDR(data)
	case 0x2007:
		b.ppu.writePPUDATA(data)
	default:
		glog.Fatalf("Unkonwn CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	}
}

// write writes a byte.
func (b *CPUBus) write(address uint16, data byte) {
	switch {
	case address < 0x0800:
		b.wram.write(address, data)
	case address < 0x2000:
		b.wram.write(address-0x0800, data)
	case address < 0x2008:
		b.writeToPPURegisters(address, data)
	case address == 0x4016: // 1P
		b.controller.write(data)
	default:
		glog.Fatalf("Unknown CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	}
}
