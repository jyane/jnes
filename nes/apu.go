package nes

import "math"

type APU struct {
	pulse1 pulse
	pulse2 pulse
	out    chan float32
	sample int
}

func NewAPU() *APU {
	return &APU{}
}

func (a *APU) Step() {
	sampleRate := 44100
	x := float32(math.Sin(2.0 * math.Pi * 440 * float64(a.sample) / float64(sampleRate)))
	select {
	case a.out <- x: // l
	default:
	}
	select {
	case a.out <- x: // r
	default:
	}
	a.sample++
	if a.sample >= sampleRate*10 {
		a.sample = 0
	}
}

func (a *APU) SetAudioOut(c chan float32) {
	a.out = c
}

func (a *APU) writeControl(data byte) {
}

// Pulse
type pulse struct {
}

func (p *pulse) writeControl(data byte) {
}

func (p *pulse) writeSweep(data byte) {
}

func (p *pulse) writeTimerLow(data byte) {
}

func (p *pulse) writeTimerHigh(data byte) {
}
