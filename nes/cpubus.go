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

// writeOAMDMA writes OAMDATA to PPU, this will be called by CPU.
func (b *CPUBus) writeOAMDMA(data [256]byte) {
	b.ppu.oamData = data
}

func (b *CPUBus) readPPURegister(address uint16) byte {
	switch address {
	case 0x2002:
		return b.ppu.readPPUSTATUS()
	case 0x2004:
		return b.ppu.readOAMDATA()
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
	case address < 0x2000:
		return b.wram.read(address % 0x0800)
	case address < 0x2008:
		return b.readPPURegister(address)
	case address == 0x4016: // 1P
		return b.controller.read()
	case address < 0x4020:
		glog.Infof("Unimplemented CPU bus read: address=0x%04x\n", address)
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
		b.ppu.writePPUDATA(data)
	default:
		glog.Fatalf("Unkonwn CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	}
}

// write writes a byte.
func (b *CPUBus) write(address uint16, data byte) {
	switch {
	case address < 0x2000:
		b.wram.write(address%0x0800, data)
	case address < 0x2008:
		b.writeToPPURegisters(address, data)
	case address == 0x4014:
		// Implemented on CPU
		glog.Fatalf("CPU bus write was probably illegally called. (Here is for writing oamdma $4014)")
	case address == 0x4016: // 1P
		b.controller.write(data)
	case address < 0x4020:
		glog.Infof("Unimplemented CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	case 0x8000 <= address:
		glog.Infof("PrgROM write: address=0x%04x, data=0x%02x\n", address, data)
	default:
		glog.Fatalf("Unknown CPU bus write: address=0x%04x, data=0x%02x\n", address, data)
	}
}
