package nes

import "fmt"

const (
	chrROMSizeUnit      int  = 0x2000 // 8 bytes
	prgROMSizeUnit      int  = 0x4000 // 16 bytes
	inesHeaderSizeBytes int  = 16     // The valid INES header has 16 bytes
	msDOSEOF            byte = 0x1A
)

type tableMirrorMode int

const (
	horizontal tableMirrorMode = iota
	vertical
)

// https://www.nesdev.org/wiki/INES
type Cartridge struct {
	prgROM  []byte
	chrROM  []byte
	flags6  byte // https://www.nesdev.org/wiki/INES#Flags_6
	flags7  byte // https://www.nesdev.org/wiki/INES#Flags_7
	flags8  byte // https://www.nesdev.org/wiki/INES#Flags_8
	flags9  byte // https://www.nesdev.org/wiki/INES#Flags_9
	flags10 byte // https://www.nesdev.org/wiki/INES#Flags_10
}

// IsValid checks whether the cartridge is valid INES format.
func isValid(data []byte) bool {
	if len(data) >= inesHeaderSizeBytes &&
		data[0] == byte('N') &&
		data[1] == byte('E') &&
		data[2] == byte('S') &&
		data[3] == msDOSEOF {
		return true
	} else {
		return false
	}
}

// ReadPRGROM retrieves Program ROM from cartridge.
func readPRGROM(data []byte) []byte {
	var l = inesHeaderSizeBytes
	var r = inesHeaderSizeBytes + int(data[4])*prgROMSizeUnit
	return data[l:r]
}

// ReadCHRROM retrieves Character ROM from cartridge.
func readCHRROM(data []byte) []byte {
	var l = inesHeaderSizeBytes + int(data[4])*prgROMSizeUnit
	var r = l + int(data[5])*chrROMSizeUnit
	return data[l:r]
}

func (c *Cartridge) mirror() tableMirrorMode {
	if c.flags6&1 == 1 {
		return vertical
	} else {
		return horizontal
	}
}

func (c *Cartridge) mapper() byte {
	l := c.flags6 & 0xF0
	h := c.flags7 & 0xF0
	return h | (l >> 4)
}

// NewCartridge creates a cartridge.
func NewCartridge(data []byte) (*Cartridge, error) {
	c := &Cartridge{}
	if !isValid(data) {
		return nil, fmt.Errorf("The buffer is not a valid NES format.")
	}
	c.prgROM = readPRGROM(data)
	c.chrROM = readCHRROM(data)
	c.flags6 = data[6]
	c.flags7 = data[7]
	c.flags8 = data[8]
	c.flags9 = data[9]
	c.flags10 = data[10]
	// TODO(jyane): Implement mappers, currently this cartridge only supports mapper0.
	if c.mapper() != 0 {
		return nil, fmt.Errorf("Mapper%d is not implemented.", c.mapper())
	}
	return c, nil
}

// Mapper0: https://www.nesdev.org/wiki/NROM

// currently only supports mapper0.
func (c *Cartridge) readFromCPU(address uint16) (byte, error) {
	if 0x8000 <= address {
		// CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
		mod := uint16(len(c.prgROM))
		return c.prgROM[(address-0x8000)%mod], nil
	}
	// CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
	return 0, fmt.Errorf("Reading PRGRAM not implemented. address: 0x%04x", address)
}

func (c *Cartridge) writeFromCPU(address uint16, data byte) error {
	if 0x8000 <= address {
		return fmt.Errorf("Writing data to PrgROM not allowed: address=0x%04x, data=0x%02x", address, data)
	}
	// CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
	return fmt.Errorf("Writing data to PRGRAM not implemented. address: 0x%04x, data: 0x%02x", address, data)
}

func (c *Cartridge) readFromPPU(address uint16) (byte, error) {
	return c.chrROM[address], nil
}

func (c *Cartridge) writeFromPPU(address uint16, data byte) error {
	return fmt.Errorf("Writing data to pattern tables not allowed, address=0x%04x, data=0x%02x", address, data)
}
