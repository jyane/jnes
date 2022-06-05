package nes

import "fmt"

type mapper0 struct {
	prgROM []byte
	chrROM []byte
}

// Mapper0: https://www.nesdev.org/wiki/NROM

// currently only supports mapper0.
func (m *mapper0) ReadFromCPU(address uint16) (byte, error) {
	if 0x8000 <= address {
		// CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
		mod := uint16(len(m.prgROM))
		return m.prgROM[(address-0x8000)%mod], nil
	}
	// CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
	return 0, fmt.Errorf("Reading PRGRAM not implemented. address: 0x%04x", address)
}

func (m *mapper0) WriteFromCPU(address uint16, data byte) error {
	if 0x8000 <= address {
		return fmt.Errorf("Writing data to PrgROM not allowed: address=0x%04x, data=0x%02x", address, data)
	}
	// CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
	return fmt.Errorf("Writing data to PRGRAM not implemented. address: 0x%04x, data: 0x%02x", address, data)
}

func (m *mapper0) ReadFromPPU(address uint16) (byte, error) {
	return m.chrROM[address], nil
}

func (m *mapper0) WriteFromPPU(address uint16, data byte) error {
	return fmt.Errorf("Writing data to pattern tables not allowed, address=0x%04x, data=0x%02x", address, data)
}
