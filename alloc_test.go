package logf

import (
	"fmt"
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

func TestAllocKindsSplicer(t *testing.T) {
	fs := []struct {
		alloc int
		arg   any
		verb  string
	}{
		{1, "string", ""},
		{3, "string", "%10s"},
		{1, true, ""},
		{2, true, "%-6v"},
		{0, 1, ""},
		{3, -1, "%+8d"},
		{0, uint64(1), ""},
		{0, 1.0, ""},
		{3, 1.111, "%2.1f"},
		{1, time.Now(), ""},
		// {2, time.Now(), time.Kitchen},
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

		s.join(nil, nil, []any{arg})
		s.interpolate(msg)
		io.WriteString(io.Discard, s.msg())
	}
}

func TestAllocKindsLogger(t *testing.T) {
	fs := []struct {
		alloc    int
		fmtAlloc int
		arg      any
		verb     string
	}{
		{1, 1, "string", ""},
		{3, 3, "string", "%10s"},
		{1, 1, true, ""},
		{2, 2, true, "%-6v"},
		{0, 0, 1, ""},
		{3, 3, -1, "%+8d"},
		{0, 0, uint64(1), ""},
		{0, 0, 1.0, ""},
		{3, 3, 1.111, "%2.1f"},
		{1, 1, time.Now(), ""},
		// {2, 2, time.Now(), time.Kitchen},
		{1, 1, time.Since(time.Now()), ""},
		{1, 1, struct{}{}, ""},
	}

	log := setupDiscardLog()

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
			wantAllocs(t, f.alloc, argFns[i])
		})
		t.Run("with "+label, func(t *testing.T) {
			wantAllocs(t, f.alloc, withFns[i])
		})
		t.Run("fmt "+label, func(t *testing.T) {
			wantAllocs(t, f.fmtAlloc, fmtFns[i])
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
