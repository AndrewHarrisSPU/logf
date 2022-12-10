package logf

import (
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

// ENCODERS

// Encoder writes values of type T to a [Buffer] containing a [TTY] log line.
//
// Flavors of Encoder expected by [TTY] encoding:
//   - time: Encoder[time.Time]
//   - level: Encoder[slog.Level]
//   - message: Encdoer[string]
//   - tag: Encoder[Attr]
//   - attr key: Encoder[string]
//   - attr value: Encoder[Value]
//   - source: Encoder[SourceLine]
type Encoder[T any] interface {
	Encode(*Buffer, T)
}

// EncodeFunc returns a T-flavored [Encoder] from a compatible function.
func EncodeFunc[T any](fn func(*Buffer, T)) Encoder[T] {
	return encFunc[T](fn)
}

type encFunc[T any] func(*Buffer, T)

func (fn encFunc[T]) Encode(b *Buffer, t T) {
	fn(b, t)
}

type ttyEncoder[T any] struct {
	color pen
	Encoder[T]
}

func (enc ttyEncoder[T]) Encode(b *Buffer, t T) {
	enc.color.use(b)
	enc.Encoder.Encode(b, t)
	enc.color.drop(b)
}

// Buffer offers an [Encoder] a way to write to a pooled resource when building [TTY] log lines.
// A Buffer is writable during [TTY] field encoding, and is invalid otherwise.
// It is not safe to store a Buffer outside of usage in [EncodeFunc], and a Buffer is not safe for use in go routines.
type Buffer struct {
	*splicer
	sep byte
}

const sepstring = "        "
const tabWidth = 3

func (b *Buffer) writeSep() {
	switch b.sep {
	case 0:
	case ' ':
		b.WriteByte(' ')
	case '\n':
		b.WriteByte('\n')
	case '\t':
		b.WriteByte('\t')
	case '?':
		b.WriteByte(' ')
	case '!':
		b.WriteByte('!')
	}
}

// TTY FIELD ENCODING

type ttyField int

const (
	ttyTimeField ttyField = iota
	ttyLevelField
	ttyMessageField
	ttyAttrsField
	ttyTagsField
	ttySourceField

	ttyNewlineField
	ttySpaceField
	ttyTabField
)

func (tty *TTY) encFields(
	s *splicer,
	level slog.Level,
	msg string,
	err error,
	src SourceLine,
) {
	b := &Buffer{s, 0}
	for _, field := range tty.fmtr.layout {
		switch field {
		case ttyTimeField:
			tty.encTime(b)
		case ttyLevelField:
			tty.encLevel(b, level)
		case ttyMessageField:
			tty.encMsg(b, msg, err)
		case ttyAttrsField:
			tty.encExportAttrs(b)
		case ttyTagsField:
			tty.encExportTags(b)
		case ttySourceField:
			tty.encSource(b, src)
		case ttyNewlineField:
			b.sep = '\n'
			b.writeSep()
			b.sep = '\t'
			b.writeSep()
			b.sep = 0
		case ttySpaceField:
			if b.sep != 0 {
				b.sep = ' '
			}
		case ttyTabField:
			if b.sep != 0 {
				b.sep = '\t'
			}
		}
	}
	b.splicer = nil

	s.WriteByte('\n')
}

func (tty *TTY) encTime(b *Buffer) {
	b.writeSep()
	tty.fmtr.time.Encode(b, time.Now())
	b.sep = ' '
}

func (tty *TTY) encLevel(b *Buffer, level slog.Level) {
	b.writeSep()
	p := tty.levelPen(level)
	p.use(b)
	tty.fmtr.level.Encoder.Encode(b, level)
	p.drop(b)
	b.sep = 0
}

func (tty *TTY) encMsg(b *Buffer, msg string, err error) {
	if len(msg) == 0 && err == nil {
		return
	}

	b.writeSep()

	tty.fmtr.message.color.use(b)
	b.splicer.WriteString(msg)
	tty.fmtr.message.color.drop(b)

	// merge error into message
	if err != nil {
		if len(msg) > 0 {
			b.WriteString(": ")
		}

		tty.fmtr.errorPen.use(b)
		b.WriteString(err.Error())
		tty.fmtr.errorPen.drop(b)
	}

	b.sep = ' '
}

func (tty *TTY) encAttr(b *Buffer, scope string, a Attr) {
	if a.Key == "" {
		return
	}

	if a.Value.Kind() == slog.LogValuerKind {
		if lv, ok := a.Value.Any().(slog.LogValuer); ok {
			a.Value = lv.LogValue().Resolve()
		}
	}

	if a.Value.Kind() == slog.GroupKind {
		tty.encAttrGroup(b, scope, a)
		return
	}

	b.writeSep()
	tty.fmtr.key.Encode(b, a.Key)
	tty.fmtr.value.Encode(b, a.Value)
	b.sep = ' '
}

func (tty *TTY) encTag(b *Buffer, scope string, a Attr) {
	if a.Value.Kind() == slog.GroupKind {
		tty.encTagGroup(b, scope, a)
		return
	}

	var tag Encoder[Attr]
	var found bool
	if tag, found = tty.fmtr.tag[a.Key]; !found {
		return
	}

	b.writeSep()
	tag.Encode(b, a)
	b.sep = ' '
}

func (tty *TTY) encSource(b *Buffer, src SourceLine) {
	if !tty.fmtr.addSource {
		return
	}

	b.writeSep()
	tty.fmtr.source.Encode(b, src)
	b.sep = ' '
}

func (tty *TTY) yankSourceLine(depth int) SourceLine {
	if !tty.fmtr.addSource {
		return SourceLine{}
	}

	// yank source from runtime
	u := [1]uintptr{}
	runtime.Callers(4+depth, u[:])
	pc := u[0]
	src, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	return SourceLine{src.File, src.Line}
}

// LISTS

func (tty *TTY) encExportAttrs(b *Buffer) {
	if len(tty.attrText)+len(b.splicer.export) == 0 {
		return
	}

	if len(tty.attrText) > 0 {
		b.writeSep()
		b.WriteString(tty.attrText)
		b.sep = ' '
	}

	if len(b.splicer.export) > 0 {
		tty.encListAttrs(b, tty.openKey, b.splicer.export)
		b.sep = ' '
	}

	if tty.nOpen > 0 {
		tty.encAttrGroupClose(b, tty.nOpen)
	}
}

func (tty *TTY) encListAttrs(b *Buffer, scope string, as []Attr) {
	for _, a := range as {
		if tty.fmtr.sink.replace != nil {
			a = tty.fmtr.sink.replace(a)
		}

		if a.Key == "source" {
			defer func() {
				b.writeSep()
				tty.fmtr.source.color.use(b)
				b.WriteValue(a.Value, nil)
				tty.fmtr.source.color.drop(b)
			}()
			continue
		}

		tty.encAttr(b, scope, a)
	}
}

func (tty *TTY) encExportTags(b *Buffer) {
	if tty.tag.Key == "#" {
		b.writeSep()
		tty.fmtr.tag["#"].Encode(b, tty.tag)
		b.sep = ' '
	}

	if len(tty.tagText) > 0 {
		b.writeSep()
		b.WriteString(tty.tagText)
		b.sep = ' '
	}

	if len(b.splicer.export) > 0 {
		tty.encListTags(b, tty.scope, b.splicer.export)
	}
}

func (tty *TTY) encListTags(b *Buffer, scope string, as []Attr) {
	for _, a := range as {
		if tty.fmtr.sink.replace != nil {
			a = tty.fmtr.sink.replace(a)
		}

		if a.Key == "source" {
			defer func() {
				b.writeSep()
				tty.fmtr.source.color.use(b)
				b.WriteValue(a.Value, nil)
				tty.fmtr.source.color.drop(b)
			}()
			continue
		}

		tty.encTag(b, scope, a)
	}
}

// GROUPS

// encodes a group with [key=val]-style text
func (tty *TTY) encAttrGroup(b *Buffer, scope string, a Attr) {
	group := a.Value.Group()
	if len(group) == 0 {
		return
	}

	b.writeSep()
	b.sep = 0

	tty.fmtr.key.color.use(b)
	tty.fmtr.key.Encode(b, a.Key)
	tty.fmtr.key.color.drop(b)

	tty.encAttrGroupOpen(b, scope)
	tty.encListAttrs(b, a.Key, group)
	tty.encAttrGroupClose(b, 1)
}

func (tty *TTY) encAttrGroupOpen(b *Buffer, scope string) {
	b.writeSep()

	tty.fmtr.groupPen.use(b)
	tty.fmtr.groupOpen.Encode(b, struct{}{})
	tty.fmtr.groupPen.drop(b)

	b.sep = 0
}

func (tty *TTY) encAttrGroupClose(b *Buffer, count int) {
	tty.fmtr.groupPen.use(b)
	tty.fmtr.groupClose.Encode(b, count)
	tty.fmtr.groupPen.drop(b)

	b.sep = '?'
}

func (tty *TTY) encTagGroup(b *Buffer, scope string, a Attr) {
	group := a.Value.Group()
	tty.encListTags(b, a.Key, group)
}
