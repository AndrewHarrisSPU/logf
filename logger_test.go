package logf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/AndrewHarrisSPU/logf/testlog"
	"golang.org/x/exp/slog"
)

func TestDepth(t *testing.T) {
	h, want := testlog.Substrings(t)
	log := UsingHandler(h)

	loc := func() {
		log.Depth(1).Msg("where am I")
	}

	loc()
	want("logger_test.go:24")

	loc()
	want("logger_test.go:27")

	func() { loc() }()
	want("logger_test.go:30")

	func() {
		func() {
			log.Depth(2).Msg("how deep am I")
		}()
	}()
	want("logger_test.go:37")
}

// test modes of failure for malformed logging calls
func TestMalformed(t *testing.T) {
	h, want := testlog.Substrings(t)
	log := UsingHandler(h)

	log.Msgf("{}")
	want(missingAttr)

	log.Msgf("{item}")
	want(missingAttr)

	// only a single string in args - not enough for an Attr
	log.Msgf("no interpolation", "not-a-key")
	want(missingArg)

	// no key string
	log.Msgf("no interpolation", 0)
	want(missingKey)

	// both xs appear, first wins for interpolation
	log.Msgf("{}", slog.Int("x", 1), slog.Int("x", 2))
	want(`"msg":"1","x":1,"x":2`)

	// both bools appear (no deduplication)
	// also: second bool wins for interpolation
	log.With("bit", true).Msgf("{bit}", slog.Bool("bit", false))
	want(`"msg":"false","bit":true,"bit":false`)
}

func TestEscaping(t *testing.T) {
	h, want := testlog.Substrings(t)
	log := UsingHandler(h)

	log.Msgf(`\{+\}`)
	want(`"msg":"\\{+\\}"`)

	log.Msgf(":")
	want(`"msg":":"`)

	log.Msgf("{:}", "unkeyed", "foo")
	want(`"msg":"foo"`)

	log.Msgf(`file\.txt`)
	want(`"msg":"file\\.txt"`)

	log.With("{}", "x").Msgf(`{\{\}}`)
	want(`"msg":"x"`)

	log.With("alpha", "x").Msgf("{alpha:%3s}")
	want(`"msg":"  x"`)

	log.With("{}", "x").Msgf(`{\{\}:%3s}`)
	want(`"msg":"  x"`)

	log.Msgf("{:%3s}", "unkeyed", "x")
	want(`"msg":"  x"`)

	log.Msgf(`{\:%3s}`, slog.String(`:%3s`, "esc"))
	want(`"msg":"esc"`)

	log.With(`:attr`, "common-lisp").Msgf(`{\:attr}`)
	want(`"msg":"common-lisp"`)

	log.Msgf("About that struct\\{\\}...")
	want(`"msg":"About that struct\\{\\}..."`)

	log.With(":color", "mauve").Msgf("The color is {\\:color}.")
	want(`"msg":"The color is mauve."`)

	log.With("x:y ratio", 2).Msgf(`What a funny ratio: {x\:y ratio}!`)
	want(`"msg":"What a funny ratio: 2!"`)

	// There is an extra slash introduced by JSON escaping vs Text escaping
	log.Msgf(`\{\\`)
	want(`"msg":"\\{\\\\"`)

	// Needs JSON Handler; Text escapes the ZWNJ in 👩‍🦰
	log.Err("👩‍🦰", errors.New("🛸"))
	want("👩‍🦰: 🛸")
}

func TestFmt(t *testing.T) {
	log := New().
		Writer(io.Discard).
		Logger()

	msg := log.Fmt("{left} <- {root} -> {right}", nil, "left", 0, "right", 2, "root", 1)
	if msg != "0 <- 1 -> 2" {
		t.Errorf("expected 0 <-1 -> 2, got %s", msg)
	}

	reason := errors.New("reason")
	err := log.NewErr("more info", reason)
	if ok := errors.Is(err, reason); !ok {
		t.Errorf("errors.Is:\n\twant %T, %s\n\tgot  %T, %s", reason, reason.Error(), err, err.Error())
	}
}

