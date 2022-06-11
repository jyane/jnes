package ui

import (
	"fmt"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100

type audio struct {
	stream  *portaudio.Stream
	channel chan float32
}

func newAudio() *audio {
	a := &audio{}
	a.channel = make(chan float32, sampleRate)
	return a
}

func (a *audio) start() error {
	portaudio.Initialize()
	cb := func(out []float32) {
		for i := range out {
			select {
			case x := <-a.channel:
				out[i] = x * 0.05
			default:
				out[i] = 0
			}
		}
	}
	stream, err := portaudio.OpenDefaultStream(0, 2, sampleRate, 0, cb)
	if err != nil {
		return fmt.Errorf("Failed to open the audio stream: %w", err)
	}
	a.stream = stream
	if err := stream.Start(); err != nil {
		return fmt.Errorf("Failed to start the audio stream: %w", err)
	}
	return nil
}

func (a *audio) terminate() {
	portaudio.Terminate()
	a.stream.Close()
}
