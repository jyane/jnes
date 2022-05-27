package nes

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"regexp"
	"strconv"
	"strings"
)

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

// DebugConsole is for debug, you can execute some commands through stdio.
// commands:
//   s:
//     execute step(s).
//   p:
//     print.
//   br:
//     set a break point.
//   q:
//     quit.
//   r:
//     reset.
type DebugConsole struct {
	*NesConsole
	lastFrame    uint64
	currentFrame uint64
	buffer       *image.RGBA
	breakpoints  []uint16
}

func (c *DebugConsole) Reset() error {
	c.lastFrame = 0
	c.currentFrame = 0
	if err := c.cpu.Reset(); err != nil {
		return err
	}
	c.ppu.Reset()
	return nil
}

func (c *DebugConsole) step() (int, error) {
	cycles, err := c.cpu.Step()
	if err != nil {
		return cycles, err
	}
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

func (c *DebugConsole) basePrint() {
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Rendered frame: %d\n", c.currentFrame)
	fmt.Println("Last exec: " + c.cpu.lastExecution)
	fmt.Printf("CPU: PC=0x%04x, A=0x%02x, X=0x%02x, Y=0x%02x, S=0x%02x\n",
		c.cpu.pc, c.cpu.a, c.cpu.x, c.cpu.y, c.cpu.s)
	fmt.Printf("PPU: cycle=%d, scanline=%d, p.v=0x%04x\n",
		c.ppu.cycle, c.ppu.scanline, c.ppu.v)
}

func (c *DebugConsole) printCommand(args []string) {
	if len(args) < 2 {
		c.basePrint()
	} else {
		switch args[1] {
		case "c", "cpu":
			fmt.Printf("%+v\n", *c.cpu)
		case "p", "ppu":
			fmt.Printf("%+v\n", *c.ppu)
		case "ca", "cartridge":
			fmt.Printf("%+v\n", *c.cpu.bus.cartridge)
		case "ct", "controller":
			fmt.Printf("%+v\n", *c.controller)
		case "wr", "wram":
			fmt.Printf("%+v\n", *c.cpu.bus.wram)
		case "vr", "vram":
			fmt.Printf("%+v\n", *c.ppu.bus.vram)
		}
	}
}

func (c *DebugConsole) checkBreak() bool {
	for i := 0; i < len(c.breakpoints); i++ {
		if c.breakpoints[i] == c.cpu.pc {
			fmt.Printf("Break at: 0x%04x\n", c.breakpoints[i])
			return true
		}
	}
	return false
}

func (c *DebugConsole) stepCommand(args []string) (int, error) {
	if len(args) < 2 {
		return c.step()
	} else {
		re := regexp.MustCompile("^([0-9]+)")
		if re.MatchString(args[1]) {
			num, _ := strconv.Atoi(re.FindString(args[1]))
			unit := args[1][len(args[1])-1]
			cycles := 0
			switch unit {
			case 's':
				// s means seconds but this doesn't execute 1 sec, this executes CPUFrequency * num
				// This will be 60 * num frames execution.
				steps := CPUFrequency * num
				for cycles < steps {
					v, err := c.step()
					if err != nil {
						return cycles, err
					}
					cycles += v
					if c.checkBreak() {
						return cycles, nil
					}
				}
			case 'd':
				// debug -> steps with debug messages.
				for i := 0; i < num; i++ {
					v, err := c.step()
					c.basePrint()
					if err != nil {
						return cycles, err
					}
					cycles += v
					if c.checkBreak() {
						return cycles, nil
					}
				}
			default: // no unit -> step
				for i := 0; i < num; i++ {
					v, err := c.step()
					if err != nil {
						return cycles, err
					}
					cycles += v
					if c.checkBreak() {
						return cycles, nil
					}
				}
			}
			return cycles, nil
		}
	}
	return 0, nil
}

func (c *DebugConsole) breakPointCommand(args []string) error {
	var i int
	fmt.Sscanf(args[1], "0x%x\n", &i)
	c.breakpoints = append(c.breakpoints, uint16(i))
	return nil
}

func (c *DebugConsole) quitCommand() {
	fmt.Println("Quitting.")
	os.Exit(0)
}

func (c *DebugConsole) Step() (int, error) {
	fmt.Printf("Debugger mode, 'q' to quit \n>> ")
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		return 0, err
	}
	args := strings.Split(strings.TrimSuffix(line, "\n"), " ")
	command := args[0]
	switch command {
	case "p", "print":
		c.printCommand(args)
	case "s", "step":
		cycles, err := c.stepCommand(args)
		c.basePrint() // Print data before it die.
		if err != nil {
			return cycles, err
		}
		fmt.Printf("Executed %d CPU cycles, %d PPU cycles.\n", cycles, 3*cycles)
		return cycles, nil
	case "br", "breakpoint":
		if err := c.breakPointCommand(args); err != nil {
			return 0, err
		}
	case "r", "reset":
		c.Reset()
	case "q", "quit":
		c.quitCommand()
	default:
		return 0, fmt.Errorf("Unkonwn command %s", line)
	}
	// step command was not executed.
	return 0, nil
}

func (c *DebugConsole) Frame() (*image.RGBA, bool) {
	if c.lastFrame < c.currentFrame {
		c.lastFrame = c.currentFrame
		return c.buffer, true
	} else {
		return c.buffer, false
	}
}

func (c *DebugConsole) SetButtons(buttons [8]bool) {
	c.controller.Set(buttons)
}
