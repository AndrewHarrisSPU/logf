package logf

import (
	"fmt"
	"io"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

func wantAllocs(t *testing.T, label string, want int, fn func()) {
	t.Helper()
	got := int(testing.AllocsPerRun(5, fn))
	if want != got {
		t.Errorf("%s allocs: want %d, got %d", label, want, got)
	}
}

func TestAllocSplicerKinds(t *testing.T) {
	fs := []struct {
		alloc int
		arg   any
		verb  string
	}{
		{0, "string", ""},
		{1, "string", "%10s"},
		{0, true, ""},
		{0, true, "%-6v"},
		{-1, 1, ""},
		{1, -1, "%+8d"},
		{-1, uint64(1), ""},
		{-1, 1.0, ""},
		{1, 1.111, "%2.1f"},
		{0, time.Now(), ""},
		{1, time.Now(), "15;04;03"},
		{0, time.Since(time.Now()), ""},
		{0, struct{}{}, ""},
	}

	var ignore []func()
	var fns []func()
	for _, f := range fs {
		ignore = append(ignore, allocSplicerFuncIgnore(f.arg, f.verb))
		fns = append(fns, allocSplicerFunc(f.arg, f.verb))
	}

	// run alloc tests
	for i, f := range fs {
		label := fmt.Sprintf("%d: %T %s", i, f.arg, f.verb)
		t.Run(label, func(t *testing.T) {
			wantAllocs(t, "ignoring", 1, ignore[i])
			// plus one for safe freezing
			wantAllocs(t, "splicing", f.alloc+1, fns[i])
		})
	}
}

func allocSplicerFuncIgnore(arg any, verb string) func() {
	msg := "none"
	a := slog.Any("key", arg)
	list := []Attr{a}

	return func() {
		s := newSplicer()
		defer s.free()

		s.joinAttrList(list)
		s.scanMessage(msg)
		s.matchAll("", nil, nil)
		if !s.interpolates {
			s.ipol(msg)
		}
		io.WriteString(io.Discard, s.line())
	}
}

func allocSplicerFunc(arg any, verb string) func() {
	var msg string
	if len(verb) == 0 {
		msg = "{}"
	} else {
		msg = fmt.Sprintf("{key:%s}", verb)
	}

	a := slog.Any("key", arg)
	list := []Attr{a}

	return func() {
		s := newSplicer()
		defer s.free()

		s.joinAttrList(list)
		s.scanMessage(msg)
		s.matchAll("", nil, nil)
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
		{0, 0, 0, "string", ""},
		{1, 1, 1, "string", "%10s"},

		// numeric
		{0, 0, 0, true, ""},
		{0, 0, 0, true, "%-6v"},
		{-1, -1, -1, 1, ""},
		{1, 1, 1, -1, "%+8d"},
		{-1, -1, -1, uint64(1), ""},
		{1, 1, 1, 1.0, ""},
		{3, 3, 3, 1.111, "%2.1f"},

		// time
		{0, 0, 0, time.Now(), ""},
		{1, 1, 1, time.Now(), time.Kitchen},
		{0, 0, 0, time.Since(time.Now()), ""},

		// any
		{1, 1, 1, struct{}{}, ""},

		// group
		{2, 2, 2, slog.GroupValue(slog.Int("A", 1), slog.Int("B", 2)), ""},

		// LogValuer
		{0, 0, 0, spoof0{}, ""},
		{1, 1, 1, spoof0{}, "%10s"},
		{0, 0, 0, spoof2{}, ""},
		{1, 1, 1, spoof2{}, "%10s"},
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
			wantAllocs(t, "arg", f.argAlloc+1, argFns[i])
		})
		t.Run("with "+label, func(t *testing.T) {
			wantAllocs(t, "with", f.withAlloc+1, withFns[i])
		})
		t.Run("fmt "+label, func(t *testing.T) {
			wantAllocs(t, "fmt", f.fmtAlloc+1, fmtFns[i])
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
		log.Info(msg, "key", arg)
	}
}

func allocLoggerWithFunc(log Logger, n int, arg any, verb string) func() {
	key := fmt.Sprintf("%d", n)
	msg := fmt.Sprintf("{%s}", key)
	log = log.With(key, arg)

	return func() {
		log.Info(msg)
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
		log.Info("")
	}

	t.Run("group", func(t *testing.T) {
		wantAllocs(t, "group", 0, fn)
	})

	fn = func() { log.Info("{1.roman}") }

	t.Run("group", func(t *testing.T) {
		wantAllocs(t, "group", 1, fn)
	})
}
