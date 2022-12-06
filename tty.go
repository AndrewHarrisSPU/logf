package logf

import (
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// TTY HANDLER

// TTY is a component that displays log lines.
//
// A TTY is a [slog.Handler], and an [io.StringWriter].
//
// On creation, a [TTY] detects whether it is writing to a terminal.
// If not, log lines are are written to the writer by a [slog.JSONHandler].
//
// Some TTY examples can be run with files in the demo folder:
//
//	go run demo/<some demo file>.go
type TTY struct {
	fmtr *ttyFormatter

	// group
	scope   string
	openKey string
	nOpen   int

	// attrs
	attrs    []Attr
	attrText string
	attrSep  byte

	// tags
	tag     Attr
	tagText string
	tagSep  byte
}

// ttyEncoder manages state relevant to encoding a record to bytes
type ttyFormatter struct {
	sink   *ttySink
	layout []ttyField
	tag    map[string]ttyEncoder[Attr]

	time       ttyEncoder[time.Time]
	level      ttyEncoder[slog.Level]
	message    ttyEncoder[string]
	key        ttyEncoder[string]
	value      ttyEncoder[Value]
	source     ttyEncoder[SourceLine]
	groupOpen  Encoder[struct{}]
	groupClose Encoder[int]

	groupPen pen
	debugPen pen
	infoPen  pen
	warnPen  pen
	errorPen pen

	addSource bool
}

// ttySink manages state relevant to writing bytes on-screen (or wherever)
type ttySink struct {
	w       io.Writer
	ref     slog.Leveler
	mu      *sync.Mutex
	replace func(Attr) Attr

	enabled bool
}

func (tty *TTY) bounceJSON() Logger {
	cfg := &Config{
		w:       tty.fmtr.sink.w,
		ref:     tty.fmtr.sink.ref,
		replace: tty.fmtr.sink.replace,
	}

	cfg.AddSource(tty.fmtr.addSource)

	log := cfg.JSON()

	log.With(tty.attrs)
	if tty.scope != "" {
		for _, name := range strings.Split(tty.scope, ".") {
			log.Group(name)
		}
	}

	return log
}

// Logger returns a [Logger] that uses the [TTY] as a handler.
func (tty *TTY) Logger() Logger {
	if !tty.fmtr.sink.enabled {
		return tty.bounceJSON()
	}

	return Logger{h: tty}
}

// LogValue returns a [slog.Value], of [slog.GroupKind].
// The group of [Attr]s is the collection of attributes present in log lines handled by the [TTY].
func (tty *TTY) LogValue() slog.Value {
	return slog.GroupValue(tty.attrs...)
}

// WriteString satisfies the [io.StringWriter] interface.
// It is safe to call Write concurrently with other methods that write [TTY] output.
// A trailing newline is appended to the string.
// If a program detects that a [TTY] does not write to a terminal device, WriteString is a no-op.
func (tty *TTY) WriteString(s string) (n int, err error) {
	if !tty.fmtr.sink.enabled {
		return 0, nil
	}

	tty.fmtr.sink.mu.Lock()
	defer tty.fmtr.sink.mu.Unlock()

	return io.WriteString(tty.fmtr.sink.w, s+"\n")
}

// HANDLER

// Enabled reports whether the [TTY] is enabled for logging at the given level.
func (tty *TTY) Enabled(level slog.Level) bool {
	return level >= tty.fmtr.sink.ref.Level()
}

// See [slog.WithAttrs].
func (tty *TTY) WithAttrs(as []Attr) slog.Handler {
	t2 := *tty

	// attr copy & extend
	scoped := scopeAttrs(t2.scope, as, t2.fmtr.sink.replace)
	t2.attrs = concat(tty.attrs, scoped)

	// for consistency, use splicer methods to write attr text
	// (but not one from the pool)
	s := newSplicer()
	defer s.free()

	b := &Buffer{s, 0}

	// append attr text
	b.sep = tty.attrSep
	if len(t2.openKey) > 0 {
		b.writeSep()

		t2.fmtr.key.color.use(b)
		t2.fmtr.key.Encode(b, t2.openKey)
		t2.fmtr.key.color.drop(b)

		t2.encAttrGroupOpen(b, t2.openKey)
		t2.openKey = ""
	}
	t2.encListAttrs(b, "", as)

	t2.attrSep = b.sep
	t2.attrText = tty.attrText + s.line()

	// append tag text
	s.text = s.text[:0]
	b.sep = t2.tagSep
	t2.encListTags(b, "", as)
	t2.tagSep = b.sep
	t2.tagText = tty.tagText + s.line()

	return &t2
}

// See [slog.Handler.WithGroup].
func (tty *TTY) WithGroup(name string) slog.Handler {
	t2 := *tty
	t2.openKey = name
	t2.nOpen = tty.nOpen + 1
	t2.scope = tty.scope + name + "."

	return &t2
}

func (tty *TTY) withTag(tag string) handler {
	t2 := *tty
	t2.tag = slog.String("#", tag)

	return &t2
}

// Handle logs the given [slog.Record] to [TTY] output.
func (tty *TTY) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.scan(r.Message, nil)
	s.join(tty.scope, tty.attrs, nil, tty.fmtr.sink.replace)

	var err error
	r.Attrs(func(a Attr) {
		// capture the latest "err"-keyed Attr
		if a.Key == "err" {
			if curr, isErr := a.Value.Any().(error); isErr {
				err = curr
			}
		}
		// in all cases, capture the attr
		s.joinOne(tty.scope, a, tty.fmtr.sink.replace)
	})

	file, line := r.SourceLine()
	tty.encFields(s, r.Level, r.Message, err, SourceLine{file, line})

	tty.fmtr.sink.mu.Lock()
	defer tty.fmtr.sink.mu.Unlock()

	tty.fmtr.sink.w.Write(s.text)

	return nil
}

func (tty *TTY) handle(
	s *splicer,
	level slog.Level,
	msg string,
	err error,
	depth int,
	args []any,
) error {
	defer s.free()

	s.join(tty.scope, tty.attrs, args, tty.fmtr.sink.replace)

	src := tty.yankSourceLine(depth)

	tty.encFields(s, level, msg, err, src)

	tty.fmtr.sink.mu.Lock()
	defer tty.fmtr.sink.mu.Unlock()

	tty.fmtr.sink.w.Write(s.text)

	return nil
}

func (tty *TTY) fmt(msg string, args []any) *splicer {
	s := newSplicer()

	s.join(tty.scope, tty.attrs, s.scan(msg, args), tty.fmtr.sink.replace)
	s.ipol(msg)

	return s
}
