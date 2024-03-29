package ui

import (
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/golang/glog"

	"github.com/jyane/jnes/nes"
)

func mainLoop(window *glfw.Window, console nes.Console, program uint32, audio *audio) {
	for range time.Tick(16 * time.Millisecond) {
		currentCycles := 0
		for currentCycles < nes.CPUFrequency/60 {
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
	audio := newAudio()
	console.SetAudioOut(audio.channel)
	if err := audio.start(); err != nil {
		glog.Fatalln(err)
	}
	defer audio.terminate()
	mainLoop(window, console, program, audio)
}
