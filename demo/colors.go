package main

import (
	"errors"

	"github.com/AndrewHarrisSPU/logf"
)

func arrow(b *logf.Buffer, level logf.Level) {
	b.WriteString("⇶⇶⇶▶")
}

func main() {
	log := logf.New().
		Level(logf.EncodeFunc[logf.Level](arrow)).
		LevelColors("blink italic cyan", "bright green", "bright yellow", "blink italic underline bright black bg red").
		Message("dim").
		Logger()

	log.Level(-4).Msg("here!")
	log.Level(0).Msg("here!")
	log.Level(4).Msg("here!")
	log.Level(8).Err("here!", errors.New("BLINK"))
}
