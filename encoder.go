package logf

import (
	"time"

	"log/slog"
	"maps"
)

// ttyFormatter manages state relevant to encoding a record to bytes
type ttyFormatter struct {
	layout []ttyField
	tag    map[string]ttyEncoder[Attr]

	time       ttyEncoder[time.Time]
	level      ttyEncoder[slog.Level]
	message    ttyEncoder[string]
	key        ttyEncoder[string]
	value      ttyEncoder[Value]
	source     ttyEncoder[*slog.Source]
	groupOpen  Encoder[int]
	groupClose Encoder[int]

	groupPen pen
	debugPen pen
	infoPen  pen
	warnPen  pen
	errorPen pen

	addSource bool
}

func newTTYFormatter() *ttyFormatter {
	return &ttyFormatter{
		// layout
		layout: []ttyField{
			ttyLevelField,
			ttyTimeField,
			ttyTagsField,
			ttyMessageField,
			ttyTabField,
			ttyAttrsField,
		},

		// field encodings
		time: ttyEncoder[time.Time]{
			"\x1b[2m",
			EncodeFunc(encTimeShort),
		},
		level: ttyEncoder[slog.Level]{
			"",
			EncodeFunc(encLevelBar),
		},
		message: ttyEncoder[string]{
			"",
			nil,
		},
		key: ttyEncoder[string]{
			"\x1b[36;2m",
			EncodeFunc(encKey),
		},
		value: ttyEncoder[Value]{
			"\x1b[36m",
			EncodeFunc(encValue),
		},
		source: ttyEncoder[*slog.Source]{
			"\x1b[2m",
			EncodeFunc(encSourceAbs),
		},
		groupOpen:  EncodeFunc(encGroupOpen),
		groupClose: EncodeFunc(encGroupClose),

		// level colors
		groupPen: "\x1b[2m",
		debugPen: "\x1b[2m",
		infoPen:  "\x1b[32;1m",
		warnPen:  "\x1b[33;1m",
		errorPen: "\x1b[31;1m",

		// tags
		tag: map[string]ttyEncoder[Attr]{
			"#": {
				"\x1b[35;1m",
				EncodeFunc(encTag),
			},
		},
	}
}

