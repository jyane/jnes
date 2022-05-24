package ui

import (
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/golang/glog"

	"github.com/jyane/jnes/nes"
)

func mainLoop(window *glfw.Window, console *nes.Console, program uint32) {
	for range time.Tick(1 * time.Second) {
		var currentCycles = 0
		var nmi bool = false
		for currentCycles < nes.CPUFrequency {
			cycles := console.CPU.Do(nmi)
			nmi = false
			// PPU's clock is 3x faster than CPU's
			for i := 0; i < cycles*3; i++ {
				// If PPU prepared an image to render, OpenGL updates a 2D texture.
				prepared, image := console.PPU.Do()
				if !nmi && console.PPU.CheckNMI() {
					nmi = true
				}
				// Here will be executed (almost) 60 times per second.
				if prepared {
					updateTexture(program, image)
					window.SwapBuffers()
					glfw.PollEvents()
					console.Controller.Set(getKeys(window))
				}
			}
			currentCycles += cycles
		}
		if window.ShouldClose() {
			return
		}
	}
}

// Start is the main entrypoint.
func Start(console *nes.Console, width int, height int) {
	err := glfw.Init()
	if err != nil {
		glog.Fatalln(err)
	}
	defer glfw.Terminate()
	window, err := glfw.CreateWindow(width, height, "JNES", nil, nil)
	if err != nil {
		glog.Fatalln(err)
	}
	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		glog.Fatalln(err)
	}
	program, err := newProgram()
	if err != nil {
		glog.Fatalln(err)
	}
	gl.UseProgram(program)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	mainLoop(window, console, program)
}
