package logf_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

func exampleLogger() logf.Logger {
	return logf.New(logf.Using.Minimal(false, false))
}

func Example() {
	log := exampleLogger()
	log.Msg("Hello, world!")

	log = log.With("name", "gophers")
	log.Msg("Hello, {name}")

	err := errors.New("no connection")
	log.Err("Couldn't greet {name}", err)

	// Output:
	// Hello, world!
	// Hello, gophers
	// Couldn't greet gophers: no connection
}

func ExampleLogger_Fmt() {
	log := logf.New()
	log = log.With("flavor", "coconut")

	msg, err := log.Fmt("{flavor} pie", nil)
	fmt.Println(msg)
	fmt.Println(err)

	errInvalidPizza := errors.New("invalid pizza")
	msg, err = log.Fmt("{flavor}", errInvalidPizza)
	fmt.Println("message:", msg)
	fmt.Println("error:", err)

	if errors.Is(err, errInvalidPizza) {
		fmt.Println("(matched invalid pizza error)")
	}

	// Output:
	// coconut pie
	// <nil>
	// message: coconut: invalid pizza
	// error: coconut: invalid pizza
	// (matched invalid pizza error)
}

func exampleWriter() (*strings.Builder, func()) {
	var b strings.Builder
	trim := func() {
		fmt.Print(b.String()[46:])
		b.Reset()
	}
	return &b, trim
}

// In package [slog], WithScope is motivated by a need to avoid key collisions.
// This is somewhat at odds with string interpolation, if it can't anticipate
// what scopes need to be included in {scoped.keys}.
//
// Fortunately, a `logf.Handler` does not see any `Attr`s or scope set by the `slog.Handler` it wraps.
// There's no need account for these scopes in interpolation symbols.
func ExampleLogger_WithScope() {
	w, trim := exampleWriter()

	// first a slog.Handler with "outer" scoped x: 1
	sl := slog.New(slog.NewTextHandler(w))
	sl = sl.WithScope("outer").With("x", 1)

	// next, a logf.Logger with "inner" scoped x: 2
	log := logf.New(logf.Using.Handler(sl.Handler()))
	log = log.WithScope("inner").With("x", 2)

	// outer scope is not visible:
	log.Msg("{outer.x}")
	trim()

	// inner scope is not nested:
	log.Msg("{outer.inner.x}")
	trim()

	// inner scope is directly visible:
	log.Msg("{inner.x}")
	trim()

	// setting a further scope:
	log = log.WithScope("local").With("x", 3)

	// inner scope:
	log.Msg("{inner.x}")
	trim()

	// local scope:
	log.Msg("{inner.local.x}")
	trim()

	// Output:
	// msg=!missing-attr outer·x=1 outer·inner·x=2
	// msg=!missing-attr outer·x=1 outer·inner·x=2
	// msg=2 outer·x=1 outer·inner·x=2
	// msg=2 outer·x=1 outer·inner·x=2 outer·inner·local·x=3
	// msg=3 outer·x=1 outer·inner·x=2 outer·inner·local·x=3
}
