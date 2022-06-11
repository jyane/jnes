package nes

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

var (
	pcRe  = regexp.MustCompile("^[A-Z0-9]{4}")
	aRe   = regexp.MustCompile("A:([A-Z0-9]*)")
	xRe   = regexp.MustCompile("X:([A-Z0-9]*)")
	yRe   = regexp.MustCompile("Y:([A-Z0-9]*)")
	pRe   = regexp.MustCompile("P:([A-Z0-9]*)")
	spRe  = regexp.MustCompile("SP:([A-Z0-9]*)")
	cycRe = regexp.MustCompile("CYC:(\\d*)")
)

func newTestCPU() *CPU {
	f, _ := os.Open("../testdata/other/nestest.nes")
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	cartridge, _ := NewCartridge(b)
	controller := NewController()
	ppuBus := NewPPUBus(NewRAM(), cartridge)
	ppu := NewPPU(ppuBus)
	apu := NewAPU()
	cpuBus := NewCPUBus(NewRAM(), ppu, apu, cartridge, controller)
	cpu := NewCPU(cpuBus)
	cpu.pc = 0xC000
	cpu.s = 0xFD
	cpu.p.decodeFrom(0x24)
	return cpu
}

func TestCPU(t *testing.T) {
	var wantCycle int
	var wantPC uint16
	var wantA, wantX, wantY, wantP, wantSP byte
	cycles := 7
	before := "initial state"
	in, _ := os.Open("../testdata/other/nestest.log")
	scanner := bufio.NewScanner(in)
	cpu := newTestCPU()
	for scanner.Scan() {
		t.Log(before)
		line := scanner.Text()
		fmt.Sscanf(pcRe.FindString(line), "%x", &wantPC)
		fmt.Sscanf(aRe.FindStringSubmatch(line)[1], "%x", &wantA)
		fmt.Sscanf(xRe.FindStringSubmatch(line)[1], "%x", &wantX)
		fmt.Sscanf(yRe.FindStringSubmatch(line)[1], "%x", &wantY)
		fmt.Sscanf(pRe.FindStringSubmatch(line)[1], "%x", &wantP)
		fmt.Sscanf(spRe.FindStringSubmatch(line)[1], "%x", &wantSP)
		fmt.Sscanf(cycRe.FindStringSubmatch(line)[1], "%d", &wantCycle)
		if cpu.pc != wantPC {
			t.Fatalf("cpu.pc: got=0x%04x, want=0x%04x", cpu.pc, wantPC)
		}
		if cpu.a != wantA {
			t.Fatalf("cpu.a: got=0x%02x, want=0x%02x", cpu.a, wantA)
		}
		if cpu.x != wantX {
			t.Fatalf("cpu.x: got=0x%02x, want=0x%02x", cpu.x, wantX)
		}
		if cpu.y != wantY {
			t.Fatalf("cpu.y: got=0x%02x, want=0x%02x", cpu.y, wantY)
		}
		if cpu.p.encode() != wantP {
			wantStatus := status{}
			wantStatus.decodeFrom(wantP)
			t.Fatalf("cpu.p: got=(%02x) %+v, want=(%02x) %+v", cpu.p.encode(), cpu.p, wantP, wantStatus)
		}
		if cpu.s != wantSP {
			t.Fatalf("cpu.sp: got=0x%02x, want=0x%02x", cpu.s, wantSP)
		}
		if cycles != wantCycle {
			t.Fatalf("cycle: got=%d, want=%d", cycles, wantCycle)
		}
		c, _ := cpu.Step()
		cycles += c
		before = line
	}
}
