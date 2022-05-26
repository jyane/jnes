package integration

import (
	"image/png"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jyane/jnes/nes"
)

func TestHelloWorld(t *testing.T) {
	f, _ := os.Open("sample1.nes")
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	console := nes.NewConsole(b, false /* debug */)
	for {
		console.Step()
		got, ok := console.Frame()
		if ok {
			r, _ := os.Open("helloworld.png")
			defer r.Close()
			want, _ := png.Decode(r)
			for y := 0; y < got.Rect.Max.Y; y++ {
				for x := 0; x < got.Rect.Max.X; x++ {
					if got.At(x, y) != want.At(x, y) {
						t.Errorf("Got a rendered color at (%d, %d) = %v, want %v", x, y, got.At(x, y), want.At(x, y))
					}
				}
			}
			return
		}
	}
}
