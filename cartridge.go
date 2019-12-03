package main

import (
	"errors"
	"fmt"
)

const (
	INES_HEADER_SIZE_BYTES      = 16
	MS_DOS_EOF             byte = 0x1A
)

type Cartridge struct {
	data []byte
}

func NewCartridge(buffer []byte) (*Cartridge, error) {
	if buffer[0] == byte('N') && buffer[1] == byte('E') && buffer[2] == byte('S') && buffer[3] == MS_DOS_EOF {
		return &Cartridge{buffer}, nil
	} else {
		return nil, errors.New("The buffer is not a valid iNES format.")
	}
}

func (c *Cartridge) ReadPRGROM() []byte {
	var v = c.data[4]
	fmt.Println(v)
	return []byte{}
}
