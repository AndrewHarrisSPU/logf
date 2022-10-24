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

// OPTION

type (
	// Options may be passed around in other packages,
	// but must be created with package-level Using variables
	Option interface {
		__option__()
	}

	option[T any]     func(*config)
	optionFunc[T any] func(T) option[T]
)

func (option[T]) __option__() {}

// stand-in types
type source struct{}

// CONFIG

type config struct {
	w          io.Writer
	h          slog.Handler
	ref        slog.Leveler
	addSource  bool
	usePrinter bool
}

func makeConfig(options ...Option) (cfg config) {
	// These depend on other configurations,
	// so evaluation is delayed
	var oSlog Option
	var oHandler option[slog.Handler]

	// consume options
	for _, o := range options {
		switch o := o.(type) {
		case option[slog.TextHandler], option[slog.JSONHandler]:
			oSlog = o
		case option[slog.Handler]:
			oHandler = o
		case option[io.Writer]:
			o(&cfg)
		case option[slog.Leveler]:
			o(&cfg)
		case option[source]:
			o(&cfg)
		default:
			panic("unknown option type")
		}
	}

	// if no writer was set, and no handler defined
	if cfg.w == nil && oHandler == nil && oSlog == nil {
		cfg.usePrinter = true
		cfg.addSource = Print.Source
		cfg.ref = Print.Level
		return
	}

	// use a specified writer
	if cfg.w == nil {
		cfg.w = os.Stdout
	}

	if cfg.ref == nil {
		cfg.ref = INFO
	}

	// use a specified Handler
	if oHandler != nil {
		oHandler(&cfg)
		return
	}

	if oSlog == nil {
		oSlog = option[slog.TextHandler](usingText)
	}

	// otherwise, build a slog Handler
	scfg := slog.HandlerOptions{
		Level:     cfg.ref,
		AddSource: cfg.addSource,
	}

	switch oSlog.(type) {
	case option[slog.JSONHandler]:
		cfg.h = scfg.NewJSONHandler(cfg.w)
	case option[slog.TextHandler]:
		cfg.h = scfg.NewTextHandler(cfg.w)
	}

	return
}
