package nes

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCPU(t *testing.T) {
	f, _ := os.Open("../testdata/other/nestest.nes")
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	cartridge, _ := NewCartridge(b)
	controller := NewController()
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	cpuBus := NewCPUBus(NewRAM(), ppu, cartridge, controller)
	c := NewCPU(cpuBus)
	c.pc = 0xC000 // test rom can be run as headless if PC is starting from 0xC000
}
