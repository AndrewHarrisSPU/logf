package logf

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// TTY HANDLER

// TTY is a component that displays log lines.
//
// A TTY is a [slog.Handler], an [io.StringWriter], and an [io.Closer].
//
// On creation, a [TTY] detects whether it is writing to a terminal.
// If not, log lines are are written to the writer by a [slog.JSONHandler].
//
// Additionally, a TTY constructs [Logger]s:
//   - [TTY.Logger] emits complete log lines
//   - [TTY.Printer] just emits log messages
type TTY struct {
	enc *ttyEncoder

	attrs    []Attr
	attrText []byte
	scope    string
	label    Attr
}

// ttyEncoder manages state relevant to encoding a record to bytes
type ttyEncoder struct {
	sink *ttySink

	fields     []ttyField
	timeFormat string
	colors     bool
	elapsed    bool
	addSource  bool
}

// ttySink manages state relevant to writing bytes on-screen (or wherever)
type ttySink struct {
	ref     slog.LevelVar
	refBase slog.Leveler
	w       io.Writer
	refresh *time.Ticker
	done    chan struct{}
	mu      *sync.Mutex
	start   time.Time
	replace func(Attr) Attr

	// stream buffer
	stream ttyStream
	enabled bool
}

type ttyStream struct {
	logLevel  slog.Level
	holdLevel slog.Level
	hold      spinner
	sample    spinner
	enabled   bool
}

type spinner struct {
	lines      []string
	cap        int
	i, written int
}

func (tty *TTY) bounceJSON() *Logger {
	cfg := &Config{
		w:	tty.enc.sink.w,
		ref: &tty.enc.sink.ref,
		replace: tty.enc.sink.replace,
		addSource: tty.enc.addSource,
	}

	log := cfg.JSON()

	log.With(tty.attrs)
	if tty.scope != "" {
		for _, name := range strings.Split(tty.scope, "."){
			log.Group(name)
		}
	}

	return log
}

// Logger returns a [Logger] that uses the [TTY] as a handler.
func (tty *TTY) Logger() *Logger {
	if !tty.enc.sink.enabled {
		return tty.bounceJSON()
	}

	return &Logger{h: tty}
}

// LogValue returns a [slog.Value], of [slog.GroupKind].
// The group of [Attr]s is the collection of attributes present in log lines handled by the [TTY].
func (tty *TTY) LogValue() slog.Value {
	return slog.GroupValue(tty.attrs...)
}

// StartTimeNow sets a start time used when reporting elapsed time.
func (tty *TTY) StartTimeNow() {
	tty.enc.sink.mu.Lock()
	defer tty.enc.sink.mu.Unlock()

	tty.enc.sink.start = time.Now()
}

// WriteString satisfies the [io.StringWriter] interface.
// It is safe to call Write concurrently with other methods that write [TTY] output.
// A trailing newline is appended to the string.
// Write trims the [TTY]'s spin buffer, if enabled.
// If a program detects that a [TTY] does not write to a terminal device, WriteString is a no-op.
func (tty *TTY) WriteString(s string) (n int, err error) {
	if !tty.enc.sink.enabled {
		return 0, nil
	}

	tty.enc.sink.mu.Lock()
	defer tty.enc.sink.mu.Unlock()

	if tty.enc.sink.stream.enabled {
		tty.enc.sink.stream.sample.hide(tty.enc.sink.w)
		tty.enc.sink.stream.hold.hide(tty.enc.sink.w)
		defer func() {
			tty.enc.sink.stream.hold.show(tty.enc.sink.w)
		}()
	}

	return io.WriteString(tty.enc.sink.w, s+"\n")
}

// Close satisfies the [io.Closer] interface.
// It should be called:
//   - to invoke Close on a [TTY]'s writer
//   - on a streaming [TTY], to trim on-screen output and prevent a go routine leak.
func (tty *TTY) Close() error {
	tty.enc.sink.mu.Lock()
	defer tty.enc.sink.mu.Unlock()

	if tty.enc.sink.stream.enabled {
		tty.enc.sink.refresh.Stop()
		tty.enc.sink.stream.hold.hide(tty.enc.sink.w)
		tty.enc.sink.stream.sample.hide(tty.enc.sink.w)
		close(tty.enc.sink.done)
	}

	if wc, isCloser := tty.enc.sink.w.(io.Closer); isCloser {
		return wc.Close()
	}

	return nil
}

// HANDLER

