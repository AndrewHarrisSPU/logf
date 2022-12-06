package logf

import (
	"fmt"
	"io"
	"testing"
	"time"

	"golang.org/x/exp/slog"
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
		{0, "string", ""},
		{0, "string", "%10s"},
		{0, true, ""},
		{0, true, "%-6v"},
		{0, 1, ""},
		{0, -1, "%+8d"},
		{0, uint64(1), ""},
		{0, 1.0, ""},
		{0, 1.111, "%2.1f"},
		{0, time.Now(), ""},
		{0, time.Now(), time.Kitchen},
		{0, time.Since(time.Now()), ""},
		{0, struct{}{}, ""},
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

		s.join("", nil, []any{arg}, nil)
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
		{2, 2, 2, 1.0, ""},
		{4, 4, 4, 1.111, "%2.1f"},

		// time
		{1, 1, 1, time.Now(), ""},
		{0, 0, 0, time.Now(), time.Kitchen},
		{1, 1, 1, time.Since(time.Now()), ""},

		// any
		{2, 2, 2, struct{}{}, ""},

		// group
		{3, 3, 3, slog.GroupValue(slog.Int("A", 1), slog.Int("B", 2)), ""},

		// LogValuer
		{2, 2, 2, spoof0{}, ""},
		{3, 3, 3, spoof0{}, "%10s"},
		{2, 2, 2, spoof2{}, ""},
		{3, 3, 3, spoof2{}, "%10s"},
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
			wantAllocs(t, f.argAlloc+3, argFns[i])
		})
		t.Run("with "+label, func(t *testing.T) {
			wantAllocs(t, f.withAlloc+3, withFns[i])
		})
		t.Run("fmt "+label, func(t *testing.T) {
			wantAllocs(t, f.fmtAlloc+3, fmtFns[i])
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
		log.Msg(msg, "key", arg)
	}
}

func allocLoggerWithFunc(log Logger, n int, arg any, verb string) func() {
	key := fmt.Sprintf("%d", n)
	msg := fmt.Sprintf("{%s}", key)
	log = log.With(key, arg)

	return func() {
		log.Msg(msg)
	}
}

func allocLoggerFmtFunc(log Logger, n int, arg any, verb string) func() {
	key := fmt.Sprintf("%d", n)
	msg := fmt.Sprintf("{%s}", key)
	log = log.With(key, arg)

	return func() {
		_ = log.Fmt(msg)
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
