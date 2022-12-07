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

	log.Debug("here!")
	log.Info("here!")
	log.Warn("here!")
	log.Error("here!", errors.New("BLINK"))
}
