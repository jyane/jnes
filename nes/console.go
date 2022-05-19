package nes

import "log"

type Console struct {
	CPU *CPU
	PPU *PPU
}

func NewConsole(buf []byte) *Console {
	cartridge, err := NewCartridge(buf)
	if err != nil {
		log.Fatalln(err)
	}
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	cpuBus := NewCPUBus(NewRAM(), ppu, cartridge)
	cpu := NewCPU(cpuBus)
	return &Console{cpu, ppu}
}
