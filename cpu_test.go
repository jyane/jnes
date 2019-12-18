package main

import "testing"

type MockBus struct {
	Bus
	MockRead func(address uint16) byte
}

func (bus *MockBus) Read(address uint16) byte {
	return bus.MockRead(address)
}

func createTestCPU(opcode byte) *CPU {
	bus := &MockBus{
		MockRead: func(address uint16) byte {
			// If the cpu is reset state, returns 0x00 as a starting point.
			if address == 0xFFFC || address == 0xFFFD {
				return 0x00
			}
			return opcode
		},
	}
	return NewCPU(bus)
}

// http://obelisk.me.uk/6502/reference.html

func TestInstructions(t *testing.T) {
	for _, test := range []struct {
		opcode byte
		cycle  int
		size   uint16
		A      byte
		X      byte
		Y      byte
		S      byte
		P      *Status
	}{
		// ADC
		{
			opcode: 0x69,
			cycle:  2,
			size:   2,
			A:      0x00,
			X:      0x00,
			Y:      0x00,
			S:      0xFD,
			P:      NewStatus(),
		},
		// NOP
		{
			opcode: 0xEA,
			cycle:  2,
			size:   1,
			A:      0x00,
			X:      0x00,
			Y:      0x00,
			S:      0xFD,
			P:      NewStatus(),
		},
	} {
		cpu := createTestCPU(test.opcode)
		cpu.Reset()
		if got := cpu.Step(); got != test.cycle {
			t.Errorf("CPU cycle want=%v, got=%v", test.cycle, got)
		}
		if got := cpu.PC; got != test.size {
			t.Errorf("CPU.PC want=%v, got=%v", test.size, got)
		}
		if got := cpu.A; got != test.A {
			t.Errorf("CPU.A want=%v, got=%v", test.A, got)
		}
		if got := cpu.X; got != test.X {
			t.Errorf("CPU.X want=%v, got=%v", test.X, got)
		}
		if got := cpu.Y; got != test.Y {
			t.Errorf("CPU.Y want=%v, got=%v", test.Y, got)
		}
		if got := cpu.P; &test.P == &got {
			t.Errorf("CPU.P want=%v, got=%v", test.P, got)
		}
	}
}
