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
		{0, 1, ""},
		{1, -1, "%+8d"},
		{0, uint64(1), ""},
		{0, 1.0, ""},
		{1, 1.111, "%2.1f"},
		{0, time.Now(), ""},
		{1, time.Now(), "15;04;03"},
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
			wantAllocs(t, "splicing", f.alloc+1, fns[i])
		})
	}
}

func allocSplicerFunc(arg any, verb string) func() {
	var msg string
	if len(verb) == 0 {
		msg = "{}"
	} else {
		msg = fmt.Sprintf("{key:%s}", verb)
	}

	store := Store{
		scope: []string{},
		as: [][]Attr{
			[]Attr{slog.Any("key", arg)},
		},
	}

	return func() {
		s := newSplicer()
		defer s.free()

		s.scanMessage(msg)
		s.joinStore(store, nil)
		s.ipol(msg)
		io.WriteString(io.Discard, s.line())
	}
}
