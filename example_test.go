package logf_test

import (
	"errors"
	"fmt"

	"github.com/AndrewHarrisSPU/logf"
)

func Example() {
	log := logf.New().
		Colors(false).
		Printer()

	log.Msg("Hello, world!")

	log = log.With("name", "gophers")
	log.Msg("Hello, {name}!")

	err := errors.New("no connection")
	log.Err("Couldn't greet {name}", err)

	// Output:
	// Hello, world!
	// Hello, gophers!
	// Couldn't greet gophers: no connection
}

func Example_formatting() {
	log := logf.New().
		Colors(false).
		Printer()

	log.Msg("{:%010s}", "left-pad")
	log.Msg("pi is about {pi:%6.5f}", "pi", 355.0/113)

	// Output:
	// 00left-pad
	// pi is about 3.14159
}

func Example_interpolation() {
	print := logf.New().
		Colors(false).
		Printer()

	// Unkeyed `{}` symbols draw one argument, like `print`:
	print.Msg("The {} {} {} ...",
		"quick",
		"brown",
		"fox",
	)

	// Keyed `{key}` symbols interpolate structure associate with a Logger.
	print = print.With(
		"speed", "quick",
		"color", "brown",
		"animal", "fox",
	)
	print.Msg("The {speed} {color} {animal} ...")

	// Extra arguments are used as attribute key value pairs.
	log := logf.New().
		Layout("message", "attrs").
		Colors(false).
		Logger()

	// because only 3.14 is used for unkeyed interpolation,
	// "greek" and "π" form an attribute
	log.Msg("pi: {}", 3.14, "greek", "π")

	// Output:
	// The quick brown fox ...
	// The quick brown fox ...
	// pi: 3.14  greek=π
}

func Example_leveled() {
	log := logf.New().
		Level(logf.DEBUG).
		Layout("message", "attrs").
		Colors(false).
		Logger()

	i := -1
	log.Level(logf.INFO).Msg("", "count", i)

	var errNegCount = errors.New("negative counter")
	if i < 0 {
		log.Level(logf.WARN).Err("oops", errNegCount, "count", i)
	}

	// Output:
	// count=-1
	// oops: negative counter  count=-1
}

type agent struct {
	title string
	name  string
}

func (a agent) LogValue() logf.Value {
	return logf.Group("",
		logf.KV("title", a.title),
		logf.KV("name", a.name),
	).Value
}

func Example_structured() {
	/*
	   type agent struct {
	   	title string
	   	name  string
	   }

	   func (a agent) LogValue() logf.Value {
	   	return logf.Segment(
	   		"title", a.title,
	   		"name", a.name,
	   	)
	   }
	*/

	log := logf.New().
		Layout("label", "message", "attrs").
		Colors(false).
		Logger()

	log = log.With("files", "X")

	log.Msg("")

	log = log.With(agent{
		"Special Agent",
		"Fox Mulder",
	})

	log.Msg("The Truth Is Out There")

	// Output:
	// files=X
	// The Truth Is Out There  files=X title=Special Agent name=Fox Mulder
}

func ExampleConfig_Layout() {
	log := logf.New().
		Layout("attrs", "label", "message").
		Colors(false).
		Logger()

	log = log.Label("🙂")

	log.Msg("Hello!", "left", "here")

	// Output:
	// 🙂.left=here  🙂  Hello!
}

func ExampleLogger_Fmt() {
	log := logf.New().
		Colors(false).
		Printer()

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

func ExampleLogger_Label() {
	log := logf.New().
		Layout("label", "message", "attrs").
		Colors(false).
		Logger()

	log = log.Label("outer").With("x", 1)
	log = log.Label("inner").With("x", 2)
	log = log.Label("local")

	log.Msg("{outer.x}", "x", 3)
	log.Msg("{outer.inner.x}", "x", 3)
	log.Msg("{outer.inner.local.x}", "x", 3)

	// Output:
	// local  1  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
	// local  2  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
	// local  3  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
}