func TestGroups(t *testing.T) {
	h, want := testlog.Substrings(t)
	log := UsingHandler(h)

	// one group
	mulder := slog.Group("1", slog.String("first", "Fox"), slog.String("last", "Mulder"))
	log.Msgf("Hi, {1.first} {1.last}", mulder)
	want("Hi, Fox Mulder")

	// two (nested) groups
	scully := slog.Group("2", slog.String("first", "Dana"), slog.String("last", "Scully"))
	agents := slog.Group("agents", mulder, scully)
	log.Msgf("Hi, {agents.1.last} and {agents.2.last}", agents)
	want("Hi, Mulder and Scully")

	// raw
	log.Msgf("{} {first}", agents, "first", "1?")
	want(`"msg":"[1=[first=Fox last=Mulder] 2=[first=Dana last=Scully]] 1?"`)
}

func TestGroups2(t *testing.T) {
	h, want := testlog.Substrings(t)

	// one scope
	log := UsingHandler(h)
	mulder := log.Group("agent").With("first", "Fox", "last", "Mulder")
	mulder.Msgf("Hi, {agent.last}")
	want("Hi, Mulder")

	// another scope
	log = UsingHandler(h)
	files := log.Group("files").With("x", true)
	files.Msgf("{files.x}")
	want(`"msg":"true"`)

	// two scopes, and a group
	log = UsingHandler(h)
	log = log.Group("files").Group("agent").With(slog.Group("name", slog.String("last", "Scully")))
	log.Msgf("Hi, {files.agent.name.last}")
	want("Hi, Scully")

	// branching in scope
	log = UsingHandler(h)
	log = log.Group("files").With("x", true).Group("agent").With(slog.Group("name", slog.String("last", "Scully")))
	log.Msgf("Hi, {files.agent.name.last}")
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
		{slog.GroupValue(slog.Int("A", 1), slog.Int("B", 2)), "", `"msg":"[A=1 B=2]"`},

		// LogValuer
		{spoof0{}, "", `"msg":"spoof"`},
		{spoof0{}, "%10s", `"msg":"     spoof"`},
		{spoof2{}, "", `"msg":"spoof"`},
		{spoof2{}, "%10s", `"msg":"     spoof"`},
	}

	h, want := testlog.Substrings(t)
	log := UsingHandler(h)

	for _, f := range fs {
		msg := fmt.Sprintf("{:%s}", f.verb)
		log.Msgf(msg, "unkeyed", f.arg)
		want(f.want)
	}
}

func TestSlogterpolate(t *testing.T) {
	var b bytes.Buffer
	h := New().
		Colors(false).
		Level(LevelText).
		Layout("level", "message", "attrs", "source").
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
	want("files:X")
	want("logger_test.go")
	b.Reset()
}

func TestReplaceTTY(t *testing.T) {
	var b bytes.Buffer

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	log := New().
		ReplaceFunc(func(a Attr) Attr {
			if a.Key == "secret" {
				a.Value = slog.StringValue("redacted")
			}
			return a
		}).
		Layout("message", "\t", "attrs").
		Writer(&b).
		Colors(false).
		ForceTTY().
		Logger()

	log = log.With("secret", 1)

	log.Msgf("{secret}", "secret", 2)
	want(`redacted   secret:redacted secret:redacted`)

	log.Msgf("{group.secret}, {group.group2.secret}", Group("group", Attrs(
		KV("secret", 3),
		Group("group2", Attrs(
			KV("secret", 4),
			KV("secret", 5),
		)),
	)))
	want(`redacted, redacted   secret:redacted group:{secret:redacted group2:{secret:redacted secret:redacted}}`)
}

func TestReplaceSlog(t *testing.T) {
	var b bytes.Buffer

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	log := New().
		ReplaceFunc(func(a Attr) Attr {
			if a.Key == "secret" {
				a.Value = slog.StringValue("redacted")
			}
			return a
		}).
		Writer(&b).
		Colors(false).
		JSON()

	log = log.With("secret", 1)

	log.Msg("{secret}", "secret", 2)
	want(`"msg":"{secret}","secret":"redacted","secret":"redacted"`)

	log.Msg("{group.secret}, {group.group2.secret}", Group("group", Attrs(
		KV("secret", 3),
		Group("group2", Attrs(
			KV("secret", 4),
			KV("secret", 5),
		)),
	)))
	want(`"msg":"{group.secret}, {group.group2.secret}","secret":"redacted","group":{"secret":"redacted","group2":{"secret":"redacted","secret":"redacted"}}`)
}