func (fmtr *ttyFormatter) clone(addSource, addColors bool) *ttyFormatter {
	fmtr2 := *fmtr

	// source
	var sourceInLayout bool
	if addSource {
		fmtr2.addSource = true
		for _, f := range fmtr.layout {
			if f == ttySourceField {
				sourceInLayout = true
				break
			}
		}
	}

	if addSource && !sourceInLayout {
		fmtr2.layout = append(fmtr2.layout, ttyNewlineField, ttySourceField)
	}

	// tags
	fmtr2.tag = maps.Clone(fmtr.tag)

	// colors
	if !addColors {
		fmtr2.time.color = ""
		fmtr2.level.color = ""
		fmtr2.message.color = ""
		fmtr2.key.color = ""
		fmtr2.value.color = ""
		fmtr2.source.color = ""

		fmtr2.groupPen = ""
		fmtr2.debugPen = ""
		fmtr2.infoPen = ""
		fmtr2.warnPen = ""
		fmtr2.errorPen = ""

		fmtr2.tag["#"] = ttyEncoder[Attr]{
			"",
			EncodeFunc(encTag),
		}
	}

	return &fmtr2
}

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
//   - source: Encoder[*slog.Source]
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
	src *slog.Source,
) {
	b := &Buffer{s, 0}
	for _, field := range tty.dev.fmtr.layout {
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
	tty.dev.fmtr.time.Encode(b, time.Now())
	b.sep = ' '
}

func (tty *TTY) encLevel(b *Buffer, level slog.Level) {
	b.writeSep()
	p := tty.levelPen(level)
	p.use(b)
	tty.dev.fmtr.level.Encoder.Encode(b, level)
	p.drop(b)
	b.sep = 0
}

func (tty *TTY) encMsg(b *Buffer, msg string, err error) {
	if len(msg) == 0 && err == nil {
		return
	}

	b.writeSep()

	tty.dev.fmtr.message.color.use(b)
	b.splicer.WriteString(msg)
	tty.dev.fmtr.message.color.drop(b)

	// merge error into message
	if err != nil {
		if len(msg) > 0 {
			b.WriteString(": ")
		}

		tty.dev.fmtr.errorPen.use(b)
		b.WriteString(err.Error())
		tty.dev.fmtr.errorPen.drop(b)
	}

	b.sep = ' '
}

func (tty *TTY) encAttr(b *Buffer, a Attr) {
	if a.Key == "" {
		return
	}

	if a.Value.Kind() == slog.KindLogValuer {
		if lv, ok := a.Value.Any().(slog.LogValuer); ok {
			a.Value = lv.LogValue().Resolve()
		}
	}

	if a.Value.Kind() == slog.KindGroup {
		tty.encAttrGroup(b, a)
		return
	}

	b.writeSep()
	tty.dev.fmtr.key.Encode(b, a.Key)
	tty.dev.fmtr.value.Encode(b, a.Value)
	b.sep = ' '
}

func (tty *TTY) encTag(b *Buffer, a Attr) {
	if a.Value.Kind() == slog.KindLogValuer {
		a.Value = a.Value.Resolve()
	}

	if a.Value.Kind() == slog.KindGroup {
		tty.encTagGroup(b, a.Key, a)
		return
	}

	var tag Encoder[Attr]
	var found bool
	if tag, found = tty.dev.fmtr.tag[a.Key]; !found {
		return
	}

	b.writeSep()
	tag.Encode(b, a)
	b.sep = ' '
}

func (tty *TTY) encSource(b *Buffer, src *slog.Source) {
	if !tty.dev.fmtr.addSource {
		return
	}

	b.writeSep()
	tty.dev.fmtr.source.Encode(b, src)
	b.sep = ' '
}

// LISTS

func (tty *TTY) encExportAttrs(b *Buffer) {
	if len(tty.attrText)+len(b.splicer.export) == 0 {
		return
	}

	if len(tty.attrText) > 0 {
		b.writeSep()
		b.WriteString(tty.attrText)
		b.sep = tty.attrSep
	}

	if len(b.splicer.export) > 0 {
		tty.encListAttrs(b, b.splicer.export)
		b.sep = ' '
	}

	if len(tty.store.scope) > 0 {
		tty.encAttrGroupClose(b, len(tty.store.scope))
	}
}

func (tty *TTY) encListAttrs(b *Buffer, as []Attr) {
	for _, a := range as {
		if tty.dev.replace != nil {
			a = tty.dev.replace(nil, a)
		}

		if a.Key == "source" {
			defer func() {
				b.writeSep()
				tty.dev.fmtr.source.color.use(b)
				b.WriteValue(a.Value, nil)
				tty.dev.fmtr.source.color.drop(b)
			}()
			continue
		}

		tty.encAttr(b, a)
	}
}

func (tty *TTY) encExportTags(b *Buffer) {
	if tty.label.Key == "#" {
		b.writeSep()
		tty.dev.fmtr.tag["#"].Encode(b, tty.label)
		b.sep = ' '
	}

	if len(tty.tagText) > 0 {
		b.writeSep()
		b.WriteString(tty.tagText)
		b.sep = ' '
	}

	if len(b.splicer.export) > 0 {
		tty.encListTags(b, b.splicer.export)
	}
}

func (tty *TTY) encListTags(b *Buffer, as []Attr) {
	for _, a := range as {
		if tty.dev.replace != nil {
			a = tty.dev.replace(nil, a)
		}

		if a.Key == "source" {
			defer func() {
				b.writeSep()
				tty.dev.fmtr.source.color.use(b)
				b.WriteValue(a.Value, nil)
				tty.dev.fmtr.source.color.drop(b)
			}()
			continue
		}

		tty.encTag(b, a)
	}
}

// GROUPS

// encodes a group with [key=val]-style text
func (tty *TTY) encAttrGroup(b *Buffer, a Attr) {
	b.writeSep()
	b.sep = 0

	tty.dev.fmtr.key.color.use(b)
	tty.dev.fmtr.key.Encode(b, a.Key)
	tty.dev.fmtr.key.color.drop(b)

	tty.encAttrGroupOpen(b)
	group := a.Value.Group()
	tty.encListAttrs(b, group)
	tty.encAttrGroupClose(b, 1)
}

func (tty *TTY) encAttrGroupOpen(b *Buffer) {
	b.writeSep()

	tty.dev.fmtr.groupPen.use(b)
	tty.dev.fmtr.groupOpen.Encode(b, 0)
	tty.dev.fmtr.groupPen.drop(b)

	b.sep = 0
}

func (tty *TTY) encAttrGroupClose(b *Buffer, count int) {
	tty.dev.fmtr.groupPen.use(b)
	tty.dev.fmtr.groupClose.Encode(b, count)
	tty.dev.fmtr.groupPen.drop(b)

	b.sep = '?'
}

func (tty *TTY) encTagGroup(b *Buffer, scope string, a Attr) {
	group := a.Value.Group()
	for _, a := range group {
		tty.encTag(b, a)
	}
}