// Enabled reports whether the [TTY] is enabled for logging at the given level.
// If the [TTY]'s spin buffer is enabled, the [TTY] is enabled at or above the spin buffer's reference level.
// Otherwise, the [TTY] is enabled at or above the [TTY]'s  reference level.
func (tty *TTY) Enabled(level slog.Level) bool {
	return level >= tty.enc.sink.ref.Level()
}

// With differs from [WithAttrs] only in that it returns a *TTY.
func (tty *TTY) With(args ...any) *TTY {
	return tty.WithAttrs(Attrs(args...)).(*TTY)
}

// See [slog.WithAttrs].
func (tty *TTY) WithAttrs(as []Attr) slog.Handler {
	tty2 := &TTY{
		scope: tty.scope,
		label: tty.label,
		enc:   tty.enc,
	}

	// attr copy & extend
	scoped := scopeAttrs(tty.scope, as, tty.enc.sink.replace)
	tty2.attrs = concat(tty.attrs, scoped)

	// attr text copy & extend
	tty2.attrText = make([]byte, len(tty.attrText))
	copy(tty2.attrText, tty.attrText)

	if len(tty2.attrText) > 0 {
		tty2.attrText = append(tty2.attrText, ' ')
	}

	// for consistency, use splicer methods to write attr text
	s := newSplicer()
	defer s.free()

	for i, a := range scoped {
		tty.enc.pushStyle(s, dimStyle)
		s.writeString(a.Key)
		s.writeByte('=')
		tty.enc.popStyle(s)

		tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		tty.enc.popStyle(s)

		if i < len(as)-1 {
			s.writeByte(' ')
		}
	}

	// append attr text
	tty2.attrText = append(tty2.attrText, s.line()...)

	return tty2
}

// See [slog.Handler.WithGroup].
func (tty *TTY) WithGroup(name string) slog.Handler {
	return &TTY{
		attrs:    tty.attrs,
		attrText: tty.attrText,
		scope:    tty.scope + name + ".",
		label:    tty.label,
		enc:      tty.enc,
	}
}

func (tty *TTY) withLabel(label string) handler {
	return &TTY{
		attrs:    tty.attrs,
		attrText: tty.attrText,
		scope:    tty.scope,
		label:    slog.String(labelKey, label),
		enc:      tty.enc,
	}
}

// Handle logs the given [slog.Record] to [TTY] output.
func (tty *TTY) Handle(r slog.Record) error {
	var args []any
	r.Attrs(func(a Attr) {
		args = append(args, a)
	})

	if tty.enc.addSource {
		if file, line := r.SourceLine(); file != "" {
			file += strconv.Itoa(line)
			args = append(args, slog.String("source", file+":"+strconv.Itoa(line)))
		}
	}

	return tty.handle(r.Level, r.Message, nil, -1, args)
}

func (tty *TTY) handle(
	level slog.Level,
	msg string,
	err error,
	depth int,
	args []any,
) error {
	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(tty.scope, tty.attrs, args, tty.enc.sink.replace)

	var sep bool
	for _, field := range tty.enc.fields {
		switch field {
		case ttyTimeField:
			sep = tty.encTime(s, sep)
		case ttyLevelField:
			sep = tty.encLevel(s, level, sep)
		case ttyMessageField:
			sep = tty.encMsg(s, msg, err, sep)
		case ttyAttrsField:
			sep = tty.encAttrs(s, sep)
		case ttySourceField:
			if depth >= 0 {
				tty.encSource(s, depth, sep)
			}
		}
	}

	s.writeByte('\n')

	tty.enc.sink.mu.Lock()
	defer tty.enc.sink.mu.Unlock()

	tty.enc.sink.writeLine(level, s.line())

	return nil
}

func (tty *TTY) fmt(
	msg string,
	err error,
	args []any,
) (string, error) {
	// shortcut: no err, no msg -> return
	if err == nil && len(msg) == 0 {
		return msg, err
	}

	// shortcut: no msg, extant err, no label -> return err
	if err != nil && len(msg) == 0 {
		if tty.label == noLabel {
			return msg, err
		} else {
			err = fmt.Errorf("%s: %w", tty.label.Value.String(), err)
			msg = err.Error()
			return msg, err			
		}
	}

	// interpolate...
	s := newSplicer()
	defer s.free()

	s.join(tty.scope, tty.attrs, s.scan(msg, args), tty.enc.sink.replace)
	s.ipol(msg)

	// err -> return error string, err
	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.line(), err)
		msg = err.Error()
		return msg, err
	}

	// no err -> return msg string, nil
	if len(msg) > 0 {
		msg = s.line()
		return msg, err
	}

	return msg, err
}

