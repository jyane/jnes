package main

import "testing"

type MockBus struct {
	Bus
	MockRead  func(address uint16) byte
	MockWrite func(address uint16)
}

func (bus *MockBus) Read(address uint16) byte {
	return bus.MockRead(address)
}

func (bus *MockBus) Write(address uint16) {
	bus.MockWrite(address)
}

func createTestCPU(opcode byte) *CPU {
	bus := &MockBus{
		MockRead: func(address uint16) byte {
			return opcode
		},
		MockWrite: func(address uint16) {
		},
	}
	return NewCPU(bus)
}

// http://obelisk.me.uk/6502/reference.html

func TestInstructions(t *testing.T) {
	var initialA byte = 0x80
	var initialX byte = 0x81
	var initialY byte = 0x82
	var initialS byte = 0xFD
	for _, test := range []struct {
		name   string
		opcode byte
		cycle  int
		size   uint16
		A      byte
		X      byte
		Y      byte
		S      byte
		P      *Status
	}{
		{
			name:   "ADC",
			opcode: 0x69,
			cycle:  2,
			size:   1,
			A:      initialA,
			X:      initialX,
			Y:      initialY,
			S:      initialS,
			P:      NewStatus(),
		},
		{
			name:   "NOP",
			opcode: 0xEA,
			cycle:  2,
			size:   1,
			A:      initialA,
			X:      initialX,
			Y:      initialY,
			S:      initialS,
			P:      NewStatus(),
		},
	} {
		cpu := createTestCPU(test.opcode)
		// sets values for testing.
		cpu.A = initialA
		cpu.X = initialX
		cpu.Y = initialY
		cpu.S = initialS
		cpu.P = NewStatus()
		cpu.PC = 0x00
		if got := cpu.Step(); got != test.cycle {
			t.Errorf("%v, CPU cycle want=%v, got=%v", test.name, test.cycle, got)
		}
		if got := cpu.PC; got != test.size {
			t.Errorf("%v, CPU.PC want=%v, got=%v", test.name, test.size, got)
		}
		if got := cpu.A; got != test.A {
			t.Errorf("%v, CPU.A want=%v, got=%v", test.name, test.A, got)
		}
		if got := cpu.X; got != test.X {
			t.Errorf("%v, CPU.X want=%v, got=%v", test.name, test.X, got)
		}
		if got := cpu.Y; got != test.Y {
			t.Errorf("%v, CPU.Y want=%v, got=%v", test.name, test.Y, got)
		}
		if got := cpu.P; &test.P == &got {
			t.Errorf("%v, CPU.P want=%v, got=%v", test.name, test.P, got)
		}
	}
}
