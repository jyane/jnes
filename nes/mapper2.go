package nes

import "fmt"

type mapper2 struct {
	banks       int
	currentBank int
	prgROM      []byte
	chrROM      []byte
}

// Mapper2: https://www.nesdev.org/wiki/UxROM

func NewMapper2(prgROM []byte) *mapper2 {
	banks := len(prgROM) / prgROMSizeUnit
	m := &mapper2{banks: banks, prgROM: prgROM, chrROM: make([]byte, 0x4000)}
	return m
}

func (m *mapper2) ReadFromCPU(address uint16) (byte, error) {
	// CPU $8000-$BFFF: 16 KB switchable PRG ROM bank
	// CPU $C000-$FFFF: 16 KB PRG ROM bank, fixed to the last bank
	if address < 0xC000 {
		i := m.currentBank*prgROMSizeUnit + int(address-0x8000)
		return m.prgROM[i], nil
	} else {
		// fixed bank
		i := (m.banks-1)*prgROMSizeUnit + int(address-0xC000)
		return m.prgROM[i], nil
	}
}

func (m *mapper2) WriteFromCPU(address uint16, data byte) error {
	// CPU $8000-$BFFF: 16 KB switchable PRG ROM bank
	// CPU $C000-$FFFF: 16 KB PRG ROM bank, fixed to the last bank
	if 0x8000 <= address {
		m.currentBank = int(data) % m.banks
		return nil
	}
	return fmt.Errorf("Writing cartridge address 0x%04x = 0x%02x is not allowed", address, data)
}

func (m *mapper2) ReadFromPPU(address uint16) (byte, error) {
	return m.chrROM[address], nil
}

func (m *mapper2) WriteFromPPU(address uint16, data byte) error {
	m.chrROM[address] = data
	return nil
}
