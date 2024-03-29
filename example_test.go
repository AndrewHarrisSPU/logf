package logf_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/AndrewHarrisSPU/logf"
)

func ExampleEncoder() {
	noTime := func(buf *logf.Buffer, t time.Time) {
		buf.WriteString("???")
	}

	log := logf.New().
		ForceTTY(true).
		AddSource(true).
		ShowColor(false).
		ShowLevel(logf.LevelBar).
		ShowSource("", logf.SourceShort).
		ShowTime("", logf.EncodeFunc(noTime)).
		Logger()

	log.Info("...")

	// Output:
	// ▏ ??? ...
	//	example_test.go:25
}

type mapWithLogValueMethod map[string]any

func (mv mapWithLogValueMethod) LogValue() logf.Value {
	var as []logf.Attr
	for k, v := range mv {
		as = append(as, logf.KV(k, v))
	}

	return logf.GroupValue(as...)
}

func Example_basic() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	// Like slog
	log.Info("Hello, Roswell")

	// Some interpolation
	log = log.With("place", "Roswell")
	log.Infof("Hello, {place}")

	// Errors
	ufo := errors.New("🛸 spotted")

	// Like slog
	log.Error("", ufo)

	// Logging with errors and interpolation
	log.Errorf("{place}", ufo)

	// Using a logger to wrap an error
	err := log.WrapErr("{place}", ufo)
	log.Error("", err)

	// Output:
	// Hello, Roswell
	// Hello, Roswell
	// 🛸 spotted
	// Roswell: 🛸 spotted
	// Roswell: 🛸 spotted
}

func ExampleFmt() {
	// (KV is equivalent to slog.Any)
	flavor := logf.KV("flavor", "coconut")

	// logf.Fmt works with slog data
	msg := logf.Fmt("{flavor} pie", flavor)
	fmt.Println(msg)

	// Output:
	// coconut pie
}

// Formatting accepts [fmt] package verbs.
// Verbs appear after the ':' in `{key:verb}` strings.
func Example_formattingVerbs() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	log.Infof("{left-pad:%010d}", "left-pad", 1)
	log.Infof("pi is about {pi:%6.5f}", "pi", 355.0/113)

	// Output:
	// 0000000001
	// pi is about 3.14159
}

func Example_interpolationArguments() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	// Unkeyed `{}` symbols parse key/value pairs in the logging call:
	log.Infof("The {} {} {} ...",
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
	log.Infof("The {speed} {color} {animal} ...", "speed", "rocketing")

	// Output:
	// The quick brown fox ...
	// The rocketing brindle Boston Terrier ...
}

func Example_interpolationArgumentsMixed() {
	log := logf.New().
		ShowColor(false).
		ShowLayout("message", "\t", "attrs").
		ForceTTY(true).
		Logger()

	// The unkeyed interpolation token `{}` consumes the first agument pair ("pi", 3.14)
	// "greek" and "π" parse to a second  attribute, which is interpolated by key
	log.Infof("{greek}: {}", "pi", 3.14, "greek", "π")

	// Output:
	// π: 3.14	pi:3.14 greek:π
}

