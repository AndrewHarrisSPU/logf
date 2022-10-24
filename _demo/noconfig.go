package main

import (
	"github.com/AndrewHarrisSPU/logf"
)

func main() {
	log := logf.New()
	log.Msg("Hello, world")
	log = log.With("name", "none")
	log = log.Label("label")
	log.Msg("Knock knock")
}
