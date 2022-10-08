package logf

import (
	"context"
	"reflect"
	"sync"
	"unsafe"

	"golang.org/x/exp/slog"
)

type splicer struct {
	text
	dict
	list
}

func newSplicer() *splicer {
	s := spool.Get().(*splicer)

	return s
}

var spool = sync.Pool{
	New: func() any {
		return &splicer{
			text: make(text, 0, 1024),
			dict: make(dict, 5),
			list: list{make([]any, 0, 5), 0, 0, 0},
		}
	},
}

func (s *splicer) free() {
	const maxTextSize = 16 << 10
	const maxAttrSize = 128

	ok := cap(s.text) < maxTextSize
	ok = ok && (len(s.dict)+cap(s.list.args)) < maxAttrSize

	if ok {
		s.text = s.text[:0]
		s.dict.clear()
		s.list.clear()

		spool.Put(s)
	}
}

// after acquired from pool, use join to load Attrs into a splicer
func (s *splicer) join(ctx context.Context, seg []Attr, args []any) {
	// reset list index
	s.list.i = 0

	// seg
	// no list insert, because segment is exported by Handler
	for _, a := range seg {
		s.dict[a.Key] = a.Value
	}

	// ctx
	if ctx != nil {
		if as, ok := ctx.Value(segmentKey{}).([]Attr); ok {
			for _, a := range as {
				s.dict[a.Key] = a.Value
				s.list.insert(a)
			}
		}
	}

	// args
	// no dictionry insert, args must interpolate with "{}"" tokens
	for len(args) > 0 {
		s.list.insert(args[0])
		args = args[1:]
	}
}

func (s *splicer) freeze() (msg string) {
	return string(s.text)
}

// after interpolation, freeze unsafely yields a string containing an interpolated message.
// It is catastrophically bad to read the string after free has been called.
func (s *splicer) freezeUnsafe() (msg string) {
	textHeader := (*reflect.SliceHeader)(unsafe.Pointer(&s.text))
	msgHeader := (*reflect.StringHeader)(unsafe.Pointer(&msg))
	msgHeader.Data, msgHeader.Len = textHeader.Data, textHeader.Len
	return
}

// DICT / LIST

type dict map[string]slog.Value

func (d dict) clear() {
	for k := range d {
		delete(d, k)
	}
}

// keeps index in i, elements in as
type list struct {
	args                  []any
	i, attrBegin, attrEnd int
}

func (l *list) insert(arg any) {
	if l.i < len(l.args) {
		l.args[l.i] = arg
	} else {
		l.args = append(l.args, arg)
	}
	l.i++
}

func (l *list) nextArg() (arg any, ok bool) {
	if l.i >= len(l.args) {
		return
	}

	arg, ok = l.args[l.i], true
	l.i++
	return
}

func (l *list) parseAttrs() (ok bool) {
	l.attrBegin, l.attrEnd = l.i, l.i

	for l.i < len(l.args) {
		switch arg := l.args[l.i].(type) {
		case string:
			if len(l.args[l.i:]) == 1 {
				l.args[l.attrEnd] = slog.String(missingArg, arg)
				l.attrEnd++
				return false
			}
			l.args[l.attrEnd] = slog.Any(arg, l.args[l.i+1])
			l.attrEnd++
			l.i += 2
		case Attr:
			l.args[l.attrEnd] = arg
			l.attrEnd++
			l.i++
		default:
			l.args[l.attrEnd] = slog.Any(missingKey, arg)
			l.attrEnd++
			l.i++
		}
	}
	return true
}

func (l *list) export(r *slog.Record) {
	for i := l.attrBegin; i < l.attrEnd; i++ {
		r.AddAttrs(l.args[i].(Attr))
	}
}

func (l *list) clear() {
	for i := range l.args {
		l.args[i] = any(nil)
	}
	l.args = l.args[:0]
	l.i, l.attrBegin, l.attrEnd = 0, 0, 0
}

// INTERPOLATE

func (s *splicer) interpolate(msg string) {
	s.list.i = 0

	// interpolation loop
	var clip []byte
	var ok bool
	for {
		if msg, clip, ok = s.text.scanKey(msg); !ok {
			break
		}
		s.interpAttr(clip)
	}

	// remaing args -> exports
	s.parseAttrs()
}

func (s *splicer) interpAttr(clip []byte) {
	key, verb := splitVerb(clip)

	if len(key) == 0 {
		s.interpUnkeyed(verb)
	} else {
		s.interpKeyed(key, verb)
	}
}

func (s *splicer) interpUnkeyed(verb []byte) {
	arg, ok := s.list.nextArg()
	if !ok {
		s.text.appendString(missingArg)
		return
	}
	if a, isAttr := arg.(Attr); isAttr {
		s.text.appendValue(a.Value, verb)
		return
	}

	s.text.appendArg(arg, verb)
}

func (s *splicer) interpKeyed(key, verb []byte) {
	switch string(key) {
	case "time":
		// s.text.appendTimeNow(verb)
		return
	case "level":
		// TODO
		return
	case "source":
		// TODO
		return
	}

	v, ok := s.dict[string(key)]
	if !ok {
		s.text.appendString(missingKey)
		return
	}

	s.text.appendValue(v, verb)
}
