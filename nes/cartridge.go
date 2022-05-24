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
	return c, nil
}

func (c *Cartridge) getTableMirrorMode() tableMirrorMode {
	if c.flags6&1 == 1 {
		return vertical
	} else {
		return horizontal
	}
}
