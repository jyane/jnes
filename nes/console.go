package nes

import (
	"bufio"
	"fmt"
	"image"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Console interface {
	Reset()
	Step() int
	Frame() (bool, *image.RGBA)
	SetButtons([8]bool)
}

type NesConsole struct {
	CPU          *CPU
	PPU          *PPU
	Controller   *Controller
	lastFrame    uint64
	currentFrame uint64
	buffer       *image.RGBA
}

func NewConsole(buf []byte, debug bool) Console {
	cartridge, err := NewCartridge(buf)
	if err != nil {
		log.Fatalln(err)
	}
	controller := NewController()
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	cpuBus := NewCPUBus(NewRAM(), ppu, cartridge, controller)
	cpu := NewCPU(cpuBus)
	if debug {
		return &DebugConsole{CPU: cpu, PPU: ppu, Controller: controller}
	} else {
		return &NesConsole{CPU: cpu, PPU: ppu, Controller: controller}
	}
}

func (c *NesConsole) Reset() {
	c.currentFrame = 0
	c.lastFrame = 0
	c.CPU.Reset()
	c.PPU.Reset()
}

// Step executes a CPU step and returns how many cycles are consumed.
func (c *NesConsole) Step() int {
	cycles := c.CPU.Do()
	// PPU's clock is exactly 3x faster than CPU's
	for i := 0; i < cycles*3; i++ {
		nmi := c.PPU.Do()
		if nmi {
			c.CPU.nmiTriggered = true
		}
		ok, f := c.PPU.Frame()
		if ok {
			c.currentFrame++
			c.buffer = f
		}
	}
	return cycles
}

// Frame returns a new frame.
func (c *NesConsole) Frame() (bool, *image.RGBA) {
	if c.lastFrame < c.currentFrame {
		c.lastFrame = c.currentFrame
		return true, c.buffer
	} else {
		return false, c.buffer
	}
}

func (c *NesConsole) SetButtons(buttons [8]bool) {
	c.Controller.Set(buttons)
}

// DebugConsole is for debug.
// s:
//   executes step(s).
// p:
//   print.
// q:
//   quit.
// r:
//   reset.

type DebugConsole struct {
	CPU          *CPU
	PPU          *PPU
	Controller   *Controller
	lastFrame    uint64
	currentFrame uint64
	buffer       *image.RGBA
}

func (c *DebugConsole) Reset() {
	c.lastFrame = 0
	c.currentFrame = 0
	c.CPU.Reset()
	c.PPU.Reset()
}

func (c *DebugConsole) step() int {
	cycles := c.CPU.Do()
	for i := 0; i < cycles*3; i++ {
		nmi := c.PPU.Do()
		if nmi {
			c.CPU.nmiTriggered = true
		}
		ok, f := c.PPU.Frame()
		if ok {
			c.currentFrame++
			c.buffer = f
		}
	}
	return cycles
}

func (c *DebugConsole) basePrint() {
	fmt.Printf("Rendered frame: %d\n", c.currentFrame)
	fmt.Println("Last exec: " + c.CPU.lastExecution)
	fmt.Printf("CPU: PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x\n",
		c.CPU.PC, c.CPU.A, c.CPU.X, c.CPU.Y, c.CPU.S)
	fmt.Printf("PPU: cycle=%d, scanline=%d, p.v=0x%04x\n",
		c.PPU.cycle, c.PPU.scanline, c.PPU.v)
}

func (c *DebugConsole) printCommand(args []string) {
	if len(args) < 2 {
		c.basePrint()
	} else {
		switch args[1] {
		case "c", "cpu":
			fmt.Printf("%+v\n", *c.CPU)
		case "p", "ppu":
			fmt.Printf("%+v\n", *c.PPU)
		case "ca", "cartridge":
			fmt.Printf("%+v\n", *c.CPU.bus.cartridge)
		case "ct", "controller":
			fmt.Printf("%+v\n", *c.Controller)
		case "wr", "wram":
			fmt.Printf("%+v\n", *c.CPU.bus.wram)
		case "vr", "vram":
			fmt.Printf("%+v\n", *c.PPU.bus.vram)
		}
	}
}

func (c *DebugConsole) stepCommand(args []string) int {
	if len(args) < 2 {
		return c.step()
	} else {
		re := regexp.MustCompile("^([0-9]+)")
		if re.MatchString(args[1]) {
			num, _ := strconv.Atoi(re.FindString(args[1]))
			unit := args[1][len(args[1])-1]
			switch unit {
			case 's':
				// s means seconds but this doesn't execute 1 sec, this executes CPUFrequency * num
				// This will be 60 * num frames execution.
				steps := CPUFrequency * num
				cycles := 0
				for cycles < steps {
					cycles += c.step()
				}
				return cycles
			default: // no unit -> step
				cycles := 0
				for i := 0; i < num; i++ {
					cycles += c.step()
				}
				return cycles
			}
		}
	}
	// ?
	return 0
}

func (c *DebugConsole) quitCommand() {
	fmt.Println("Quitting.")
	os.Exit(0)
}

func (c *DebugConsole) Step() int {
	fmt.Printf("Debugger mode, 'q' to quit -----------------------------------------------\n>> ")
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read debug msg.")
		os.Exit(1)
	}
	args := strings.Split(strings.TrimSuffix(line, "\n"), " ")
	command := args[0]
	switch command {
	case "p", "print":
		c.printCommand(args)
	case "s", "step":
		cycles := c.stepCommand(args)
		c.basePrint()
		fmt.Printf("Executed %d CPU cycles, %d PPU cycles.\n", cycles, 3*cycles)
		return cycles
	case "r", "reset":
		c.Reset()
	case "q", "quit":
		c.quitCommand()
	default:
		fmt.Printf("Unkonwn command %s\n", line)
	}
	// step command was not executed.
	return 0
}

func (c *DebugConsole) Frame() (bool, *image.RGBA) {
	if c.lastFrame < c.currentFrame {
		c.lastFrame = c.currentFrame
		return true, c.buffer
	} else {
		return false, c.buffer
	}
}

func (c *DebugConsole) SetButtons(buttons [8]bool) {
	c.Controller.Set(buttons)
}
