package main

type PPU struct {
	vram *RAM
}

func NewPPU(vram *RAM) *PPU {
	return &PPU{vram}
}
