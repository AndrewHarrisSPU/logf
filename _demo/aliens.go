package main

import (
	"strings"
	"time"

	"github.com/AndrewHarrisSPU/logf"
)

var space string = "     "
var beams string = " -~- "
var width int = len(beams) * 10

func main() {
	tty := logf.New().
		Layout("label", "message").
		Level(logf.INFO+1).
		Spin(logf.INFO, 1).
		TTY()

	log := tty.Logger().With("Scully", "👩‍🦰")
	ufo := log.Label("🛸").Level(logf.INFO)
	mulder := log.Label("👦🏻").Level(logf.INFO + 1)

	tick := 0.0
	step := 15.0
	for i := 0; i < width; i++ {
		lpad := strings.Repeat(beams, 10)[i:]
		rpad := strings.Repeat(space, 10)[width-i:]

		<-time.NewTimer(30 * time.Millisecond).C
		ufo.Msg("{}{Scully}{}", lpad, rpad)

		progress := (100 * float64(i)) / float64(width)
		if progress-tick > step {
			tick += step
			mulder.Msg("oh no! {Scully} is {}% abducted!", tick)
		}

	}

	log.Level(logf.INFO + 1).Msg("{Scully} was abducted")
	tty.Write(nil)
}
