package logf

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"log/slog"
)

func TestFmt(t *testing.T) {
	msg := Fmt("{left} <- {root} -> {right}", "left", 0, "right", 2, "root", 1)
	if msg != "0 <- 1 -> 2" {
		t.Errorf("expected 0 <-1 -> 2, got %s", msg)
	}

	reason := errors.New("reason")
	err := WrapErr("more info", reason)
	if ok := errors.Is(err, reason); !ok {
		t.Errorf("errors.Is:\n\twant %T, %s\n\tgot  %T, %s", reason, reason.Error(), err, err.Error())
	}
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
func TestFmtKinds(t *testing.T) {
	want := func(ok string, got string) {
		if ok != got {
			t.Errorf("want: %s, got: %s", ok, got)
		}
	}

	fs := []struct {
		arg  any
		verb string
		want string
	}{
		{"a", "", `a`},
		{"b", "%10s", `         b`},
		{true, "", `true`},
		{true, "%-6v", `true  `},
		{1, "", `1`},
		{-1, "%+8d", `      -1`},
		{uint64(1), "", `1`},
		{uint64(1), "%+d", `+1`},
		{1.111, "", `1.111`},
		{1.111, "%2.1f", `1.1`},

		// time fmting
		{time.Unix(0, 0), "", `1969-12-31T16:00:00.000-08:00`},
		{time.Unix(0, 0), "RFC3339", `1969-12-31T16:00:00-08:00`},
		{time.Unix(0, 0), "epoch", `0`},
		{time.Unix(0, 0), "kitchen", `4:00PM`},
		{time.Unix(0, 0), "stamp", `Dec 31 16:00:00`},
		{time.Unix(0, 0), "01/02 03;04;05PM '06 -0700", `12/31 04:00:00PM '69 -0800`},

		// duration fmting
		{time.Unix(3661, 0).Sub(time.Unix(0, 0)), "", `1h1m1s`},
		{time.Unix(1, 0).Sub(time.Unix(0, 0)), "", `1s`},
		{time.Unix(1, 0).Sub(time.Unix(0, 999999000)), "", `1µs`},
		{time.Unix(1, 0).Sub(time.Unix(1, 0)), "", `0s`},
		{time.Unix(1, 2).Sub(time.Unix(1, 1)), "epoch", `1`},

		// any fmting
		{struct{}{}, "", `{}`},

		// group
		{slog.GroupValue(slog.Int("A", 1), slog.Int("B", 2)), "", `[A=1 B=2]`},

		// LogValuer
		{spoof0{}, "", `spoof`},
		{spoof0{}, "%10s", `     spoof`},
		{spoof2{}, "", `spoof`},
		{spoof2{}, "%10s", `     spoof`},
	}

	for _, f := range fs {
		msg := fmt.Sprintf("{key:%s}", f.verb)
		want(f.want, Fmt(msg, "key", f.arg))
	}
}

func TestLoggerLogValuer(t *testing.T) {
	want := func(ok string, got string) {
		if ok != got {
			t.Errorf("want: %s, got: %s", ok, got)
		}
	}

	// Logging
	log := New().
		ForceTTY(true).
		Logger()

	// one scope
	mulder := log.WithGroup("agent").With("first", "Fox", "last", "Mulder")
	// mulder.Info("Hi, {agent.last}")
	want("Hi, Mulder", Fmt("Hi, {agent.last}", mulder))

	// another scope
	files := log.WithGroup("files").With("x", true)
	want("true", Fmt("{files.x}", files))

	// two scopes, and a group
	log2 := log.WithGroup("files").WithGroup("agent").With(slog.Group("name", slog.String("last", "Scully")))
	want("Hi, Scully", Fmt("Hi, {files.agent.name.last}", log2))

	// branching in scope
	log3 := log.WithGroup("files").With("x", true).WithGroup("agent").With(slog.Group("name", slog.String("last", "Scully")))
	want("Hi, Scully", Fmt("Hi, {files.agent.name.last}", log3))
}

// test modes of failure for malformed formatting calls
func TestMalformed(t *testing.T) {
	log := New().JSON()

	want := func(ok string, got string) {
		if ok != got {
			t.Errorf("want: %s, got: %s", ok, got)
		}
	}

	want(missingAttr, Fmt("{}"))
	want(missingMatch.String(), Fmt("{item}"))

	// both xs appear, first wins for interpolation
	want("1", Fmt("{}", slog.Int("x", 1), slog.Int("x", 2)))

	// econd bool wins for interpolation
	want("false", Fmt("{bit}", log, slog.Bool("bit", false)))
}

func TestEscaping(t *testing.T) {
	want := func(ok string, got string) {
		if ok != got {
			t.Errorf("want: %s, got: %s", ok, got)
		}
	}

	log := New().ForceTTY(true).Logger()

	want(`{+}`, Fmt(`\{+\}`))

	want(":", Fmt(":"))

	want("foo", Fmt("{:}", "", "foo"))

	want(`file.txt`, Fmt(`file\.txt`))

	log1 := log.With("{}", "x")
	want(`x`, Fmt(`{\{\}}`, log1))

	log2 := log.With("alpha", "x")
	want(`  x`, Fmt("{alpha:%3s}", log2))

	log3 := log.With("{}", "x")
	want(`  x`, Fmt(`{\{\}:%3s}`, log3))

	want(`  x`, Fmt("{:%3s}", "", "x"))

	want(`esc`, Fmt(`{\:%3s}`, ":%3s", "esc"))

	log4 := log.With(":attr", "common-lisp")
	want(`common-lisp`, Fmt(`{\:attr}`, log4))

	want("About that struct{}...", Fmt("About that struct\\{\\}..."))

	log5 := log.With(":color", "mauve")
	want("The color is mauve", Fmt(`The color is {\:color}`, log5))

	log6 := log.With("x:y ratio", 2)
	want("2", Fmt(`{x\:y ratio}`, log6))

	log7 := log.With("👩‍🦰", "🛸")
	want("🛸", Fmt("{👩‍🦰}", log7))
}

func TestGroupsFmt(t *testing.T) {
	want := func(ok string, got string) {
		if ok != got {
			t.Errorf("want: %s, got: %s", ok, got)
		}
	}

	// one group
	mulder := slog.Group("1", slog.String("first", "Fox"), slog.String("last", "Mulder"))
	want("Hi, Fox Mulder", Fmt("Hi, {1.first} {1.last}", mulder))

	// two (nested) groups
	scully := slog.Group("2", slog.String("first", "Dana"), slog.String("last", "Scully"))
	agents := slog.Group("agents", mulder, scully)
	want("Hi, Mulder and Scully", Fmt("Hi, {agents.1.last} and {agents.2.last}", agents))

	want("[1=[first=Fox last=Mulder] 2=[first=Dana last=Scully]] 1?", Fmt("{} {first}", agents, "first", "1?"))
}
