package nes

import "log"

type Console struct {
	CPU        *CPU
	PPU        *PPU
	Controller *Controller
}

func NewConsole(buf []byte) *Console {
	cartridge, err := NewCartridge(buf)
	if err != nil {
		log.Fatalln(err)
	}
  controller := NewController()
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	cpuBus := NewCPUBus(NewRAM(), ppu, cartridge, controller)
	cpu := NewCPU(cpuBus)
	return &Console{cpu, ppu, controller}
}
