package ui

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/jyane/jnes/nes"
)

// getKey gets the state of keyboard, WASD for directions, J for primary.
func getKeys(window *glfw.Window) [8]bool {
	var keys [8]bool
	keys[nes.ButtonRight] = window.GetKey(glfw.KeyD) == glfw.Press
	keys[nes.ButtonLeft] = window.GetKey(glfw.KeyA) == glfw.Press
	keys[nes.ButtonDown] = window.GetKey(glfw.KeyS) == glfw.Press
	keys[nes.ButtonUp] = window.GetKey(glfw.KeyW) == glfw.Press
	keys[nes.ButtonStart] = window.GetKey(glfw.KeyG) == glfw.Press
	keys[nes.ButtonSelect] = window.GetKey(glfw.KeyF) == glfw.Press
	keys[nes.ButtonB] = window.GetKey(glfw.KeyH) == glfw.Press
	keys[nes.ButtonA] = window.GetKey(glfw.KeyJ) == glfw.Press
	return keys
}
