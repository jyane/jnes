package nes

// Reference:
//   http://hp.vector.co.jp/authors/VA042397/nes/joypad.html (In Japanese)
//   https://www.nesdev.org/wiki/Controller_reading
//   https://www.nesdev.org/wiki/Controller_reading_code

type button int

// Controller bit assignments, 1 means pressed otherwise 0.
// bit    7 6      5     4  3    2    1     0
// button A B Select Start Up Down Left Right
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
	strobe  byte
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Set(buttons [8]bool) {
	c.buttons = buttons
}

func (c *Controller) read() byte {
	ret := byte(0)
	if c.index < 8 && c.buttons[c.index] {
		ret = 1
	}
	c.index++
	if c.strobe&1 == 1 {
		c.index = 0
	}
	return ret
}

// write writes strobe.
// https://bugzmanov.github.io/nes_ebook/chapter_7.html
// - strobe bit on - controller reports only status of the button A on every read
// - strobe bit off - controller cycles through all buttons
func (c *Controller) write(data byte) {
	c.strobe = data
	if c.strobe&1 == 1 {
		c.index = 0
	}
}
