package logf

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"golang.org/x/exp/slog"
)

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

	return New(options...), wantFunc
}

// DISCARD

func setupDiscardLog() Logger {
	d := discardSink{make([]Attr, 0)}
	return New(Using.Handler(&d))
}

type discardSink struct {
	as []Attr
}

func (*discardSink) Enabled(slog.Level) bool {
	return true
}

func (*discardSink) Handle(slog.Record) error {
	return nil
}

func (d *discardSink) WithAttrs(as []Attr) slog.Handler {
	return &discardSink{concat(d.as, as)}
}

// TODO
func (d *discardSink) WithGroup(string) slog.Handler {
	return d
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
	)

	d.c = New(
		Using.Writer(d.cbuf),
		Using.Level(d.level),
		Using.Source,
	).Contextual()

	d.ctx = context.Background()

	// slog options
	slogOptions := slog.HandlerOptions{
		AddSource: true,
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

	// logf
	d2.f = New(
		Using.Writer(d2.fbuf),
		Using.Level(d.level),
		Using.Source,
	).With(args...)

	// logf contextual
	d2.c = New(
		Using.Writer(d2.cbuf),
		Using.Level(d.level),
		Using.Source,
	).Contextual()

	d2.ctx = context.WithValue(d.ctx, segmentKey{}, Segment(args...))

	// slog
	slogOptions := slog.HandlerOptions{
		AddSource: true,
	}
	d2.s = slog.New(slogOptions.NewTextHandler(d2.sbuf)).With(args...)

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
	fstr := stripTime(d.fbuf.String())

	d.c.Level(level).Msg(d.ctx, imsg, args...)
	cstr := stripTime(d.cbuf.String())

	d.s.LogDepth(0, level, smsg, sargs...)
	sstr := stripTime(d.sbuf.String())

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