// Interpolation can require escaping of '{', '}', and ':'
func Example_interpolationEscapes() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	// A Salvador Dali mustache emoji needs no escaping - there is no interpolation
	log.Infof(`:-}`)

	// Also surreal: escaping into JSON
	log.Infof(`\{"{key}":"{value}"\}`, "key", "color", "value", "mauve")

	// A single colon is parsed as a separator between an interpolation key and a formatting verb
	log.Infof(`{:}`, "", "plaintext")

	// Escaping a common lisp keyword symbol
	log.Infof(`{\:keyword}`, ":keyword", "lisp")

	// \Slashes, "quotes", and `backticks`
	log.Infof("{\\\\}", `\`, `slash`)
	log.Infof(`{\\}`, `\`, `slash`)

	// Output:
	// :-}
	// {"color":"mauve"}
	// plaintext
	// lisp
	// slash
	// slash
}

// Interpolation of time values accepts some additional verbs.
// See [Config.TimeFormat] for formatting of [TTY] time fields.
func Example_interpolationTimeVerbs() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	log.Infof("time interpolation formatting:")
	log.Infof("no verb {}", time.Time{})
	log.Infof("RFC3339 {:RFC3339}", time.Time{})
	log.Infof("kitchen {:kitchen}", time.Time{})
	log.Infof("timestamp {:stamp}", time.Time{})
	log.Infof("epoch {:epoch}", time.Time{})

	// custom formatting uses strings like time.ShowLayout, using a semicolon rather than ':'
	log.Infof("custom {:15;03;04}", time.Time{})

	log.Infof("duration interpolation formatting:")
	d := time.Unix(1000, 0).Sub(time.Unix(1, 0))
	log.Infof("no verb {}", d)
	log.Infof("epoch {:epoch}", d)

	// Output:
	// time interpolation formatting:
	// no verb 0001-01-01T00:00:00.000Z
	// RFC3339 0001-01-01T00:00:00Z
	// kitchen 12:00AM
	// timestamp Jan  1 00:00:00
	// epoch -62135596800
	// custom 00:12:00
	// duration interpolation formatting:
	// no verb 16m39s
	// epoch 999000000000
}

// Interpolation of [slog.LogValuer]s is powerful, but can be subtle.
func Example_interpolationLogValuer() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	vmap := mapWithLogValueMethod{
		"first":  1,
		"second": [2]struct{}{},
		"third":  "Hello, world",
	}

	log.Infof("{vmap.first}", "vmap", vmap)
	log.Infof("{vmap.second}", "vmap", vmap)

	// SUBTLE:
	// this won't work, becuase vmap is not associated with "vmap"
	log.Infof("{vmap.third}", vmap)

	// Output:
	// 1
	// [{} {}]
	// !missing-match
}

// Building attributes is essential to capturing structure.
// For convenience, logf aliases or reimplements some [slog.Attr]-forming functions.
func Example_structure() {
	log := logf.New().
		ShowLayout("message", "\t", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	// logf.Attr <=> slog.Attr
	// (likewise for logf.Value)
	var files logf.Attr

	// KV <=> slog.Any
	files = logf.KV("files", "X")

	// Attrs builds a slice of attrs, munging arguments
	mulder := []any{
		files,
		"title", "Special Agent",
		"name", "Fox Mulder",
	}

	// Group <=> slog.Group
	agent := logf.Group("agent", mulder...)

	log = log.With(agent)
	log.Info("The Truth Is Out There")

	// Output:
	// The Truth Is Out There	agent:{files:X title:Special Agent name:Fox Mulder}
}

// Logging, wrapping, and bubbling errors are all possible
func ExampleWrapErr() {
	log := logf.New().
		ShowLayout("message", "\t", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	log = log.WithGroup("emails").With("user", "Strong Bad", "id", "12345")
	err := errors.New("the system is down")

	// i. logging the error
	log.Error("", err)

	// with added context
	log.Errorf("{emails.user}", err)

	// ii. wrapping the error, with no msg -> the error
	err2 := logf.WrapErr("", err)
	fmt.Println(err2.Error())

	// iii. wrapping the error, with interpolated context
	err3 := log.WrapErr("{emails.user}", err)
	fmt.Println(err3.Error())

	// (equivalently)
	err3 = logf.WrapErr("{emails.user}", err, log)
	fmt.Println(err3.Error())

	// Output:
	// the system is down	emails:{user:Strong Bad id:12345 err:the system is down}
	// Strong Bad: the system is down	emails:{user:Strong Bad id:12345 err:the system is down}
	// the system is down
	// Strong Bad: the system is down
	// Strong Bad: the system is down
}

func ExampleConfig_ShowLayout() {
	log := logf.New().
		ShowLayout("level", "attrs", "message", "tag", "\n", "source").
		ShowLevel(logf.LevelBar).
		ShowSource("", logf.SourcePkg).
		ShowColor(false).
		AddSource(true).
		ForceTTY(true).
		Logger().
		With("#", "rightTag")

	log.Info("Hello!", "leftAttr", "here")

	// Output:
	// ▏ leftAttr:here Hello! rightTag
	// 	logf
}

func ExampleLogger_WrapErr() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	log = log.With("flavor", "coconut")

	errInvalidPizza := errors.New("invalid pizza")
	err := log.WrapErr("{flavor}", errInvalidPizza)
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
		ShowColor(false).
		ForceTTY(true).
		Printer()

	log = log.With("aliens", "Kang and Kodos, the Conquerors of Rigel VII")

	log.Info("Hello, world")
	log.Info(logf.Fmt("{}", "", "Hello, world"))
	log.Info(logf.Fmt("With menace, {aliens} uttered \"{}\"", "", "Hello, world", log))

	// log.Info("Hello, world")
	// log.Info("{}", "", "Hello, world")
	// log.Info("With menace, {aliens} uttered \"{}\"", "", "Hello, world")

	// Output:
	// Hello, world
	// Hello, world
	// With menace, Kang and Kodos, the Conquerors of Rigel VII uttered "Hello, world"
}

func ExampleLogger_Errorf() {
	log := logf.New().
		ShowColor(false).
		ForceTTY(true).
		Printer()

	errNegative := errors.New("negative number")

	log.Error("", errNegative)

	log = log.With("component", "math")

	log.Errorf("{component}: square root of {}", errNegative, "n", -1)

	err := log.WrapErr("{component}: square root of {}", errNegative, "n", -1)
	log.Error("", err)

	// Output:
	// negative number
	// math: square root of -1: negative number
	// math: square root of -1: negative number
}

func ExampleLogger_WithGroup() {
	log := logf.New().
		ShowLayout("message", "\t", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	log = log.
		WithGroup("outer").With("x", 1).
		WithGroup("inner").With("x", 2).
		WithGroup("local")

	log.Infof("outer {outer.x}", "x", 3)
	log.Infof("inner {outer.inner.x}", "x", 3)
	log.Infof("local {outer.inner.local.x}", "x", 3)
	log.Infof("local {x}", "x", 3)

	// Output:
	// outer 1	outer:{x:1 inner:{x:2 local:{x:3}}}
	// inner 2	outer:{x:1 inner:{x:2 local:{x:3}}}
	// local 3	outer:{x:1 inner:{x:2 local:{x:3}}}
	// local 3	outer:{x:1 inner:{x:2 local:{x:3}}}
}

func ExampleLogger_With() {
	log := logf.New().
		ShowLayout("message", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	log = log.With("species", "gopher")
	log.Info("")

	// Output:
	// species:gopher
}

func ExampleLogger_tag() {
	log := logf.New().
		ShowLayout("message", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Printer()

	l1 := log.With("#", "Log-9000")
	l2 := l1.With("#", "Log-9001")

	l1.Info("Hi!")
	l2.Info("Plus one!")

	// Output:
	// Log-9000 Hi!
	// Log-9001 Plus one!
}

func ExampleJSONValue() {
	log := logf.New().
		ShowLayout("message", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	object :=
		`{
	"vegetables":
		[
			"tomato",
			"pepper",
			"green onion"
		],
	"protein":"tofu"
}`

	v, _ := logf.JSONValue(object)
	recipe := logf.KV("recipe", v)

	log.Info("", recipe)
	log.Info(logf.Fmt("{recipe.vegetables.1}", recipe))

	// Output:
	// recipe:{vegetables:{0:tomato 1:pepper 2:green onion} protein:tofu}
	// pepper
}
