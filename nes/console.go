package nes

import "image"

type Console interface {
	Reset() error
	Step() (int, error)
	Frame() (*image.RGBA, bool)
	SetButtons([8]bool)
}

type NesConsole struct {
	cpu          *CPU
	ppu          *PPU
	controller   *Controller
	lastFrame    uint64
	currentFrame uint64
	buffer       *image.RGBA
}

// NewConsole creates a console. If debug is true, this creates a debug console.
func NewConsole(buf []byte, debug bool) (Console, error) {
	cartridge, err := NewCartridge(buf)
	if err != nil {
		return nil, err
	}
	controller := NewController()
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	cpuBus := NewCPUBus(NewRAM(), ppu, cartridge, controller)
	cpu := NewCPU(cpuBus)
	console := &NesConsole{cpu: cpu, ppu: ppu, controller: controller}
	if debug {
		return &DebugConsole{NesConsole: console}, nil
	} else {
		return console, nil
	}
}

func (c *NesConsole) Reset() error {
	c.currentFrame = 0
	c.lastFrame = 0
	if err := c.cpu.Reset(); err != nil {
		return err
	}
	c.ppu.Reset()
	return nil
}

// Step executes a CPU step and returns how many cycles are consumed.
func (c *NesConsole) Step() (int, error) {
	cycles, err := c.cpu.Step()
	if err != nil {
		return cycles, err
	}
	// PPU's clock is exactly 3x faster than CPU's
	for i := 0; i < cycles*3; i++ {
		nmi, err := c.ppu.Step()
		if err != nil {
			return cycles, err
		}
		if nmi {
			c.cpu.nmiTriggered = true
		}
		ok, f := c.ppu.Frame()
		if ok {
			c.currentFrame++
			c.buffer = f
		}
	}
	return cycles, nil
}

// Frame returns a new frame.
func (c *NesConsole) Frame() (*image.RGBA, bool) {
	if c.lastFrame < c.currentFrame {
		c.lastFrame = c.currentFrame
		return c.buffer, true
	} else {
		return c.buffer, false
	}
}

func (c *NesConsole) SetButtons(buttons [8]bool) {
	c.controller.Set(buttons)
}
