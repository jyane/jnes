package main

import "testing"

const PC = 0xFFFC
const AD = 0xFFFD

func createTestCPU(opcode byte) *CPU {
	var rom = make([]byte, 0xFFFF, 0xFFFF)
	// reset address
	rom[PC-0x8000] = 0xFD
	rom[PC-0x8000+1] = 0xFF
	rom[AD-0x8000] = opcode
	bus := NewCPUBus(NewRAM(), rom)
	cpu := NewCPU(bus)
	cpu.Reset()
	return cpu
}

// func TestReset(t *testing.T) {
// rom := make([]byte, 0xFFFF, 0xFFFF)
// rom[0xFFFC
// bus := NewCPUBus(NewRAM(), rom)
// cpu := NewCPU(bus)
// cpu.Reset()
// if got := cpu.PC; got != 0xFFFC {
// t.Errorf("cpu.PC has invalid value=%v", got)
// }
// }

// http://obelisk.me.uk/6502/reference.html

func TestAdc(t *testing.T) {
	cpu := createTestCPU(0x69)
	if got := cpu.Step(); got != 2 {
		t.Errorf("want=%v, got=%v", 2, got)
	}
	if got := cpu.PC; got != AD+2 {
		t.Errorf("want=%v, got=%v", AD+2, got)
	}
}

func TestNop(t *testing.T) {
	cpu := createTestCPU(0xEA)
	if got := cpu.Step(); got != 2 {
		t.Errorf("want=%v, got=%v", 2, got)
	}
	if got := cpu.PC; got != 1 {
		t.Errorf("want=%v, got=%v", AD+1, got)
	}
}
