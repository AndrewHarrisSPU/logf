package logf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

// type TB wraps an instance of testing.TB
type TB struct {
	// turns time on / off in logs
	Time bool

	// adjust depth
	Depth int

	// embed testing.TB, get a lot of free methods...
	testing.TB

	// encoded output writes to buf
	buf bytes.Buffer

	// last record held
	last slog.Record

	// encoder
	enc *Handler
}

func WithTest(t testing.TB) *TB {
	tb := new(TB)
	tb.Depth = 1

	tb.TB = t
	tb.TB.Cleanup(func() {
		tb.Clear()
	})

	enc := slog.HandlerOptions{
		AddSource: true,
	}.NewTextHandler(&tb.buf)

	h := &Handler{
		enc: enc,
	}

	tb.enc = h.WithAttrs([]Attr{
		slog.String("test", t.Name()),
	}).(*Handler)

	return tb
}

// slog.Handler methods

func (tb *TB) Enabled(level slog.Level) bool {
	return true
}

func (tb *TB) Handle(r slog.Record) error {
	tb.last = r
	return tb.enc.Handle(r)
}

func (tb *TB) WithAttrs(as []Attr) slog.Handler {
	tb.enc = tb.enc.withAttrs(as).(*Handler)
	return tb
}

func (tb *TB) WithGroup(name string) slog.Handler {
	tb.enc = tb.enc.withGroup(name).(*Handler)
	return tb
}

// testing.TB Overrides

func (tb *TB) Error(args ...any) {
	tb.TB.Helper()
	tb.record(3, args...)
	tb.Fail()
	tb.dump()
}

func (tb *TB) Errorf(format string, args ...any) {
	tb.recordf(3, format, args...)
	tb.Fail()
	tb.dump()
}

func (tb *TB) Fatal(args ...any) {
	tb.record(3, args...)
	tb.Fail()
	tb.dump()
	tb.FailNow()
}

func (tb *TB) Fatalf(format string, args ...any) {
	tb.recordf(3, format, args...)
	tb.Fail()
	tb.dump()
	tb.FailNow()
}

func (tb *TB) Log(args ...any) {
	tb.record(0, args...)
}

func (tb *TB) Logf(format string, args ...any) {
	tb.recordf(0, format, args...)
}

func (tb *TB) Setenv(key, value string) {
	tb.enc = tb.enc.withAttrs([]Attr{
		slog.String(key, value),
	}).(*Handler)
}

func (tb *TB) Skip(args ...any) {
	tb.record(3, args...)
	tb.SkipNow()
	tb.dump()
}

func (tb *TB) Skipf(format string, args ...any) {
	tb.recordf(3, format, args...)
	tb.SkipNow()
	tb.dump()
}

// TB operations

func (tb *TB) time() (t time.Time) {
	if tb.Time {
		t = time.Now()
	}
	return
}

func (tb *TB) addDepth(depth int) int {
	if depth != 0 {
		return depth + tb.Depth
	}
	return 0
}

func (tb *TB) record(depth int, args ...any) {
	msg := fmt.Sprint(args...)
	r := slog.NewRecord(tb.time(), INFO, msg, tb.addDepth(depth), nil)
	tb.last = r
	tb.enc.Handle(r)
}

func (tb *TB) recordf(depth int, f string, args ...any) {
	msg := fmt.Sprintf(f, args...)
	r := slog.NewRecord(tb.time(), INFO, msg, tb.addDepth(depth), nil)
	tb.last = r
	tb.enc.Handle(r)
}

func (tb *TB) show(msg string) {
	tb.TB.Helper()
	tb.TB.Logf("%s:\n%s\n", msg, tb.buf.String())
	tb.Clear()
}

func (tb *TB) dump() {
	tb.TB.Helper()
	if tb.Failed() && !tb.Skipped() {
		tb.TB.Logf("%s:\n%s\n", tb.TB.Name(), tb.buf.String())
	}
	tb.Clear()
}

// Utility

func (tb *TB) Clear() {
	tb.buf.Reset()
	tb.last = slog.NewRecord(time.Time{}, ERROR, "", 0, nil)
}

// Asserts

func (tb *TB) Want(want string) (found bool) {
	tb.TB.Helper()
	defer tb.Clear()

	if strings.Contains(tb.buf.String(), want) {
		found = true
	}

	if !found {
		tb.TB.Errorf("\nwant: %s\nin:   %s", want, tb.buf.String())
	}

	return
}

func (tb *TB) WantBuffer(want string) (found bool) {
	tb.TB.Helper()
	defer tb.Clear()

	if want == tb.buf.String() {
		found = true
	}

	if !found {
		tb.TB.Errorf("\nwant: %s\nin:   %s", want, tb.buf.String())
	}

	return
}
