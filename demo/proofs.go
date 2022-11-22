package main

import (
	"errors"

	"github.com/AndrewHarrisSPU/logf"
)

func main() {
	levels := logf.New().
		Ref(logf.DEBUG-4).
		Layout("time", "level", "message", "attrs" ).
		Logger()

	levels.Level(logf.DEBUG-4).Msg("_")
	levels.Level(logf.DEBUG).Msg("_")
	levels.Level(logf.INFO-1).Msg("_")
	levels.Level(logf.INFO).Msg("_")
	levels.Level(logf.INFO+1).Msg("_")
	levels.Level(logf.WARN).Msg("_")
	levels.Level(logf.ERROR).Msg("_")
	levels.Level(logf.ERROR+4).Msg("_")

	log := logf.New().
		AddSource(true).
		Logger()

	log.With("key", "value")
	log.Label("label")
	log.Err("msg", errors.New("err"))

	print := logf.New().
		Printer()

	print.Msg("{}", "lorem ipsum...")
}
