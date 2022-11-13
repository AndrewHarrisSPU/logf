package logf

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

type ttyField int

const (
	ttyTimeField ttyField = iota
	ttyLevelField
	ttyLabelField
	ttyMessageField
	ttyAttrsField
	ttySourceField
)

// TTY HANDLER

type TTY struct {
	level    slog.Leveler
	seg      []Attr
	attrText []byte
	labels   string
	label    string
	enc      *ttyEncoder
}

// Logger returns a [Logger] that uses the TTY as a handler.
func (tty *TTY) Logger() Logger {
	return Logger{h: tty}
}

// Printer returns a Logger using a near-clone of the TTY configuration.
// Notably, the printer only emits label and message fields.
func (tty *TTY) Printer() Logger {
	return Logger{
		h: &TTY{
			level:    tty.level,
			seg:      tty.seg,
			attrText: tty.attrText,
			labels:   tty.labels,
			label:    tty.label,
			enc: &ttyEncoder{
				w:          tty.enc.w,
				mu:         tty.enc.mu,
				colors:     tty.enc.colors,
				addLabel:   true,
				timeFormat: tty.enc.timeFormat,
				start:      tty.enc.start,
				layout:     []ttyField{ttyMessageField},
				spin: spinner{
					enabled: false,
					level:   tty.level,
					cap:     0,
				},
			},
		},
	}
}

// Enabled reports whether the TTY is enabled for the given level.
// If the TTY uses a spin buffer (see [Config.Spin]), the TTY is enabled at or above the spin buffer's level.
// Otherwise, the TTY is enabled at or above the TTY's level.
func (tty *TTY) Enabled(level slog.Level) bool {
	return tty.enabled(level)
}

func (tty *TTY) enabled(level slog.Level) bool {
	if !tty.enc.spin.enabled {
		return level >= tty.level.Level()
	}
	return level >= tty.enc.spin.level.Level()
}

// WithAttrs adds Attrs to the TTY.
// See also: [slog.WithAttrs]
func (tty *TTY) WithAttrs(as []Attr) slog.Handler {
	return tty.withAttrs(as).(*TTY)
}

func (tty *TTY) withAttrs(as []Attr) handler {
	tty2 := new(TTY)

	tty2.level = tty.level
	tty2.labels = tty.labels
	tty2.label = tty.label
	tty2.enc = tty.enc

	// attr text copy & extend
	tty2.attrText = make([]byte, len(tty.attrText))
	copy(tty2.attrText, tty.attrText)

	if len(tty2.attrText) > 0 {
		tty2.attrText = append(tty2.attrText, ' ')
	}

	s := newSplicer()
	defer s.free()

	for i, a := range as {
		if tty.enc.replace != nil {
			a = tty.enc.replace(a)
		}

		pop := tty.enc.pushStyle(s, dimStyle)
		s.writeString(tty.labels)
		s.writeString(a.Key)
		s.writeByte('=')
		pop()

		pop = tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		pop()

		if i < len(as)-1 {
			s.writeByte(' ')
		}
	}

	tty2.attrText = append(tty2.attrText, s.line()...)

	// segment copy & extend
	scoped := scopeSegment(tty.labels, as)
	tty2.seg = concat(tty.seg, scoped)

	return tty2
}

// WithGroup opens a new group of Attrs associated with the TTY.
// Attrs appended to the TTY or logged later are members of this group.
// See also: [slog.Handler.WithGroup]
func (tty *TTY) WithGroup(name string) slog.Handler {
	return tty.withGroup(name).(*TTY)
}

func (tty *TTY) withGroup(label string) handler {
	return &TTY{
		level:    tty.level,
		seg:      tty.seg,
		attrText: tty.attrText,
		labels:   tty.labels + label + ".",
		label:    label,
		enc:      tty.enc,
	}
}

func (tty *TTY) Handle(r slog.Record) error {
	var args []any
	r.Attrs(func(a Attr) {
		args = append(args, a)
	})

	if file, line := r.SourceLine(); file != "" {
		file += strconv.Itoa(line)
		args = append(args, slog.String("source", file+":"+strconv.Itoa(line)))
	}

	return tty.handle(r.Level, r.Message, nil, 0, args)
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

	s.replace = tty.enc.replace
	args = s.scan(msg, args)
	s.join(tty.labels, tty.seg, args)

	var sep bool
	for _, field := range tty.enc.layout {
		if sep {
			s.writeString("  ")
		}

		switch field {
		case ttyTimeField:
			tty.encTime(s)
			sep = true
		case ttyLevelField:
			tty.encLevel(s, level)
			sep = true
		case ttyMessageField:
			tty.encMsg(s, msg, err)
			sep = len(msg) > 0 || err != nil
		case ttyAttrsField:
			if len(tty.attrText)+len(s.export) == 0 {
				sep = false
			} else {
				tty.encAttrs(s)
				sep = true
			}
		case ttySourceField:
			tty.encSource(s, depth)
			sep = true
		}
	}

	s.writeByte('\n')

	tty.enc.mu.Lock()
	defer tty.enc.mu.Unlock()

	tty.enc.writeLine(level, tty.level.Level(), s.line())

	return nil
}

func (tty *TTY) fmt(
	msg string,
	err error,
	args []any,
) (string, error) {
	s := newSplicer()
	defer s.free()

	s.replace = tty.enc.replace
	s.join(tty.labels, tty.seg, s.scan(msg, args))
	s.ipol(msg)

	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.line(), err)
		msg = err.Error()
	} else {
		msg = s.line()
	}

	return msg, err
}

func (tty *TTY) LogValue() slog.Value {
	return slog.GroupValue(tty.seg...)
}

// TTY ENCODER