// ENCODE

type ttyField int

const (
	ttyTimeField ttyField = iota
	ttyLevelField
	ttyLabelField
	ttyMessageField
	ttyAttrsField
	ttySourceField
)

func encSep(s *splicer, sep bool) {
	if sep {
		s.writeString("  ")
	}
}

func (tty *TTY) encTime(s *splicer, sep bool) bool {
	encSep(s, sep)
	tty.enc.pushStyle(s, dimStyle)
	if tty.enc.elapsed {
		d := time.Since(tty.enc.sink.start).Round(time.Second)
		s.text = appendDuration(s.text, d)
	} else {
		s.writeTimeVerb(time.Now(), tty.enc.timeFormat)
	}
	tty.enc.popStyle(s)
	return sep
}

func (tty *TTY) encLevel(s *splicer, level slog.Level, sep bool) bool {
	// compute padding
	n := len(level.String())

	pad := (12 - n) / 2
	padl := n % 2

	// leftpad
	for i := 0; i < pad + padl - 1; i++ {
		s.writeByte(' ')
	}

	// map level -> style
	var style string
	switch {
	case level < INFO:
		style = debugStyle
	case level < WARN:
		style = infoStyle
	case level < ERROR:
		style = warnStyle
	default:
		style = errStyle
	}

	// encode
	tty.enc.pushStyle(s, style)
	s.writeString(level.String())
	tty.enc.popStyle(s)

	// rightpad
	for i := 0; i < pad; i++ {
		s.writeByte(' ')
	}
	return sep
}

func (tty *TTY) encMsg(s *splicer, msg string, err error, sep bool) bool {
	if tty.label != noLabel {
		sep = tty.encLabel(s, sep)
	}

	if len(msg) == 0 && err == nil {
		return sep
	}

	encSep(s, sep)

	// interpolate message
	tty.enc.pushStyle(s, msgStyle)
	if !s.ipol(msg) {
		s.writeString(msg)
	}
	tty.enc.popStyle(s)

	// merge error into message
	if err != nil {
		if len(msg) > 0 {
			s.writeString(": ")
		}

		tty.enc.pushStyle(s, errStyle)
		s.writeString(err.Error())
		tty.enc.popStyle(s)
	}
	return true
}

// encodes a label, if it exists
func (tty *TTY) encLabel(s *splicer, sep bool) bool {
	tty.enc.pushStyle(s, dimStyle)
	s.writeString(tty.label.Value.String())
	tty.enc.popStyle(s)

	s.writeRune(' ')
	return false
}

func (tty *TTY) encAttrs(s *splicer, sep bool) bool {
	if len(tty.attrText) + len(s.export) == 0 {
		return sep
	}

	encSep(s, sep)

	// write preformatted attr text
	s.Write(tty.attrText)

	// write splicer exports
	space := len(tty.attrText) > 0
	for _, a := range s.export {
		if tty.enc.sink.replace != nil {
			a = tty.enc.sink.replace(a)
		}

		if space {
			s.writeByte(' ')
		}

		if a.Value.Kind() == slog.GroupKind {
			tty.encGroup(s, a.Key, a)
			continue
		}

		switch a.Key {
		case "":
			continue
		case "source":
			defer func() {
				if space {
					s.writeString("  ")
				}

				tty.enc.pushStyle(s, srcStyle)
				s.writeString(a.Value.String())
				tty.enc.popStyle(s)
			}()
			continue
		case "err":
			tty.encAttrErr(s, a)
			space = true
			continue
		}

		tty.enc.pushStyle(s, dimStyle)
		s.writeString(tty.scope)
		s.writeString(a.Key)
		s.writeByte('=')
		tty.enc.popStyle(s)

		tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		tty.enc.popStyle(s)
	}
	return true
}

// encodes an error with error styling
func (tty *TTY) encAttrErr(s *splicer, a Attr) {
	tty.enc.pushStyle(s, dimStyle)
	s.writeString(tty.scope)
	s.writeString(a.Key)
	s.writeByte('=')
	tty.enc.popStyle(s)

	tty.enc.pushStyle(s, errStyle)
	s.writeValueNoVerb(a.Value)
	tty.enc.popStyle(s)
}

