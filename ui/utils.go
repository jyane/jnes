package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/jyane/jnes/nes"
)

// Shaders for a 2D texture.
const (
	vertexShader = `
  #version 330

  attribute vec3 position;
  attribute vec2 uv;
  varying vec2 vuv;
  void main(void){
    gl_Position = vec4(position, 1.0);
    vuv = uv;
  }
  ` + "\x00"

	fragmentShader = `
  #version 330

  varying vec2 vuv;
  uniform sampler2D texture;
  void main(void){
    gl_FragColor = texture2D(texture, vuv);
  }
  ` + "\x00"
)

// compileShader comples a shader.
func compileShader(code string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	ccode := gl.Str(code)
	gl.ShaderSource(shader, 1, &ccode, nil)
	gl.CompileShader(shader)
	var result int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &result)
	if result == gl.FALSE {
		var length int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &length)
		log := strings.Repeat("\x00", int(length+1))
		gl.GetShaderInfoLog(shader, length, nil, gl.Str(log))
		return 0, fmt.Errorf("Failed to compile a shader: %v\n %v", code, log)
	}
	return shader, nil
}

// newProgram creates a new program.
func newProgram() (uint32, error) {
	vertexShader, err := compileShader(vertexShader, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentShader, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	var result int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &result)
	if result == gl.FALSE {
		var length int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &length)
		log := strings.Repeat("\x00", int(length+1))
		gl.GetProgramInfoLog(program, length, nil, gl.Str(log))
		return 0, fmt.Errorf("Failed to link a program: %v", log)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	return program, nil
}

var vertexPosition = []float32{
	1, 1,
	-1, 1,
	-1, -1,
	1, -1,
}
var vertexUV = []float32{
	1, 0,
	0, 0,
	0, 1,
	1, 1,
}

// updateTexture updates a texture.
func updateTexture(program uint32, image *image.RGBA) {
	var textureId uint32
	gl.GenTextures(1, &textureId)
	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA,
		int32(image.Rect.Size().X), int32(image.Rect.Size().Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(image.Pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	positionLocation := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	uvLocation := uint32(gl.GetAttribLocation(program, gl.Str("uv\x00")))
	textureLocation := gl.GetUniformLocation(program, gl.Str("texture\x00"))
	gl.EnableVertexAttribArray(positionLocation)
	gl.EnableVertexAttribArray(uvLocation)
	gl.Uniform1i(textureLocation, 0)
	gl.VertexAttribPointer(positionLocation, 2, gl.FLOAT, false, 0, gl.Ptr(vertexPosition))
	gl.VertexAttribPointer(uvLocation, 2, gl.FLOAT, false, 0, gl.Ptr(vertexUV))
	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)
}

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
