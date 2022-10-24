package logf

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

type zeroTimeHandler struct {
	h slog.Handler
}

func newZeroTimeHandler(h slog.Handler) slog.Handler {
	return zeroTimeHandler{h}
}

func (z zeroTimeHandler) Enabled(level slog.Level) bool {
	return z.h.Enabled(level)
}

func (z zeroTimeHandler) Handle(r slog.Record) error {
	r.Time = time.Time{}
	return z.h.Handle(r)
}

func (z zeroTimeHandler) WithAttrs(seg []Attr) slog.Handler {
	return zeroTimeHandler{z.h.WithAttrs(seg)}
}

func (z zeroTimeHandler) WithGroup(name string) slog.Handler {
	return zeroTimeHandler{z.h.WithGroup(name)}
}

// SUBSTRINGS

func substringTestLogger(t *testing.T, options ...Option) (Logger, func(string)) {
	var b bytes.Buffer
	options = append(options, Using.Writer(&b))

	wantFunc := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	l := New(options...)
	l.h.(*Handler).enc = zeroTimeHandler{l.h.(*Handler).enc}

	return l, wantFunc
}

// DIFF

func setupDiffLog() *diffLogger {
	d := &diffLogger{
		mu:    new(sync.Mutex),
		ref:   new(slog.AtomicLevel),
		level: new(slog.AtomicLevel),
		fbuf:  new(bytes.Buffer),
		cbuf:  new(bytes.Buffer),
		sbuf:  new(bytes.Buffer),
	}

	d.f = New(
		Using.Writer(d.fbuf),
		Using.Level(d.level),
		Using.Source,
	).Depth(2)

	d.f.h.(*Handler).enc = zeroTimeHandler{d.f.h.(*Handler).enc}

	d.c = New(
		Using.Writer(d.cbuf),
		Using.Level(d.level),
		Using.Source,
	).Contextual()

	d.c.h.(*Handler).enc = zeroTimeHandler{d.c.h.(*Handler).enc}

	d.ctx = context.Background()

	// slog options
	slogOptions := slog.HandlerOptions{
		AddSource: true,
		ReplaceAttr: func(a Attr) Attr {
			if a.Key == "time" {
				a.Value = slog.TimeValue(time.Time{})
			}
			return a
		},
	}
	d.s = slog.New(slogOptions.NewTextHandler(d.sbuf))

	return d
}

type diffLogger struct {
	mu               *sync.Mutex
	ref              *slog.AtomicLevel
	level            *slog.AtomicLevel
	fbuf, sbuf, cbuf *bytes.Buffer
	f                Logger
	c                LoggerCtx
	ctx              context.Context
	s                slog.Logger
}

func (d *diffLogger) With(args ...any) *diffLogger {
	d.mu.Lock()
	defer d.mu.Unlock()

	d2 := &diffLogger{
		mu:    new(sync.Mutex),
		ref:   d.ref,
		level: d.level,
		fbuf:  new(bytes.Buffer),
		cbuf:  new(bytes.Buffer),
		sbuf:  new(bytes.Buffer),
	}

	d2.f = d.f.With(args...)
	d2.c = d.c.With(args...)
	d2.s = d.s.With(args...)

	return d2
}

func (d *diffLogger) Diff(t *testing.T, msg string, n int, args ...any) {
	// because buffer writers are non-sync'd ...
	d.mu.Lock()
	defer d.mu.Unlock()

	level := d.level.Level()

	imsg := msg
	for i := 0; i < n; i++ {
		imsg += " {}"
	}

	sargs := args
	smsg := msg
	for i := 0; i < n; i++ {
		smsg = smsg + " " + fmt.Sprint(sargs[0])
		sargs = sargs[1:]
	}

	d.f.Level(level).Msg(imsg, args...)
	fstr := d.fbuf.String()

	d.c.Level(level).Msg(d.ctx, imsg, args...)
	cstr := d.cbuf.String()

	d.s.LogDepth(0, level, smsg, sargs...)
	sstr := d.sbuf.String()

	if fstr != sstr {
		t.Errorf("diff:\n\tlogf: %s\tslog: %s", fstr, sstr)
	}

	if cstr != sstr {
		t.Errorf("diff:\n\tlogc: %s\tslog: %s", cstr, sstr)
	}

	d.fbuf.Reset()
	d.cbuf.Reset()
	d.sbuf.Reset()
}

func stripTime(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.SplitN(s, " ", 2)[1]
}
