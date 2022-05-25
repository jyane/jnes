package nes

// Reference:
//   http://hp.vector.co.jp/authors/VA042397/nes/joypad.html (In Japanese)
//   https://www.nesdev.org/wiki/Controller_reading
//   https://www.nesdev.org/wiki/Controller_reading_code

type button int

const (
	ButtonA button = iota
	ButtonB
	ButtonSelect
	ButtonStart
	ButtonUp
	ButtonDown
	ButtonLeft
	ButtonRight
)

type Controller struct {
	buttons [8]bool
	index   byte
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Set(buttons [8]bool) {
	c.buttons = buttons
}

func (c *Controller) read() byte {
	c.index++
	if c.index == 8 {
		c.index = 0
		return 0
	}
	if c.buttons[c.index] {
		return 1
	}
	return 0
}

func (c *Controller) write(data byte) {
}
