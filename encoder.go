package logf

import (
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

// ENCODERS

type Encoder[T any] interface {
	Encode(*Buffer, T)
}

func EncodeFunc[T any](fn func(*Buffer, T)) Encoder[T] {
	return encFunc[T](fn)
}

type encFunc[T any] func(*Buffer, T)

func (fn encFunc[T]) Encode(b *Buffer, t T){
	fn(b, t)
}

type ttyEncoder[T any] struct {
	color pen
	Encoder[T]
}

func (enc ttyEncoder[T]) Encode(b *Buffer, t T){
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
		tab := tabWidth - (len(b.splicer.text) % tabWidth)
		b.WriteString(sepstring[:tab])
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
	ttySourceField

	ttyTagsField
	ttyNewlineField
	ttySpaceField
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
		case ttySpaceField:
			b.writeSep()
			b.WriteString("  ")
			b.sep = 0
		}
	}
	b.splicer = nil

	s.WriteByte('\n')
}

func (tty *TTY) encTime(b *Buffer){
	b.writeSep()
	tty.fmtr.time.Encode(b, time.Now())
	b.sep = ' '
}

func (tty *TTY) encLevel(b *Buffer, level slog.Level){
	b.writeSep()
	p := tty.levelPen(level)
	p.use(b)
	tty.fmtr.level.Encoder.Encode(b, level)
	p.drop(b)
	b.sep = ' '
}

func (tty *TTY) encMsg(b *Buffer, msg string, err error){
	if len(msg) == 0 && err == nil {
		return
	}

	b.writeSep()

	tty.fmtr.message.color.use(b)
	if !b.splicer.ipol(msg) {
		b.WriteString(msg)
	}
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

	b.sep = '\t'
}

func (tty *TTY) encAttr(b *Buffer, scope string, a Attr){
	if a.Key == "" {
		return
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

func (tty *TTY) encTag(b *Buffer, scope string, a Attr){
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

func (tty *TTY) encSource(b *Buffer, src SourceLine){
	if !tty.fmtr.addSource {
		return
	}

	b.writeSep()
	tty.fmtr.source.Encode(b, src)
	b.sep = '\t'
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

const (
	listAttrs bool = false
	listTags  bool = true
)

func (tty *TTY) encList(b *Buffer, scope string, as []Attr, mode bool){
	// if mode == listAttrs && scope != "" {
	// 	tty.encAttrGroupOpen(b, scope)
	// }

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

		switch mode {
		case listAttrs:
			tty.encAttr(b, scope, a)
		case listTags:
			tty.encTag(b, scope, a)
		}
	}

	b.sep = ' '
}

func (tty *TTY) encExportAttrs(b *Buffer){
	if len(tty.attrText)+len(b.splicer.export) == 0 {
		return
	}

	b.writeSep()
	b.WriteString(tty.attrText)
	b.sep = ' '

	tty.encList(b, tty.openKey, b.splicer.export, false)
	tty.encAttrGroupClose(b, tty.nOpen)

	return
}

func (tty *TTY) encExportTags(b *Buffer){
	// i. tty label
	if tag, found := tty.fmtr.tag["#"]; found {
		b.writeSep()
		tag.Encode(b, tty.tag)
		b.sep = ' '
	}

	if len(tty.tagText)+len(b.splicer.export) == 0 {
		return
	}

	// ii. tty pre-foramtted tags
	b.writeSep()
	b.WriteString(tty.tagText)
	b.sep = ' '

	// iii. splicer exports
	tty.encList(b, tty.scope, b.splicer.export, listTags)
}

// GROUPS

// encodes a group with [key=val]-style text
func (tty *TTY) encAttrGroup(b *Buffer, scope string, a Attr){
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
	tty.encList(b, a.Key, group, listAttrs)
	tty.encAttrGroupClose(b, 1)
	return
}

func (tty *TTY) encAttrGroupOpen(b *Buffer, scope string){
	b.writeSep()

	tty.fmtr.groupPen.use(b)
	tty.fmtr.groupOpen.Encode(b, "")
	tty.fmtr.groupPen.drop(b)

	// tty.fmtr.key.color.use(b)
	// b.WriteString(scope)
	// tty.fmtr.key.color.drop(b)

	b.sep = 0
	return
}

func (tty *TTY) encAttrGroupClose(b *Buffer, count int){
	tty.fmtr.groupPen.use(b)
	tty.fmtr.groupClose.Encode(b, count)
	tty.fmtr.groupPen.drop(b)

	b.sep = '?'
}

func (tty *TTY) encTagGroup(b *Buffer, scope string, a Attr){
	group := a.Value.Group()
	tty.encList(b, a.Key, group, listTags)
}
