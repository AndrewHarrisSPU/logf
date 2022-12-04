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
		TTY().
		With("Scully", "👩‍🦰")

	log := tty.Logger().Level(logf.INFO + 2)
	ufo := log.Label("🛸").Level(logf.INFO)
	mulder := log.Label("Mulder 👦🏻").Level(logf.INFO + 1)

	for i := 0; i < width; i++ {
		lpad := strings.Repeat(beams, 10)[i:]
		rpad := strings.Repeat(space, 10)[width-i:]

		<-time.NewTimer(25 * time.Millisecond).C

		if (i % 10) == 0 {
			progress := (100 * float64(i)) / float64(width)
			mulder.Msg("oh no! {Scully} is {:%2.2f}% abducted!", progress)
		}
		ufo.Msg("{}{Scully}{}", lpad, rpad)
	}

	log.Msg("{Scully} was abducted")
	tty.Close()
}
