package main

import (
	"errors"
	"time"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

func main() {
	logger()
	level()
	levelText()
	levelMono()
	spacing()
	reality()
	styles()
}

func logger() {
	log := logf.New().Logger()

	log.Msg("message", "key", "value")
}

func spacing() {
	log := logf.New().
		AddSource(true).
		Time("dim", logf.TimeShort).
		Layout("level", "time", "message", "\t", "attrs", "source").
		Source("dim", logf.SourceShort).
		Logger().
		Tag("spacing")

	log.Msg("_", "len", 1)
	log.Msg("__", "len", 2)
	log.Msg("___", "len", 3)
	log.Msg("____", "len", 4)
	log.Msg("_____", "len", 5)
	log.Msg("______", "len", 6)
	log.Msg("_______", "len", 7)
	log.Msg("________", "len", 8)
	log.Msg("_________", "len", 9)
	log.Msg("__________", "len", 10)
}

func level() {
	log := logf.New().
		Ref(logf.DEBUG-4).
		LevelColors("dim", "bright green", "bright yellow", "bright red").
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Level(logf.DEBUG - 4).Msg("_")
	log.Level(logf.DEBUG).Msg("_")
	log.Level(logf.INFO - 1).Msg("_")
	log.Level(logf.INFO).Msg("_")
	log.Level(logf.INFO + 1).Msg("_")
	log.Level(logf.WARN).Msg("_")
	log.Level(logf.ERROR).Msg("_")
	log.Level(logf.ERROR + 4).Msg("_")
}

func levelText() {
	log := logf.New().
		Ref(logf.DEBUG-4).
		Level(logf.LevelText).
		Layout("time", "level", "tags", "message", "attrs").
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Level(logf.DEBUG - 4).Msg("_")
	log.Level(logf.DEBUG).Msg("_")
	log.Level(logf.INFO - 1).Msg("_")
	log.Level(logf.INFO).Msg("_")
	log.Level(logf.INFO + 1).Msg("_")
	log.Level(logf.WARN).Msg("_")
	log.Level(logf.ERROR).Msg("_")
	log.Level(logf.ERROR + 4).Msg("_")
}

func levelMono() {
	log := logf.New().
		Ref(logf.DEBUG-4).
		Colors(false).
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Level(logf.DEBUG - 4).Msg("_")
	log.Level(logf.DEBUG).Msg("_")
	log.Level(logf.INFO - 1).Msg("_")
	log.Level(logf.INFO).Msg("_")
	log.Level(logf.INFO + 1).Msg("_")
	log.Level(logf.WARN).Msg("_")
	log.Level(logf.ERROR).Msg("_")
	log.Level(logf.ERROR + 4).Msg("_")
}

func proofs(log *logf.Logger) {
	println()
	log.Msg("lorem ipsum...")
	log.With("key", "value")
	log.Group("group")
	log.With("key", "value")
	log.Err("message text", errors.New("error text"))
	log.Msg("{}", logf.Group("group2", logf.Attrs("key", "value")))
}

func styles() {
	log := logf.New().
		Layout("level", "time", "label", "message", "tags", "\n", "attrs").
		Level(logf.LevelBar).
		Time("dim", logf.TimeShort).
		Tag("method", "bright yellow").
		Tag("span_id", "yellow").
		Logger()

	log.Group("http")

	log = log.With("method", "GET", "uuid", 1)
	log = log.Tag("styles")

	log.Msg("a request")

	baggage := logf.Group("otel", logf.Attrs(
		logf.Group("span", logf.Attrs(
			"trace_id", "0x5b8aa5a2d2c872e8321cf37308d69df2",
			"span_id", "0x5fb397be34d26b51",
		)),
		"parent_id", nil,
	))

	log.Msgf("request #{http.uuid}", baggage)
}

func reality() {
	h := logf.New().
		AddSource(true).
		Layout("level", "time", "message", "\n", "attrs", "\n", "source").
		Logger().
		Handler()

	log := slog.New(h)

	var args []any
	for _, a := range TestAttrs {
		args = append(args, a)
	}

	log.Info(TestMessage, args...)
}

const TestMessage = "Test logging, but use a somewhat realistic message length."

var TestAttrs = logf.Attrs(
	"time", time.Date(2022, time.May, 1, 0, 0, 0, 0, time.UTC),
	"string", "7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190",
	"int", 32768,
	"duration", 23*time.Second,
	"err", errors.New("fail"),
)
