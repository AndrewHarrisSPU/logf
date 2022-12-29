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

	log.Info("message", "key", "value")
}

func spacing() {
	log := logf.New().
		AddSource(true).
		Time("dim", logf.TimeShort).
		Layout("level", "time", "message", "\t", "attrs", "\t", "source").
		Source("dim", logf.SourceShort).
		Logger().
		With("#", "spacing")

	log.Info("_", "len", 1)
	log.Info("__", "len", 2)
	log.Info("___", "len", 3)
	log.Info("____", "len", 4)
	log.Info("_____", "len", 5)
	log.Info("______", "len", 6)
	log.Info("_______", "len", 7)
	log.Info("________", "len", 8)
	log.Info("_________", "len", 9)
	log.Info("__________", "len", 10)
}

func level() {
	logf.StdRef.Set(logf.DEBUG)
	defer logf.StdRef.Set(logf.INFO)

	log := logf.New().
		LevelColors("dim", "bright green", "bright yellow", "bright red").
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Log(logf.DEBUG-4, "_")
	log.Debug("_")
	log.Log(logf.INFO-1, "_")
	log.Info("_")
	log.Log(logf.INFO+1, "_")
	log.Warn("_")
	log.Error("_", nil)
	log.Log(logf.ERROR+4, "_")
}

func levelText() {
	logf.StdRef.Set(logf.DEBUG)
	defer logf.StdRef.Set(logf.INFO)

	log := logf.New().
		Level(logf.LevelText).
		Layout("time", "level", "tags", "message", "attrs").
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Log(logf.DEBUG-4, "_")
	log.Debug("_")
	log.Log(logf.INFO-1, "_")
	log.Info("_")
	log.Log(logf.INFO+1, "_")
	log.Warn("_")
	log.Error("_", nil)
	log.Log(logf.ERROR+4, "_")
}

func levelMono() {
	logf.StdRef.Set(logf.DEBUG)
	defer logf.StdRef.Set(logf.INFO)

	log := logf.New().
		Colors(false).
		Logger().
		With("key", "value").
		With("key2", "value2")

	log.Log(logf.DEBUG-4, "_")
	log.Debug("_")
	log.Log(logf.INFO-1, "_")
	log.Info("_")
	log.Log(logf.INFO+1, "_")
	log.Warn("_")
	log.Error("_", nil)
	log.Log(logf.ERROR+4, "_")
}

func proofs(log *logf.Logger) {
	println()
	log.Info("lorem ipsum...")
	log.With("key", "value")
	log.WithGroup("group")
	log.With("key", "value")
	log.Error("message text", errors.New("error text"))
	log.Info("{}", logf.Group("group2", logf.Attrs("key", "value")...))
}

func styles() {
	log := logf.New().
		Layout("level", "time", "message", "tags", "\n", "attrs").
		Level(logf.LevelBar).
		Time("dim", logf.TimeShort).
		Tag("method", "bright yellow").
		Tag("span_id", "yellow").
		Logger()

	log = log.WithGroup("http")
	log = log.With("method", "GET", "uuid", 1)
	log = log.With("#", "styles")

	log.Info("a request")

	baggage := logf.Group("otel", logf.Attrs(
		logf.Group("span", logf.Attrs(
			"trace_id", "0x5b8aa5a2d2c872e8321cf37308d69df2",
			"span_id", "0x5fb397be34d26b51",
		)...),
		"parent_id", nil,
	)...)

	log.Infof("request #{http.uuid}", baggage)
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
