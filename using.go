package logf

import (
	"io"
	"os"

	"golang.org/x/exp/slog"
)

// USING

// Using is an aggregation of Logger and Handler configuration options.
// Each may be passed to [New]. For example:
//
//	New(Using.Text, Using.Stdout)
//
// creates a new Logger, using text encoding, and writing to standard output.
//
// Elements of Using are either option[T] or optionFunc[T].
// An option[T] is a function that sets a field in an (unexported) configuration struct.
// An optionFunc[T] is a function, taking one argument of type T, and returning an option[T].
var Using struct {
	// slog Handlers
	Text option[slog.TextHandler]
	JSON option[slog.JSONHandler]

	// any handler
	Handler optionFunc[slog.Handler]

	// minimal encoder
	// layout is time layout
	// elapsed hints to use elapsed time rather than clock time
	// export hints to use
	//
	// Minimal also observes io.Writer, slog.Leveler, and source options.
	Minimal func(elapsed, export bool) option[slog.Handler]

	// os pipes
	Stdout option[io.Writer]
	Stderr option[io.Writer]

	// any writer
	Writer optionFunc[io.Writer]

	// reference level
	Level optionFunc[slog.Leveler]

	// using source
	Source option[source]
}

func init() {
	Using.Stdout = usingWriter(os.Stdout)
	Using.Stderr = usingWriter(os.Stderr)
	Using.Writer = usingWriter
	Using.JSON = usingJSON
	Using.Text = usingText
	Using.Handler = usingHandler
	Using.Level = usingLevel
	Using.Source = usingSource
	Using.Minimal = usingMinimal // see minimal.go
}

func usingWriter(w io.Writer) option[io.Writer] {
	return func(cfg *config) {
		cfg.w = w
	}
}

func usingJSON(cfg *config) {
	cfg.h = slog.HandlerOptions{AddSource: cfg.addSource}.NewJSONHandler(cfg.w)
}

func usingText(cfg *config) {
	cfg.h = slog.HandlerOptions{AddSource: cfg.addSource}.NewTextHandler(cfg.w)
}

func usingHandler(h slog.Handler) option[slog.Handler] {
	return func(cfg *config) {
		cfg.h = h
	}
}

func usingLevel(ref slog.Leveler) option[slog.Leveler] {
	return func(cfg *config) {
		cfg.ref = ref
	}
}

func usingSource(cfg *config) {
	cfg.addSource = true
}

// OPTION / CFG

type (
	// Options may be passed around in other packages,
	// but must be created with package-level Using variables
	Option interface {
		__option__()
	}

	option[T any] func(*config)

	optionFunc[T any] func(T) option[T]

	// options set one or more of these fields
	config struct {
		w         io.Writer
		h         slog.Handler
		ref       slog.Leveler
		addSource bool
	}

	// stand-in types
	source struct{}
)

func (option[T]) __option__() {}
