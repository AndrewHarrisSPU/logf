package main

import (
	"github.com/AndrewHarrisSPU/logf"
)

func main() {
	log := logf.New().
		Level(logf.DEBUG - 4).
		Logger()

	for i := logf.DEBUG - 4; i <= logf.ERROR+4; i++ {
		log.Level(i).Msg(".")
	}
}