type ttyEncoder struct {
	// writer
	w  io.Writer
	mu *sync.Mutex

	// encoding
	layout     []ttyField
	spin       spinner
	timeFormat string
	start      time.Time
	replace    func(Attr) Attr

	elapsed   bool
	addLabel  bool
	colors    bool
	addSource bool
}

// SPIN

type spinner struct {
	level      slog.Leveler
	lines      []string
	cap        int
	i, written int
	enabled    bool
}

func (tty *TTY) Write(p []byte) (n int, err error) {
	tty.enc.mu.Lock()
	defer tty.enc.mu.Unlock()

	// trim encoder feed
	if tty.enc.spin.enabled {
		tty.enc.spin.clear(tty.enc.w)
		tty.enc.spin.lines = tty.enc.spin.lines[:0]
		tty.enc.spin.i = 0
	}

	return tty.enc.w.Write(p)
}

func (enc *ttyEncoder) writeLine(level slog.Level, ref slog.Level, line string) {
	if level >= ref {
		enc.spin.clear(enc.w)
		io.WriteString(enc.w, line)
		enc.spin.show(enc.w)
	} else {
		enc.spin.insertLine(enc.w, line)
	}
}

func (s *spinner) insertLine(w io.Writer, line string) {
	if len(s.lines) < s.cap {
		s.lines = append(s.lines, line)
	} else {
		s.lines[s.i] = line
	}
	s.i = (s.i + 1) % len(s.lines)

	s.clear(w)
	s.show(w)
}

func (s *spinner) clear(w io.Writer) {
	for i := 0; i < s.written; i++ {
		io.WriteString(w, "\x1b[1A\x1b[2K")
	}
	s.written = 0
}

func (s *spinner) show(w io.Writer) {
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

func (enc *ttyEncoder) pushStyle(s *splicer, style string) func() {
	if !enc.colors {
		return func() {}
	}

	s.writeString(style)
	return func() {
		s.writeString(closeStyle)
	}
}

// ENCODE

func (tty *TTY) encTime(s *splicer) {
	pop := tty.enc.pushStyle(s, dimStyle)
	if tty.enc.elapsed {
		d := time.Since(tty.enc.start).Round(time.Second)
		s.text = appendDuration(s.text, d)
	} else {
		s.writeTimeVerb(time.Now(), tty.enc.timeFormat)
	}
	pop()
}

func (tty *TTY) encLevel(s *splicer, level slog.Level) {
	// compute padding
	n := len(level.String())
	padl := (7 - n) / 2
	padr := (8 - n) / 2

	// leftpad
	for i := 0; i < padl; i++ {
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
	pop := tty.enc.pushStyle(s, style)
	s.writeString(level.String())
	pop()

	// rightpad
	for i := 0; i < padr; i++ {
		s.writeByte(' ')
	}
}

func (tty *TTY) encMsg(s *splicer, msg string, err error) {
	// if configured, add label
	if tty.enc.addLabel {
		tty.encLabel(s)
	}

	// interpolate message
	pop := tty.enc.pushStyle(s, msgStyle)
	s.ipol(msg)
	pop()

	// merge error into message
	if err != nil {
		if len(msg) > 0 {
			s.writeString(": ")
		}
		pop := tty.enc.pushStyle(s, errStyle)
		s.writeString(err.Error())
		pop()
	}
}

func (tty *TTY) encLabel(s *splicer) {
	if len(tty.label) == 0 {
		return
	}

	pop := tty.enc.pushStyle(s, dimStyle)
	s.writeString(tty.label)
	pop()

	s.writeString("  ")
}

func (tty *TTY) encAttrs(s *splicer) {
	// write preformatted attr text
	s.Write(tty.attrText)

	// write splicer exports
	sep := len(tty.attrText) > 0
	for _, a := range s.export {
		if tty.enc.replace != nil {
			a = tty.enc.replace(a)
		}

		if len(a.Key) == 0 {
			continue
		}

		if sep {
			s.writeByte(' ')
		}

		if a.Value.Kind() == slog.GroupKind {
			tty.encGroup(s, a.Key, a)
			continue
		}

		pop := tty.enc.pushStyle(s, dimStyle)
		s.writeString(tty.labels)
		s.writeString(a.Key)
		s.writeByte('=')
		pop()

		pop = tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		pop()

		sep = true
	}

	return
}

func (tty *TTY) encGroup(s *splicer, name string, a Attr) {
	pop := tty.enc.pushStyle(s, dimStyle)
	s.writeByte('[')
	pop()

	group := a.Value.Group()
	sep := false
	for _, a := range group {
		if tty.enc.replace != nil {
			a = tty.enc.replace(a)
		}

		if a.Value.Kind() == slog.GroupKind {
			tty.encGroup(s, a.Key, a)
			continue
		}

		if sep {
			s.writeByte(' ')
		}

		pop := tty.enc.pushStyle(s, dimStyle)
		s.writeString(name)
		s.writeByte('.')
		s.writeString(a.Key)
		s.writeByte('=')
		pop()

		pop = tty.enc.pushStyle(s, attrStyle)
		s.writeValueNoVerb(a.Value)
		pop()
		sep = true
	}
	pop = tty.enc.pushStyle(s, dimStyle)
	s.writeByte(']')
	pop()
}

func (tty *TTY) encSource(s *splicer, depth int) {
	pop := tty.enc.pushStyle(s, srcStyle)

	// yank source from runtime
	u := [1]uintptr{}
	runtime.Callers(4+depth, u[:])
	pc := u[0]
	src, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	// encode file/line
	s.writeString(src.File)
	s.writeByte(':')
	s.text = strconv.AppendInt(s.text, int64(src.Line), 10)

	pop()
}
