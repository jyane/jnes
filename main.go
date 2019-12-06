package main

import (
	"fmt"
	"io/ioutil"
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
	buffer, err := readFile("./sample.nes")
	if err != nil {
		panic(err)
	}
	cartridge, err := NewCartridge(buffer)
	if err != nil {
		panic(err)
	}
	wram := NewRAM()
	NewCPU(wram)
	fmt.Println(cartridge.ReadPRGROM())
}
