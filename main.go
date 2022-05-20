package main

import (
	"flag"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/golang/glog"

	"github.com/jyane/jnes/nes"
	"github.com/jyane/jnes/ui"
)

var (
	path       = flag.String("path", "./rom/sample1.nes", "path to NES ROM file")
	width      = flag.Int("width", 256*4, "widow width")
	height     = flag.Int("height", 240*4, "widow height")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

// readFile reads file as bytes
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

func init() {
	runtime.LockOSThread()
}

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create("cpu.pprof")
		if err != nil {
			glog.Fatal("Failed to create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			glog.Fatal("Failed to start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	buf, err := readFile(*path)
	if err != nil {
		glog.Fatalln("Failed to read: " + *path)
	}
	console := nes.NewConsole(buf)
	ui.Start(console, *width, *height)
}
