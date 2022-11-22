package logf_test

import (
	"errors"
	"fmt"

	"github.com/AndrewHarrisSPU/logf"
)

// Interpolation can require escaping of '{', '}', and ':'
func Example_interpolationEscapes() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	// A Salvador Dali mustache emoji needs no escaping - there is no interpolation
	log.Msg(`:-}`)

	// Also surreal: escaping into JSON
	log.Msg(`\{"{key}":"{value}"\}`, "key", "color", "value", "mauve")

	// A single colon is parsed as a separator between an interpolation key and a formatting verb
	log.Msg(`{:}`, "plaintext")

	// Escaping a common lisp keyword symbol
	log.Msg(`{\:keyword}`, ":keyword", "lisp")

	// \Slashes, "quotes", and `backticks`
	log.Msg("{\\\\}", `\`, `slash`)
	log.Msg(`{\\}`, `\`, `slash`)

	// Output:
	// :-}
	// {"color":"mauve"}
	// plaintext
	// lisp
	// slash
	// slash
}

func Example_formattingVerbs() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log.Msg("{left-pad:%010d}", "left-pad", 1)
	log.Msg("pi is about {pi:%6.5f}", "pi", 355.0/113)

	// Output:
	// 0000000001
	// pi is about 3.14159
}

func Example_interpolationArguments() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	// Unkeyed `{}` symbols draw one argument each from a logging call:
	log.Msg("The {} {} {} ...",
		"quick",
		"brown",
		"fox",
	)

	// Keyed `{key}` symbols interpolate on attribute keys
	// These attributes may exist in logger structure, or they may be provided in a logging call.
	log.With(
		"color", "brindle",
		"animal", "Boston Terrier",
	)
	log.Msg("The {speed} {color} {animal} ...", "speed", "rocketing")

	// Output:
	// The quick brown fox ...
	// The rocketing brindle Boston Terrier ...
}

func Example_interpolationArgumentsMixed() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	// Because only 3.14 is used for unkeyed interpolation,
	// "greek" and "π" parse to an attribute
	log.Msg("{greek}: {}", 3.14, "greek", "π")

	// Output:
	// π: 3.14  greek=π
}

// Building attributes is essential to capturing structure.
// Mostly to avoid needing to import slog, but also to offer a few tweaked behaviors, logf repackages Attr constructors.
func Example_structure() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	// KV <=> slog.Any
	files := logf.KV("files", "X")

	// Attrs builds a slice of attrs, munging arguments
	mulder := logf.Attrs(
		files,
		"title", "Special Agent",
		"name", "Fox Mulder",
	)

	// Group <=> slog.Group
	agent := logf.Group("agent", mulder...)

	log = log.With(agent)
	log.Msg("The Truth Is Out There")

	// A Logger is a LogValuer, and the value is a slog.Group
	print := logf.New().
		Colors(false).
		ForceTTY().
		Printer()
	print.Msg("{}", log)

	// the With method understands a LogValuer that resolves to a slog.Group
	print.With(log)
	print.Msg("{agent.name}")

	// Output:
	// The Truth Is Out There  agent=[files=X title=Special Agent name=Fox Mulder]
	// [agent=[files=X title=Special Agent name=Fox Mulder]]
	// Fox Mulder

}

// With a logf.Logger and interpolation, there are a variety of ways to handle an error
func Example_structureErrors() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	log.Label("emails").With("user", "Strong Bad", "id", "12345")
	err := errors.New("the system is down")

	// i. logging the error
	log.Err("", err)

	// ii. wrapping the error, with no msg -> add label
	err2 := log.Errf("", err)
	fmt.Println(err2.Error())

	// iii. wrapping the error, with interpolated context
	err3 := log.Errf("{user}", err)
	fmt.Println(err3.Error())

	// iv. wrapping the error, with all available structure
	//   - log's type is logf.Logger
	//   - a logf.Logger is also a slog.LogValuer
	//   - "{}" consumes log's LogValue
	err4 := log.Errf("{}", err, log)
	fmt.Println(err4.Error())

	// Output:
	// emails the system is down  user=Strong Bad id=12345
	// emails: the system is down
	// Strong Bad: the system is down
	// [user=Strong Bad id=12345]: the system is down
}

func ExampleConfig_Layout() {
	log := logf.New().
		Colors(false).
		Layout("attrs", "message").
		ForceTTY().
		Logger()

	log.Msg("Hello!", "left", "here")

	// Output:
	// left=here  Hello!
}

func ExampleLogger_Msgf() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("flavor", "coconut")

	msg := log.Msgf("{flavor} pie", nil)
	fmt.Println("msg:", msg)

	// Output:
	// msg: coconut pie
}

func ExampleLogger_Errf() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("flavor", "coconut")

	errInvalidPizza := errors.New("invalid pizza")
	err := log.Errf("{flavor}", errInvalidPizza)
	fmt.Println("err:", err)

	if errors.Is(err, errInvalidPizza) {
		fmt.Println("(matched invalid pizza error)")
	}

	// Output:
	// err: coconut: invalid pizza
	// (matched invalid pizza error)
}

func ExampleLogger_Msg() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("aliens", "Kang and Kodos, the Conquerors of Rigel VII")

	log.Msg("Hello, world")
	log.Msg("{}", "Hello, world")
	log.Msg("With menace, {aliens} uttered \"{}\"", "Hello, world")

	// Output:
	// Hello, world
	// Hello, world
	// With menace, Kang and Kodos, the Conquerors of Rigel VII uttered "Hello, world"
}

func ExampleLogger_Err() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	errNegative := errors.New("negative number")

	log.Err("", errNegative)

	log = log.With("component", "math")
	log.Err("{component}: square root of {}", errNegative, -1)

	// Output:
	// negative number
	// math: square root of -1: negative number
}

func ExampleLogger_Group() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	log.Group("outer").With("x", 1).
		Group("inner").With("x", 2).
		Group("local").
		Label("xs")

	log.Msg("outer {outer.x}", "x", 3)
	log.Msg("inner {outer.inner.x}", "x", 3)
	log.Msg("local {outer.inner.local.x}", "x", 3)

	// Output:
	// xs outer 1  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
	// xs inner 2  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
	// xs local 3  outer.x=1 outer.inner.x=2 outer.inner.local.x=3
}

func ExampleLogger_Level() {
	log := logf.New().
		Colors(false).
		Ref(logf.INFO). // the reference level (receiver type is *Config)
		ForceTTY().
		Printer().
		Level(logf.INFO) // the logger level (receiver type is Logger)

	// not visible, because logger level is less than reference level
	log.Level(logf.DEBUG).Msg("i'm hiding")

	// visible, because the receiver of the previous call was a new Logger created by log.Level
	log.Msg("back to INFO level")

	// not visible, because the Logger returned by log.Level is assigned to log
	log = log.Level(logf.DEBUG)
	log.Msg("now i'm invisible")

	// Output:
	// back to INFO level
}

func ExampleLogger_With() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	log.With("species", "gopher")
	log.Msg("")

	// Output:
	// species=gopher
}