// encodes a group with [key=val]-style text
func (tty *TTY) encGroup(s *splicer, name string, a Attr) {
	tty.enc.pushStyle(s, dimStyle)
	s.writeByte('[')
	tty.enc.popStyle(s)

	group := a.Value.Group()
	sep := false
	for _, a := range group {
		if tty.enc.sink.replace != nil {
			a = tty.enc.sink.replace(a)
		}

		if a.Value.Kind() == slog.GroupKind {
			tty.encGroup(s, a.Key, a)
			continue
		}
		
		if a.Key == "" {
			continue
		}

		if sep {
			s.writeByte(' ')
		}

		tty.enc.pushStyle(s, dimStyle)
		s.writeString(name)
		s.writeByte('.')
		s.writeString(a.Key)
		s.writeByte('=')
		tty.enc.popStyle(s)

		tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		tty.enc.popStyle(s)

		sep = true
	}

	tty.enc.pushStyle(s, dimStyle)
	s.writeByte(']')
	tty.enc.popStyle(s)
}

func (tty *TTY) encSource(s *splicer, depth int, sep bool) {
	if !tty.enc.addSource {
		return
	}

	encSep(s, sep)

	tty.enc.pushStyle(s, srcStyle)

	// yank source from runtime
	u := [1]uintptr{}
	runtime.Callers(4+depth, u[:])
	pc := u[0]
	src, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	// encode file/line
	s.writeString(src.File)
	s.writeByte(':')
	s.text = strconv.AppendInt(s.text, int64(src.Line), 10)

	tty.enc.popStyle(s)
}

// STYLES

const (
	labelStyle = "\x1b[37;1m"
	msgStyle   = "\x1b[37;1m"
	attrStyle  = "\x1b[37;1m"
	srcStyle   = "\x1b[37;2m"
	dimStyle   = "\x1b[2m"
	closeStyle = "\x1b[0m"

	debugStyle = "\x1b[34;1m"
	infoStyle  = "\x1b[32;1m"
	warnStyle  = "\x1b[33;1m"
	errStyle   = "\x1b[31;1m"
)

func (enc *ttyEncoder) pushStyle(s *splicer, style string) {
	if !enc.colors {
		return
	}

	s.writeString(style)
}

func (enc *ttyEncoder) popStyle(s *splicer) {
	if !enc.colors {
		return
	}

	s.writeString(closeStyle)
}

// BUFFERS

func (sink *ttySink) writeLine(level slog.Level, line string) {
	if !sink.stream.enabled {
		io.WriteString(sink.w, line)
		return
	}

	sink.stream.insertLine(sink.w, level, line)

	// if too many samples are written, set ref level to ignore more
	if sink.stream.sample.written >= len(sink.stream.sample.lines) {
		sink.ref.Set(sink.stream.holdLevel)
	}
}

func (s *ttyStream) insertLine(w io.Writer, level slog.Level, line string) {
	switch {
	// draw logLevel events to screen
	case level >= s.logLevel:
		s.hold.hide(w)
		s.sample.hide(w)
		io.WriteString(w, line)
		s.hold.show(w)
		s.sample.show(w)
	// otherwise, insert into buffers
	case level >= s.holdLevel:
		s.hold.insertLine(w, line)
	default:
		s.sample.insertLine(w, line)
	}
}

func (s *spinner) insertLine(w io.Writer, line string) {
	if s == nil {
		return
	}

	if len(s.lines) < s.cap {
		s.lines = append(s.lines, line)
	} else {
		s.lines[s.i] = line
	}
	s.i = (s.i + 1) % len(s.lines)
}

func (s *spinner) hide(w io.Writer) {
	if s == nil {
		return
	}

	for i := 0; i < s.written; i++ {
		io.WriteString(w, "\x1b[1A\x1b[2K")
	}
	s.written = 0
}

func (s *spinner) show(w io.Writer) {
	if s == nil {
		return
	}

	if len(s.lines) < s.cap {
		for i := 0; i < s.i; i++ {
			io.WriteString(w, s.lines[i])
			s.written++
		}
		return
	}

	for i := range s.lines {
		i = (i + s.i) % len(s.lines)
		io.WriteString(w, s.lines[i])
		s.written++
	}
}

// update loop refreshes on-screen buffers for each tick of refresh
func (sink *ttySink) update() {
	go func() {
		for {
			select {
			case <-sink.refresh.C:
				sink.mu.Lock()
				sink.stream.hold.hide(sink.w)
				sink.stream.sample.hide(sink.w)
				sink.stream.hold.show(sink.w)
				sink.stream.sample.show(sink.w)
				sink.ref.Set(sink.refBase.Level())
				sink.mu.Unlock()
			case <-sink.done:
				return
			}
		}
	}()
}
