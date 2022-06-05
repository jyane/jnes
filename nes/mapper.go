package nes

type Mapper interface {
	ReadFromCPU(uint16) (byte, error)
	WriteFromCPU(uint16, byte) error
	ReadFromPPU(uint16) (byte, error)
	WriteFromPPU(uint16, byte) error
}

func NewMapper(number byte, prgROM []byte, chrROM []byte) Mapper {
	switch number {
	case 0:
		return &mapper0{prgROM, chrROM}
	case 2:
		return NewMapper2(prgROM)
	}
	return nil
}
