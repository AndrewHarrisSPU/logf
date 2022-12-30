package logf

import (
	"io"
	"os"
	"sync"

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
	dev *ttyDevice
	aux slog.Handler

	// unformatted
	store Store
	label Attr

	// attr preformatting
	attrText string
	attrSep  byte

	// tag preformatting
	tagText string
	tagSep  byte
}

type ttyDevice struct {
	w      *ttySyncWriter
	fmtr   *ttyFormatter
	filter *ttyFilter

	ref *slog.LevelVar

	replace replaceFunc
}

// ttySyncWriter manages state relevant to writing bytes, concurrently, on-screen (or wherever)
type ttySyncWriter struct {
	io.Writer
	*sync.Mutex
}

func newTTYSyncWriter(w io.Writer, mu *sync.Mutex) (*ttySyncWriter, bool) {
	var isTTY bool
	file, isFile := w.(*os.File)
	if !isFile {
		isTTY = false
	} else {
		stat, _ := file.Stat()
		isTTY = (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	}
	return &ttySyncWriter{w, mu}, isTTY
}

func (w *ttySyncWriter) Write(p []byte) (n int, err error) {
	w.Lock()
	n, err = w.Writer.Write(p)
	w.Unlock()
	return
}

// ttyFilter manages some state relevant to filtering log lines
type ttyFilter struct {
	tag map[string]struct{}
}

// Logger returns a [Logger] that uses the [TTY] as a handler.
func (tty *TTY) Logger() Logger {
	if tty.dev.w == nil {
		return newLogger(tty.aux.(handler))
	}

	return newLogger(tty)
}

// LogValue returns a [slog.Value], of [slog.GroupKind].
// The group of [Attr]s is the collection of attributes present in log lines handled by the [TTY].
func (tty *TTY) LogValue() slog.Value {
	return tty.store.LogValue()
}

// WriteString satisfies the [io.StringWriter] interface.
// It is safe to call Write concurrently with other methods that write [TTY] output.
// A trailing newline is appended to the output.
// If a program detects that a [TTY] does not write to a terminal device, WriteString is a no-op.
func (tty *TTY) WriteString(s string) (n int, err error) {
	if tty.dev.w == nil {
		return 0, nil
	}

	return io.WriteString(tty.dev.w, s+"\n")
}

// Println formats the given string, and then writes it (with [TTY.WriteString])
func (tty *TTY) Printf(f string, args ...any) {
	if tty.dev.w == nil {
		return
	}

	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	s.joinStore(tty.store, tty.dev.replace)
	for _, a := range Attrs(args...) {
		s.joinLocal(tty.store.scope, a, tty.dev.replace)
	}
	s.ipol(f)

	tty.WriteString(s.line())
}

func (tty *TTY) SetRef(level slog.Level) {
	tty.dev.ref.Set(level)
}

// Filter sets a filter on [TTY] output, using the given set of tags.
func (tty *TTY) Filter(tags ...string) {
	tty.dev.w.Lock()
	defer tty.dev.w.Unlock()

	for tag := range tty.dev.filter.tag {
		delete(tty.dev.filter.tag, tag)
	}

	for _, tag := range tags {
		tty.dev.filter.tag[tag] = struct{}{}
	}
}

// HANDLER

// Enabled reports whether the [TTY] is enabled for logging at the given level.
func (tty *TTY) Enabled(level slog.Level) bool {
	return level >= tty.dev.ref.Level()
}

// See [slog.WithAttrs].
func (tty *TTY) WithAttrs(as []Attr) slog.Handler {
	t2 := *tty

	// find & assign label
	as, t2.label = detectLabel(as, tty.label)

	// store
	t2.store = tty.store.WithAttrs(as)

	// aux
	if t2.aux != nil {
		t2.aux = tty.aux.WithAttrs(as)
	}

	// preformatting
	if t2.dev.w == nil {
		return &t2
	}

	// (for consistency, using splicer methods to write attr and tag text)
	s := newSplicer()
	defer s.free()

	b := &Buffer{s, 0}

	// append attr text
	b.sep = tty.attrSep
	t2.encListAttrs(b, as)

	t2.attrSep = b.sep
	t2.attrText = tty.attrText + s.line()

	// append tag text
	s.text = s.text[:0]
	b.sep = t2.tagSep
	t2.encListTags(b, as)
	t2.tagSep = b.sep
	t2.tagText = tty.tagText + s.line()

	return &t2
}

// See [slog.Handler.WithGroup].
func (tty *TTY) WithGroup(name string) slog.Handler {
	t2 := *tty

	// handler store
	t2.store = tty.store.WithGroup(name)

	// device aux
	if t2.aux != nil {
		t2.aux = tty.aux.WithGroup(name)
	}

	// preformatting
	if t2.dev.w == nil {
		return &t2
	}

	s := newSplicer()
	defer s.free()

	b := &Buffer{s, 0}
	b.sep = tty.attrSep

	b.writeSep()
	b.sep = 0

	t2.dev.fmtr.key.Encode(b, name)
	t2.encAttrGroupOpen(b)

	t2.attrSep = b.sep

	t2.attrText = tty.attrText + s.line()
	return &t2
}

// Handle logs the given [slog.Record] to [TTY] output.
func (tty *TTY) Handle(r slog.Record) (auxErr error) {
	if tty.aux != nil {
		auxErr = tty.aux.Handle(r)
	}

	if tty.dev.w == nil {
		return
	}

	_, enabled := tty.dev.filter.tag[tty.label.Value.String()]

	// formatting
	s := newSplicer()
	defer s.free()

	s.joinStore(tty.store, tty.dev.replace)

	var recordErr error
	r.Attrs(func(a Attr) {
		if a.Key == "#" {
			_, enabled = tty.dev.filter.tag[a.Value.String()]
			return
		}
		if a.Key == "err" {
			if curr, isErr := a.Value.Any().(error); isErr {
				recordErr = curr
			}
		}
		s.joinLocal(tty.store.scope, a, tty.dev.replace)
	})

	if len(tty.dev.filter.tag) > 0 && !enabled {
		return nil
	}

	file, line := r.SourceLine()
	tty.encFields(s, r.Level, r.Message, recordErr, SourceLine{file, line})

	tty.dev.w.Write(s.text)

	return nil
}
