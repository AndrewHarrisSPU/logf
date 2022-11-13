package logf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

func substringTestLogger(t *testing.T) (Logger, func(string)) {
	var b bytes.Buffer

	wantFunc := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	log := New().
		Writer(&b).
		AddSource(true).
		JSON()

	return log, wantFunc
}

func TestDepth(t *testing.T) {
	log, want := substringTestLogger(t)
	fn := func() {
		log.Depth(1).Msg("where am I")
	}

	fn()
	want("logger_test.go:40")

	fn()
	want("logger_test.go:43")

	func() { fn() }()
	want("logger_test.go:46")

	func() {
		func() {
			log.Depth(2).Msg("how deep am I")
		}()
	}()
	want("logger_test.go:53")
}

// test modes of failure for malformed logging calls
func TestMalformed(t *testing.T) {
	log, want := substringTestLogger(t)

	log.Msg("{}")
	want(missingArg)

	log.Msg("{item}")
	want(missingAttr)

	log.Msg("{something")
	want(missingRightBracket)

	// only string in args - not enough for an Attr
	log.Msg("no interpolation", "not-a-key")
	want(missingArg)

	// no key string
	log.Msg("no interpolation", 0)
	want(missingKey)

	// both bools appear (no deduplication)
	// also: second bool wins for interpolation
	log.With("bit", true).Msg("{bit}", slog.Bool("bit", false))
	want(`"msg":"false","bit":true,"bit":false`)

	// both xs appear, first wins for interpolation
	log.Msg("{}", slog.Int("x", 1), slog.Int("x", 2))
	want(`"msg":"1","x":1,"x":2`)
}

func TestEscaping(t *testing.T) {
	log, want := substringTestLogger(t)

	log.Msg(`\{+\}`)
	want(`"msg":"{+}"`)

	log.Msg(":")
	want(`"msg":":"`)

	log.Msg("{:}", "foo")
	want(`"msg":"foo"`)

	log.Msg(`file\.txt`)
	want(`"msg":"file.txt"`)

	log.With("{}", "x").Msg(`{\{\}}`)
	want(`"msg":"x"`)

	log.With("alpha", "x").Msg("{alpha:%3s}")
	want(`"msg":"  x"`)

	log.With("{}", "x").Msg(`{\{\}:%3s}`)
	want(`"msg":"  x"`)

	log.Msg("{:%3s}", "x")
	want(`"msg":"  x"`)

	log.Msg(`{\:%3s}`, slog.String(`:%3s`, "esc"))
	want(`"msg":"esc"`)

	log.With(`:attr`, "common-lisp").Msg(`{\:attr}`)
	want(`"msg":"common-lisp"`)

	log.Msg("About that struct\\{\\}...")
	want(`"msg":"About that struct{}..."`)

	log.With(":color", "mauve").Msg("The color is {\\:color}.")
	want(`"msg":"The color is mauve."`)

	log.With("x:y ratio", 2).Msg(`What a funny ratio: {x\:y ratio}!`)
	want(`"msg":"What a funny ratio: 2!"`)

	// There is an extra slash introduced by JSON escaping vs Text escaping
	log.Msg(`\{\\`)
	want(`"msg":"{\\"`)

	// Needs JSON Handler; Text escapes the ZWNJ in 👩‍🦰
	log.Err("👩‍🦰", errors.New("🛸"))
	want("👩‍🦰: 🛸")
}

func TestFmt(t *testing.T) {
	log := New().
		Writer(io.Discard).
		Logger()

	msg, err := log.Fmt("{left} <- {root} -> {right}", nil, "left", 0, "right", 2, "root", 1)
	if msg != "0 <- 1 -> 2" {
		t.Errorf("expected 0 <-1 -> 2, got %s", msg)
	}
	if err != nil {
		t.Errorf("expected nil err: %s", err.Error())
	}

	reason := errors.New("reason")
	msg, err = log.Fmt("more info", reason)
	if msg != err.Error() {
		t.Errorf("want equivalence: got msg %s, err %s", msg, err.Error())
	}
	if ok := errors.Is(err, reason); !ok {
		t.Errorf("errors.Is:\n\twant %T, %s\n\tgot  %T, %s", reason, reason.Error(), err, err.Error())
	}
}

func TestGroups(t *testing.T) {
	log, want := substringTestLogger(t)

	// one group
	mulder := slog.Group("1", slog.String("first", "Fox"), slog.String("last", "Mulder"))
	log.Msg("Hi, {1.first} {1.last}", mulder)
	want("Hi, Fox Mulder")

	// two (nested) groups
	scully := slog.Group("2", slog.String("first", "Dana"), slog.String("last", "Scully"))
	agents := slog.Group("agents", mulder, scully)
	log.Msg("Hi, {agents.1.last} and {agents.2.last}", agents)
	want("Hi, Mulder and Scully")

	// raw
	log.Msg("{} {first}", agents, "first", "1?")
	want(`"msg":"{1:{first:Fox last:Mulder} 2:{first:Dana last:Scully}} 1?"`)
}

