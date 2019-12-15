package main

import (
	"io/ioutil"
	"log"
	"os"
)

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func main() {
	var romPath = "./sample.nes"
	log.Println("ROM path = " + romPath)
	buffer, err := readFile(romPath)
	if err != nil {
		panic(err)
	}
	log.Printf("Rom size = %d bytes\n", len(buffer))
	cartridge := NewCartridge(buffer)
	var check = cartridge.IsValid()
	if !check {
		panic("The cartridge is not a valid INES format.")
	}
	log.Printf("Catridge(%s) is valid INES format\n", romPath)
	prgROM := cartridge.ReadPRGROM()
	log.Printf("Program ROM size = %d bytes\n", len(prgROM))
	chrROM := cartridge.ReadCHRROM()
	log.Printf("Character ROM size = %d bytes\n", len(chrROM))
	cpu := NewCPU(NewCPUBus(NewRAM(), cartridge.ReadPRGROM()))
	cpu.Step()
	NewPPU(NewRAM())
}
