package logf

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

// test modes of failure for malformed logging calls
func TestMalformed(t *testing.T) {
	log, want := substringTestLogger(t, Using.JSON)

	log.Msg("{}")
	want(missingArg)

	log.Msg("{item}")
	want(missingKey)

	log.Msg("{something")
	want(missingRightBracket)

	log.Msg("no interpolation", "not-a-key")
	want(missingArg)

	// no key string
	log.Msg("no interpolation", 0)
	want(missingKey)

	// can't interpolate from arg segment
	log.Msg("{x}", slog.Int("x", 1))
	want(missingKey)

	// both bools appear (no deduplication)
	log.With("bool", true).Msg("{bool}", slog.Bool("bool", false))
	want(`"msg":"true","bool":true,"bool":false`)

	// just the second x appears (first consumed by {} in msg)
	log.Msg("{}", slog.Int("x", 1), slog.Int("x", 2))
	want(`"msg":"1","x":2`)

	// Needs JSON Handler at the moment
	log.Err("👩‍🦰", errors.New("🛸"))
	want("👩‍🦰")
}

// test error interpolation/wrapping behaviors
func TestLoggerErr(t *testing.T) {
	log, want := substringTestLogger(t)

	reason := errors.New("reason")
	log.Err("more info", reason)
	want("more info: reason")

	msg, err := log.Fmt("more info", reason)
	log.Msg(msg)
	want("more info: reason")

	log.Err("", err)
	want("more info: reason")

	if ok := errors.Is(err, reason); !ok {
		t.Errorf("errors.Is:\n\twant %T, %s\n\tgot  %T, %s", reason, reason.Error(), err, err.Error())
	}
}

// test correctness of interpolation and formatting
func TestLoggerKinds(t *testing.T) {
	fs := []struct {
		arg  any
		verb string
		want string
	}{
		{"a", "", "msg=a"},
		{"b", "%10s", "msg=\"         b\""},
		{true, "", "msg=true"},
		{true, "%-6v", "msg=\"true  \""},
		{1, "", "msg=1"},
		{-1, "%+8d", "msg=\"      -1\""},
		{uint64(1), "", "msg=1"},
		{uint64(1), "%+d", "msg=+1"},
		{1.111, "", "msg=1.111"},
		{1.111, "%2.1f", "msg=1.1"},

		// time fmting
		{time.Unix(0, 0), "", "msg=1969-12-31T16:00:00.000-08:00"},
		{time.Unix(0, 0), time.Kitchen, "msg=4:00PM"},
		// duration fmting
		{time.Unix(3661, 0).Sub(time.Unix(0, 0)), "", "msg=1h1m1s"},
		{time.Unix(1, 0).Sub(time.Unix(0, 0)), "", "msg=1s"},
		{time.Unix(1, 0).Sub(time.Unix(0, 999999000)), "", "msg=1µs"},
		{time.Unix(1, 0).Sub(time.Unix(1, 0)), "", "msg=0s"},
		// any fmting
		{struct{}{}, "", "msg={}"},
	}

	log, want := substringTestLogger(t)

	for _, f := range fs {
		msg := fmt.Sprintf("{:%s}", f.verb)
		log.Msg(msg, f.arg)
		want(f.want)
	}
}

// test outputs agains canonical slog output
// diagnostically, not sharp but broad
// covers Logger and CtxLogger against slog
func TestDiff(t *testing.T) {
	f := struct {
		msg  string
		seg  []any
		args []any
	}{
		"Hi, Mulder",
		[]any{"Agent", "Scully"},
		[]any{"files", "X"},
	}

	log := setupDiffLog().With(f.seg...)

	// level testing
	log.Diff(t, f.msg, f.args...)

	log.ref.Set(ERROR)
	log.Diff(t, f.msg, f.args...)

	log.ref.Set(DEBUG)
	log.Diff(t, f.msg, f.args...)

	// in parallel ...
	n := 100
	wg := new(sync.WaitGroup)
	wg.Add(n)

	for i := 0; i < n; i++ {
		glog := log.With("x", i)

		go func() {
			glog.Diff(t, f.msg, f.args...)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestSlogterpolate(t *testing.T) {
	var b bytes.Buffer
	h := NewHandler(Using.Writer(&b), Using.Source)
	log := slog.New(h).With("Agent", "Mulder")

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
	}

	log.Info("Hi, {Agent}")
	want("INFO")
	want("Hi, Mulder")
	want("logger_test.go")
	b.Reset()
}
