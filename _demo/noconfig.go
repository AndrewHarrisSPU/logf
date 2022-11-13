package main

import (
	"github.com/AndrewHarrisSPU/logf"
)

func main() {
	log := logf.StdTTY.Printer()
	log.Msg("Hello, world")
	log = log.With("name", "none")
	log = log.Label("label")
	log.Msg("Knock knock")
}

/*
	w  io.Writer
	mu *sync.Mutex

	// encoding
	layout     []ttyField
	spin       spinner
	timeFormat string
	start      time.Time

	elapsed  bool
	addLabel bool
	colors   bool
	addSource   bool
*/
