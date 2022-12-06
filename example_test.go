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

func Example_interpolatonLogValuer() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	vmap := mapWithLogValueMethod{
		"first":  1,
		"second": [2]struct{}{},
		"third":  "Hello, world",
	}

	log.Msgf("{vmap.first}", "vmap", vmap)
	log.Msgf("{vmap.second}", "vmap", vmap)

	// VERY SUBTLE:
	// this won't work, becuase vmap is not associated with "vmap"
	log.Msgf("{vmap.third}", vmap)

	// this works
	log.Msgf("{third}", vmap)

	// Output:
	// 1
	// [{} {}]
	// !missing-attr
	// Hello, world
}

// Interpolation can require escaping of '{', '}', and ':'
func Example_interpolationEscapes() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	// A Salvador Dali mustache emoji needs no escaping - there is no interpolation
	log.Msgf(`:-}`)

	// Also surreal: escaping into JSON
	log.Msgf(`\{"{key}":"{value}"\}`, "key", "color", "value", "mauve")

	// A single colon is parsed as a separator between an interpolation key and a formatting verb
	log.Msgf(`{:}`, "", "plaintext")

	// Escaping a common lisp keyword symbol
	log.Msgf(`{\:keyword}`, ":keyword", "lisp")

	// \Slashes, "quotes", and `backticks`
	log.Msgf("{\\\\}", `\`, `slash`)
	log.Msgf(`{\\}`, `\`, `slash`)

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

	log.Msgf("{left-pad:%010d}", "left-pad", 1)
	log.Msgf("pi is about {pi:%6.5f}", "pi", 355.0/113)

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
	log.Msgf("The {} {} {} ...",
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
	log.Msgf("The {speed} {color} {animal} ...", "speed", "rocketing")

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
	log.Msgf("{greek}: {}", "pi", 3.14, "greek", "π")

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

	log.Msgf("time interpolation formatting:")
	log.Msgf("no verb {}", time.Time{})
	log.Msgf("RFC3339 {:RFC3339}", time.Time{})
	log.Msgf("kitchen {:kitchen}", time.Time{})
	log.Msgf("timestamp {:stamp}", time.Time{})
	log.Msgf("epoch {:epoch}", time.Time{})

	// custom formatting uses strings like time.Layout, using a semicolon rather than ':'
	log.Msgf("custom {:15;03;04}", time.Time{})

	log.Msgf("duration interpolation formatting:")
	d := time.Unix(1000, 0).Sub(time.Unix(1, 0))
	log.Msgf("no verb {}", d)
	log.Msgf("epoch {:epoch}", d)

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
	log.Msgf("The Truth Is Out There")

	// A Logger is a LogValuer, and the value is a slog.Group
	print := logf.New().
		Colors(false).
		ForceTTY().
		Printer()
	print.Msgf("{}", log)

	// Output:
	// The Truth Is Out There   agent:{files:X title:Special Agent name:Fox Mulder}
	// [files=X title=Special Agent name=Fox Mulder]
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
	log.Err("", err)

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
	// the system is down   emails:{user:Strong Bad id:12345}
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

	log.Msg("Hello!", "left", "here")

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

func ExampleLogger_Msgf() {
	log := logf.New().
		Colors(false).
		ForceTTY().
		Printer()

	log = log.With("aliens", "Kang and Kodos, the Conquerors of Rigel VII")

	log.Msg("Hello, world")
	log.Msgf("{}", "", "Hello, world")
	log.Msgf("With menace, {aliens} uttered \"{}\"", "", "Hello, world")

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
	err := log.NewErr("{component}: square root of {}", errNegative, -1)
	log.Err("", err)

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

	log.Msgf("outer {outer.x}", "x", 3)
	log.Msgf("inner {outer.inner.x}", "x", 3)
	log.Msgf("local {outer.inner.local.x}", "x", 3)

	// Output:
	// outer 1   outer:{x:1 inner: {x:2 x:3}}}
	// inner 2   outer:{x:1 inner: {x:2 x:3}}}
	// local 3   outer:{x:1 inner: {x:2 x:3}}}
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

	log = log.With("species", "gopher")
	log.Msg("")

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

	l1.Msg("Hi!")
	l2.Msg("Plus one!")

	// Output:
	// Log-9000 Hi!
	// Log-9001 Plus one!
}

func ExampleEncoder() {
	noTime := func(buf *logf.Buffer, t time.Time){
		buf.WriteString( "???" )
	}

	log := logf.New().
		Colors(false).
		ForceTTY().
		Level(logf.LevelBar).
		Source("", logf.SourceShort).
		AddSource(true).
		Time("", logf.EncodeFunc(noTime)).
		Logger()

	log.Msg("...")

	// Output:
	// ▕▎ ??? ...
	//    example_test.go:421
}