func TestLabel(t *testing.T) {
	log, want := substringTestLogger(t)

	// one scope
	mulder := log.Label("agent").With("first", "Fox", "last", "Mulder")
	mulder.Msg("Hi, {agent.last}")
	want("Hi, Mulder")

	// another scope
	files := log.Label("files").With("x", true)
	files.Msg("{files.x}")
	want(`"msg":"true"`)

	// two scopes, and a group
	log = log.Label("files").Label("agent").With(slog.Group("name", slog.String("last", "Scully")))
	log.Msg("Hi, {files.agent.name.last}")
	want("Hi, Scully")

	// branching in scope
	log = log.Label("files").With("x", true).Label("agent").With(slog.Group("name", slog.String("last", "Scully")))
	log.Msg("Hi, {files.agent.name.last}")
	want("Hi, Scully")
}

// spoofy types to test LogValuer
type (
	spoof0 struct{}
	spoof1 struct{}
	spoof2 struct{}
)

func (s spoof0) LogValue() slog.Value {
	return slog.StringValue("spoof")
}

func (s spoof1) LogValue() slog.Value {
	return slog.AnyValue(spoof0{})
}

func (s spoof2) LogValue() slog.Value {
	return slog.AnyValue(spoof1{})
}

// test correctness of interpolation and formatting
func TestLoggerKinds(t *testing.T) {
	fs := []struct {
		arg  any
		verb string
		want string
	}{
		{"a", "", `"msg":"a"`},
		{"b", "%10s", `"msg":"         b"`},
		{true, "", `"msg":"true"`},
		{true, "%-6v", `"msg":"true  "`},
		{1, "", `"msg":"1"`},
		{-1, "%+8d", `"msg":"      -1"`},
		{uint64(1), "", `"msg":"1"`},
		{uint64(1), "%+d", `"msg":"+1"`},
		{1.111, "", `"msg":"1.111"`},
		{1.111, "%2.1f", `"msg":"1.1"`},

		// time fmting
		{time.Unix(0, 0), "", `"msg":"1969-12-31T16:00:00.000-08:00"`},
		{time.Unix(0, 0), "RFC3339", `"msg":"1969-12-31T16:00:00-08:00"`},
		{time.Unix(0, 0), "epoch", `"msg":"0"`},
		{time.Unix(0, 0), "kitchen", `"msg":"4:00PM"`},
		{time.Unix(0, 0), "stamp", `"msg":"Dec 31 16:00:00"`},
		{time.Unix(0, 0), "01/02 03;04;05PM '06 -0700", `"msg":"12/31 04:00:00PM '69 -0800"`},

		// duration fmting
		{time.Unix(3661, 0).Sub(time.Unix(0, 0)), "", `"msg":"1h1m1s"`},
		{time.Unix(1, 0).Sub(time.Unix(0, 0)), "", `"msg":"1s"`},
		{time.Unix(1, 0).Sub(time.Unix(0, 999999000)), "", `"msg":"1µs"`},
		{time.Unix(1, 0).Sub(time.Unix(1, 0)), "", `"msg":"0s"`},
		{time.Unix(1, 2).Sub(time.Unix(1, 1)), "epoch", `"msg":"1"`},

		// any fmting
		{struct{}{}, "", `"msg":"{}"`},

		// group
		{slog.Group("row", slog.Int("A", 1), slog.Int("B", 2)), "", `"msg":"{A:1 B:2}"`},

		// LogValuer
		{spoof0{}, "", `"msg":"spoof"`},
		{spoof0{}, "%10s", `"msg":"     spoof"`},
		{spoof2{}, "", `"msg":"spoof"`},
		{spoof2{}, "%10s", `"msg":"     spoof"`},
	}

	log, want := substringTestLogger(t)

	for _, f := range fs {
		msg := fmt.Sprintf("{:%s}", f.verb)
		log.Msg(msg, f.arg)
		want(f.want)
	}
}

func TestSlogterpolate(t *testing.T) {
	var b bytes.Buffer
	h := New().
		Colors(false).
		Writer(&b).
		AddSource(true).
		TTY()

	log := slog.New(h).With("Agent", "Mulder")

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
	}

	log.Info("Hi, {Agent}", "files", "X")
	want("INFO")
	want("Hi, Mulder")
	want("files=X")
	want("logger_test.go")
	b.Reset()
}
