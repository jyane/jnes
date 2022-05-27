package ui

import (
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/golang/glog"

	"github.com/jyane/jnes/nes"
)

func mainLoop(window *glfw.Window, console nes.Console, program uint32) {
	// TODO(jyane): Currently this syncs within a second, this is probably too fast to render 60 frames.
	for range time.Tick(1 * time.Second) {
		currentCycles := 0
		for currentCycles < nes.CPUFrequency {
			cycles, err := console.Step()
			if err != nil {
				glog.Fatalln(err)
			}
			frame, ok := console.Frame()
			if ok {
				updateTexture(program, frame)
				window.SwapBuffers()
				glfw.PollEvents()
				console.SetButtons(getKeys(window))
			}
			currentCycles += cycles
		}
		if window.ShouldClose() {
			return
		}
	}
}

// Start is the main entrypoint.
func Start(console nes.Console, width int, height int) {
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
