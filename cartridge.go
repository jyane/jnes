package main

const (
	CHR_ROM_SIZE_UNIT      int  = 0x2000 // 8 bytes
	PRG_ROM_SIZE_UNIT      int  = 0x4000 // 16 bytes
	INES_HEADER_SIZE_BYTES int  = 16     // The valid INES header has 16 bytes
	MS_DOS_EOF             byte = 0x1A
)

type Cartridge struct {
	data []byte
}

func NewCartridge(buffer []byte) *Cartridge {
	return &Cartridge{buffer}
}

func (c *Cartridge) IsValid() bool {
	// TODO(jyane): Re-consider this size validation
	if len(c.data) >= INES_HEADER_SIZE_BYTES &&
		c.data[0] == byte('N') &&
		c.data[1] == byte('E') &&
		c.data[2] == byte('S') &&
		c.data[3] == MS_DOS_EOF {
		return true
	} else {
		return false
	}
}

func (c *Cartridge) ReadPRGROM() []byte {
	var l = INES_HEADER_SIZE_BYTES
	var r = INES_HEADER_SIZE_BYTES + int(c.data[4])*PRG_ROM_SIZE_UNIT
	return c.data[l:r]
}

func (c *Cartridge) ReadCHRROM() []byte {
	var l = INES_HEADER_SIZE_BYTES + int(c.data[4])*PRG_ROM_SIZE_UNIT
	var r = l + int(c.data[5])*CHR_ROM_SIZE_UNIT
	return c.data[l:r]
}
