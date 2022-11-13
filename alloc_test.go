package logf

import (
	"fmt"
	"golang.org/x/exp/slog"
	"io"
	"testing"
	"time"
)

func wantAllocs(t *testing.T, want int, fn func()) {
	t.Helper()
	got := int(testing.AllocsPerRun(5, fn))
	if want != got {
		t.Errorf("allocs: want %d, got %d", want, got)
	}
}

func TestAllocSplicerKinds(t *testing.T) {
	fs := []struct {
		alloc int
		arg   any
		verb  string
	}{
		{1, "string", ""},
		{1, "string", "%10s"},
		{1, true, ""},
		{1, true, "%-6v"},
		{1, 1, ""},
		{1, -1, "%+8d"},
		{1, uint64(1), ""},
		{1, 1.0, ""},
		{1, 1.111, "%2.1f"},
		{1, time.Now(), ""},
		{1, time.Now(), time.Kitchen},
		{1, time.Since(time.Now()), ""},
		{1, struct{}{}, ""},
	}

	var fns []func()
	for _, f := range fs {
		fns = append(fns, allocSplicerFunc(f.arg, f.verb))
	}

	// run alloc tests
	for i, f := range fs {
		label := fmt.Sprintf("%d: %T %s", i, f.arg, f.verb)
		t.Run(label, func(t *testing.T) {
			// plus one for safe freezing
			wantAllocs(t, f.alloc, fns[i])
		})
	}
}

func allocSplicerFunc(arg any, verb string) func() {
	var msg string
	if len(verb) == 0 {
		msg = "{}"
	} else {
		msg = fmt.Sprintf("{:%s}", verb)
	}

	return func() {
		s := newSplicer()
		defer s.free()

		s.join("", nil, []any{arg})
		s.ipol(msg)
		io.WriteString(io.Discard, s.line())
	}
}

func TestAllocLoggerKinds(t *testing.T) {
	fs := []struct {
		argAlloc  int
		withAlloc int
		fmtAlloc  int
		arg       any
		verb      string
	}{
		// strings
		{0, 1, 1, "string", ""},
		{2, 2, 2, "string", "%10s"},

		// numeric
		{1, 1, 1, true, ""},
		{1, 1, 1, true, "%-6v"},
		{0, 0, 0, 1, ""},
		{2, 2, 2, -1, "%+8d"},
		{0, 0, 0, uint64(1), ""},
		{0, 0, 0, 1.0, ""},
		{2, 2, 2, 1.111, "%2.1f"},

		// time
		{1, 1, 1, time.Now(), ""},
		{1, 1, 1, time.Now(), time.Kitchen},
		{1, 1, 1, time.Since(time.Now()), ""},

		// any
		{1, 1, 1, struct{}{}, ""},

		// group
		{3, 3, 3, slog.Group("row", slog.Int("A", 1), slog.Int("B", 2)), ""},

		// LogValuer
		{1, 1, 1, spoof0{}, ""},
		{2, 2, 2, spoof0{}, "%10s"},
		{1, 1, 1, spoof2{}, ""},
		{2, 2, 2, spoof2{}, "%10s"},
	}

	log := New().
		Writer(io.Discard).
		JSON()

	var argFns []func()
	var withFns []func()
	var fmtFns []func()

	for i, f := range fs {
		argFns = append(argFns, allocLoggerArgFunc(log, f.arg, f.verb))
		withFns = append(argFns, allocLoggerWithFunc(log, i, f.arg, f.verb))
		fmtFns = append(argFns, allocLoggerFmtFunc(log, i, f.arg, f.verb))
	}

	for i, f := range fs {
		label := fmt.Sprintf("%d: %T %s", i, f.arg, f.verb)
		t.Run("arg "+label, func(t *testing.T) {
			wantAllocs(t, f.argAlloc+2, argFns[i])
		})
		t.Run("with "+label, func(t *testing.T) {
			wantAllocs(t, f.withAlloc+2, withFns[i])
		})
		t.Run("fmt "+label, func(t *testing.T) {
			wantAllocs(t, f.fmtAlloc+2, fmtFns[i])
		})
	}
}

func allocLoggerArgFunc(log Logger, arg any, verb string) func() {
	var msg string
	if len(verb) > 0 {
		msg = fmt.Sprintf("{:%s}", verb)
	} else {
		msg = "{}"
	}

	return func() {
		log.Msg(msg, arg)
	}
}

func allocLoggerWithFunc(log Logger, n int, arg any, verb string) func() {
	key := fmt.Sprintf("%d", n)
	msg := fmt.Sprintf("{%s}", key)
	log = log.With(key, arg)

	return func() {
		log.Msg(msg, arg)
	}
}

func allocLoggerFmtFunc(log Logger, n int, arg any, verb string) func() {
	key := fmt.Sprintf("%d", n)
	msg := fmt.Sprintf("{%s}", key)
	log = log.With(key, arg)

	return func() {
		_, _ = log.Fmt(msg, nil)
	}
}

func TestAllocLoggerGroups(t *testing.T) {
	log := New().
		Writer(io.Discard).
		Text()

	g := slog.Group("1", slog.String("roman", "i"))
	log = log.With(g)

	fn := func() {
		log.Msg("")
	}

	t.Run("group", func(t *testing.T) {
		wantAllocs(t, 1, fn)
	})

	fn = func() { log.Msg("{1.roman}") }

	t.Run("group", func(t *testing.T) {
		wantAllocs(t, 1, fn)
	})
}
