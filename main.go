package main

import "fmt"

type Status struct {
	N bool // negative flag
	V bool // overflow flag
	R bool // reserved flag
	B bool // break flag
	D bool // decimal flag
	I bool // IRQ flag
	Z bool // zero flag
	C bool // carry flag
}

func newStatus() *Status {
	return &Status{false, false, true, true, false, true, false, false}
}

type CPU struct {
	PC uint16  // program counter
	A  byte    // accumulator register
	X  byte    // index register
	Y  byte    // index register
	S  byte    // stack pointer
	P  *Status // processor status bits
}

func newCPU() *CPU {
	status := newStatus()
	return &CPU{0, 0, 0, 0, 0, status}
}

func main() {
	cpu := newCPU()
	fmt.Println(cpu)
}
