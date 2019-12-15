package main

import "log"

type CPUBus struct {
	wram   *RAM
	prgROM []byte
}

func NewCPUBus(wram *RAM, prgROM []byte) *CPUBus {
	return &CPUBus{wram, prgROM}
}

// CPU memory map
// 0x0000 - 0x07FF	WRAM
// 0x0800 - 0x1FFF	Unused 0x0000-0x07FF
// 0x2000 - 0x2007	I/O Port (PPU)
// 0x2008 - 0x3FFF	Unused 0x2000-0x2007
// 0x4000 - 0x401F	I/O Port
// 0x4020 - 0x5FFF	Extended RAM
// 0x6000 - 0x7FFF	Battery Backup RAM
// 0x8000 - 0xBFFF	ProgramROM Low
// 0xC000 - 0xFFFF	ProgramROM High
func (bus *CPUBus) Read(address uint16) byte {
	switch {
	case address < 0x0800:
		return bus.wram.Read(address)
	case address < 0x2000:
		return bus.wram.Read(address - 0x0800)
	case 0x8000 <= address:
		return bus.prgROM[address-0x8000]
	default:
		log.Printf("Unknown bus reference from CPU: 0x%04x\n", address)
	}
	return 0
}

func (bus *CPUBus) Read16(address uint16) uint16 {
	low := uint16(bus.Read(address))
	high := uint16(bus.Read(address+1)) << 8 // e.g. 11011011 00000000
	return high | low
}
