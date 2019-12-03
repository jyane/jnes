package main

type RAM struct {
	ram [2048]byte
}

func NewRAM() *RAM {
	return &RAM{[2048]byte{}}
}
