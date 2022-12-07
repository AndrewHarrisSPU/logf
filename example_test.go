package logf_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/AndrewHarrisSPU/logf"
)

type mapWithLogValueMethod map[string]any

func (mv mapWithLogValueMethod) LogValue() logf.Value {
	var as []logf.Attr
	for k, v := range mv {
		as = append(as, logf.KV(k, v))
	}

	return logf.GroupValue(as)
}

func Example_interpolationLogValuer() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	vmap := mapWithLogValueMethod{
		"first":  1,
		"second": [2]struct{}{},
		"third":  "Hello, world",
	}

	log.Info("{vmap.first}", "vmap", vmap.LogValue())
	log.Info("{vmap.second}", "vmap", vmap.LogValue())

	// SUBTLE:
	// this won't work, becuase vmap is not associated with "vmap"
	log.Info("{vmap.third}", vmap)

	// Output:
	// 1
	// [{} {}]
	// !missing-match
}

// Interpolation can require escaping of '{', '}', and ':'
func Example_interpolationEscapes() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	// A Salvador Dali mustache emoji needs no escaping - there is no interpolation
	log.Info(`:-}`)

	// Also surreal: escaping into JSON
	log.Info(`\{"{key}":"{value}"\}`, "key", "color", "value", "mauve")

	// A single colon is parsed as a separator between an interpolation key and a formatting verb
	log.Info(`{:}`, "", "plaintext")

	// Escaping a common lisp keyword symbol
	log.Info(`{\:keyword}`, ":keyword", "lisp")

	// \Slashes, "quotes", and `backticks`
	log.Info("{\\\\}", `\`, `slash`)
	log.Info(`{\\}`, `\`, `slash`)

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

	log.Info("{left-pad:%010d}", "left-pad", 1)
	log.Info("pi is about {pi:%6.5f}", "pi", 355.0/113)

	// Output:
	// 0000000001
	// pi is about 3.14159
}

func Example_interpolationArguments() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	// Unkeyed `{}` symbols parse key/value pairs in the logging call:
	log.Info("The {} {} {} ...",
		"speed", "quick",
		"color", "brown",
		"animal", "fox",
	)

	// Keyed `{key}` symbols interpolate on attribute keys
	// These attributes may exist in logger structure, or they may be provided in a logging call.
	log = log.With(
		"color", "brindle",
		"animal", "Boston Terrier",
	)
	log.Info("The {speed} {color} {animal} ...", "speed", "rocketing")

	// Output:
	// The quick brown fox ...
	// The rocketing brindle Boston Terrier ...
}

func Example_interpolationArgumentsMixed() {
	log := logf.New().
		Colors(false).
		Layout("message", "\t", "attrs").
		ForceTTY().
		Logger()

	// Because only 3.14 is used for unkeyed interpolation,
	// "greek" and "π" parse to an attribute
	log.Info("{greek}: {}", "pi", 3.14, "greek", "π")

	// Output:
	// π: 3.14   pi:3.14 greek:π
}

// Interpolation of time values in message strings.
// This is distinct from how [Config.TimeFormat], which affects [TTY] time fields.
func Example_inerpolationTimeVerbs() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log.Info("time interpolation formatting:")
	log.Info("no verb {}", time.Time{})
	log.Info("RFC3339 {:RFC3339}", time.Time{})
	log.Info("kitchen {:kitchen}", time.Time{})
	log.Info("timestamp {:stamp}", time.Time{})
	log.Info("epoch {:epoch}", time.Time{})

	// custom formatting uses strings like time.Layout, using a semicolon rather than ':'
	log.Info("custom {:15;03;04}", time.Time{})

	log.Info("duration interpolation formatting:")
	d := time.Unix(1000, 0).Sub(time.Unix(1, 0))
	log.Info("no verb {}", d)
	log.Info("epoch {:epoch}", d)

	// Output:
	// time interpolation formatting:
	// no verb 1754-08-30T22:43:41.128Z
	// RFC3339 1754-08-30T22:43:41Z
	// kitchen 10:43PM
	// timestamp Aug 30 22:43:41
	// epoch -6795364579
	// custom 22:10:43
	// duration interpolation formatting:
	// no verb 16m39s
	// epoch 999000000000
}

// Building attributes is essential to capturing structure.
// Mostly to avoid needing to import slog, but also to offer a few tweaked behaviors, logf repackages Attr constructors.
func Example_structure() {
	log := logf.New().
		Colors(false).
		Layout("message", "\t", "attrs").
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
	agent := logf.Group("agent", mulder)

	log = log.With(agent)
	log.Info("The Truth Is Out There")

	// A Logger is a LogValuer, and the value is a slog.Group
	print := logf.New().
		Colors(false).
		ForceTTY().
		Printer()
	print.Info("{}", log)

	// Output:
	// The Truth Is Out There   agent:{files:X title:Special Agent name:Fox Mulder}
	// [agent=[files=X title=Special Agent name=Fox Mulder]]
}

// With a logf.Logger and interpolation, there are a variety of ways to handle an error
func Example_structureErrors() {
	log := logf.New().
		Colors(false).
		Layout("message", "\t", "attrs").
		ForceTTY().
		Logger()

	log = log.Group("emails").With("user", "Strong Bad", "id", "12345")
	err := errors.New("the system is down")

	// i. logging the error
	log.Error("", err)

	// ii. wrapping the error, with no msg -> the error
	err2 := log.NewErr("", err)
	fmt.Println(err2.Error())

	// iii. wrapping the error, with interpolated context
	err3 := log.NewErr("{emails.user}", err)
	fmt.Println(err3.Error())

	// iv. wrapping the error, with all available structure
	//   - log's type is logf.Logger
	//   - a logf.Logger is also a slog.LogValuer
	//   - "{}" consumes log's LogValue
	err4 := log.NewErr("{}", err, "log", log)
	fmt.Println(err4.Error())

	// Output:
	// the system is down   emails:{user:Strong Bad id:12345 err:the system is down}
	// the system is down
	// Strong Bad: the system is down
	// [emails.user=Strong Bad emails.id=12345]: the system is down
}

func ExampleConfig_Layout() {
	log := logf.New().
		Colors(false).
		Layout("attrs", "message").
		ForceTTY().
		Logger()

	log.Info("Hello!", "left", "here")

	// Output:
	// left:here Hello!
}

func ExampleLogger_Fmt() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("flavor", "coconut")

	msg := log.Fmt("{flavor} pie", nil)
	fmt.Println("msg:", msg)

	// Output:
	// msg: coconut pie
}

func ExampleLogger_NewErr() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("flavor", "coconut")

	errInvalidPizza := errors.New("invalid pizza")
	err := log.NewErr("{flavor}", errInvalidPizza)
	fmt.Println("err:", err)

	if errors.Is(err, errInvalidPizza) {
		fmt.Println("(matched invalid pizza error)")
	}

	// Output:
	// err: coconut: invalid pizza
	// (matched invalid pizza error)
}

func ExampleLogger_Info() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("aliens", "Kang and Kodos, the Conquerors of Rigel VII")

	log.Info("Hello, world")
	log.Info("{}", "", "Hello, world")
	log.Info("With menace, {aliens} uttered \"{}\"", "", "Hello, world")

	// Output:
	// Hello, world
	// Hello, world
	// With menace, Kang and Kodos, the Conquerors of Rigel VII uttered "Hello, world"
}

func ExampleLogger_Error() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	errNegative := errors.New("negative number")

	log.Error("", errNegative)

	log = log.With("component", "math")
	err := log.NewErr("{component}: square root of {}", errNegative, -1)
	log.Error("", err)

	// Output:
	// negative number
	// math: square root of -1: negative number
}

func ExampleLogger_Group() {
	log := logf.New().
		Colors(false).
		Layout("message", "\t", "attrs").
		ForceTTY().
		Logger()

	log = log.Group("outer").With("x", 1).
		Group("inner").With("x", 2).
		Group("local")

	log.Info("outer {outer.x}", "x", 3)
	log.Info("inner {outer.inner.x}", "x", 3)
	log.Info("local {outer.inner.local.x}", "x", 3)

	// Output:
	// outer 1   outer:{x:1 inner: {x:2 x:3}}}
	// inner 2   outer:{x:1 inner: {x:2 x:3}}}
	// local 3   outer:{x:1 inner: {x:2 x:3}}}
}

func ExampleLogger_With() {
	log := logf.New().
		Colors(false).
		Layout("message", "attrs").
		ForceTTY().
		Logger()

	log = log.With("species", "gopher")
	log.Info("")

	// Output:
	// species:gopher
}

func ExampleLogger_Tag() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	l1 := log.Tag("Log-9000")
	l2 := l1.Tag("Log-9001")

	l1.Info("Hi!")
	l2.Info("Plus one!")

	// Output:
	// Log-9000 Hi!
	// Log-9001 Plus one!
}

func ExampleEncoder() {
	noTime := func(buf *logf.Buffer, t time.Time) {
		buf.WriteString("???")
	}

	log := logf.New().
		Colors(false).
		ForceTTY().
		Level(logf.LevelBar).
		Source("", logf.SourceShort).
		AddSource(true).
		Time("", logf.EncodeFunc(noTime)).
		Logger()

	log.Info("...")

	// Output:
	// ▕▎ ??? ...
	//    example_test.go:416
}
