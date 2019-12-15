package main

import "testing"

func createTestCPU(opcode byte) *CPU {
	var rom = make([]byte, 0xFFFF, 0xFFFF)
	// reset address
	rom[0x7ffc] = 0x00
	rom[0x7ffd] = 0x01
	rom[0x0001] = opcode
	bus := NewCPUBus(NewRAM(), rom)
	return NewCPU(bus)
}

// http://obelisk.me.uk/6502/reference.html

func TestAdc(t *testing.T) {
	cpu := createTestCPU(0x00)
	cycles := cpu.Step()
	if cycles != 42 {
		t.Fail()
	}
}
