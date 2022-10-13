package logf

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/exp/slog"
)

type splicer struct {
	// final spliced output is written to text
	text

 	// holds parts of interpolated message that need escaping
	unescaped []byte

	// holds map of keyed interpolation symbols
	dict

	// holds list of unkeyed arguments
	list []any 

	// holds ordered list of exported attrs
	export []Attr
}

func newSplicer() *splicer {
	return spool.Get().(*splicer)
}

var spool = sync.Pool{
	New: func() any {
		return &splicer{
			text: make(text, 0, 1024),
			unescaped: make([]byte, 0, 1024),
			dict: make(dict, 5),
			list: make([]any, 0, 5),
			export: make([]Attr, 0, 5),
		}
	},
}

func (s *splicer) free() {
	const maxTextSize = 16 << 10
	const maxAttrSize = 128

	ok := cap(s.text) < maxTextSize
	ok = ok && (len(s.dict)+cap(s.list)+cap(s.export)) < maxAttrSize

	if ok {
		s.dict.clear()

		// TODO: clear smarter
		s.unescaped = s.unescaped[:0]
		s.list = s.list[:0]
		s.export = s.export[:0]

		spool.Put(s)
	}
}

func (s *splicer) scan(msg string, args []any) []any {
	var clip string
	var found bool
	var unkeyed int
	for {
		msg, clip, found = scanKey(msg)
		if !found {
			break
		}

		key := s.scanSplitKey(clip)
		if len(key) > 0 {
			key = s.unescape(key)
			s.dict.prematch(key)
		} else {
			unkeyed++
		}
	}

	for i := 0; i < unkeyed; i++ {
		if len(args) == 0 {
			s.list = append(s.list, missingArg)
			continue
		}
		s.list = append(s.list, args[0])
		args = args[1:]
	}

	return args
}

// TODO: micro-optimizing allocs etc. here could be possible.
// putting it off for now.
func (s *splicer) unescape(key string)(ukey string ){
	// TODO: is this worth it?
	if !strings.ContainsRune(key, '\\' ){
		return key
	}

	lpos := len(s.unescaped)
	var esc bool
	for _, r := range key {
		if r == '\\' && !esc {
			esc = true
			continue
		}
		esc = false
		s.unescaped = utf8.AppendRune(s.unescaped, r)
	}
	rpos := len(s.unescaped)

	return string(s.unescaped[lpos:rpos])

	// TODO: this hould be safe

	// u := s.unescaped[lpos:rpos]
	// uHeader := (*reflect.SliceHeader)(unsafe.Pointer(&u))
	// uKeyHeader := (*reflect.StringHeader)(unsafe.Pointer(&ukey))
	// uKeyHeader.Data, uKeyHeader.Len = uHeader.Data, uHeader.Len
	// return
}

func scanKey(msg string) (tail, clip string, found bool) {
	var lpos, rpos int

	if tail, lpos = scanEscape( msg, '{'); lpos < 0 {
		return "", "", false
	}
	lpos++

	if tail, rpos = scanEscape( tail, '}'); rpos < 0 {
		return "", "", false
	}
	rpos++

	tail = msg[lpos+rpos:]
	clip = msg[lpos:lpos+rpos-1]
	found = true
	return
}

func scanEscape(msg string, sep rune)( tail string, n int){
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			fallthrough
		default:
		case r == '\\':
			esc = true
		case r == sep:
			return msg[n+1:], n
		}
	}
	return "", -1
}

// count unkeyed, basically
func (s *splicer) scanSplitKey(clip string) (key string){
	n := bytes.LastIndexByte([]byte(clip), ':')

	// no colon, no verb
	if n < 0 {
		// the unique string that is unkeyed with no verb -> unkeyed
		if clip == "{}" {
			return ""
		}
		// otherwise -> keyed
		return clip
	}

	// colon in 0-pos can't be escaped
	// -> unkeyed
	if n == 0 {
		return ""
	}

	// last colon escaped
	// -> clip is key
	if clip[n-1] == '\\' {
		return s.unescape(clip)
	}

	// last colon unescaped
	// -> clip up to n is key
	return s.unescape(clip[:n])
}

func (s *splicer) join(seg []Attr, ctx context.Context, args []any) {
	for _, a := range seg {
		s.dict.match(a)
	}

	ex := Segment(args...)
	for _, a := range ex {
		s.dict.match(a)
	}
	s.export = append(s.export, ex...)

	if ctx != nil {
		if as, ok := ctx.Value(segmentKey{}).([]Attr); ok {
			for _, a := range as {
				s.dict.match(a)
				s.export = append(s.export, a )
			}
		}
	}
}

// get a message. once.
func (s *splicer) msg() (msg string) {
	msg = string(s.text)
	s.text = s.text[:0]
	return
}

// after interpolation, freeze unsafely yields a string containing an interpolated message.
// It is catastrophically bad to read the string after free has been called.
// func (s *splicer) freezeUnsafe() (msg string) {
// 	textHeader := (*reflect.SliceHeader)(unsafe.Pointer(&s.text))
// 	msgHeader := (*reflect.StringHeader)(unsafe.Pointer(&msg))
// 	msgHeader.Data, msgHeader.Len = textHeader.Data, textHeader.Len
// 	return
// }

// DICT / LIST

type dict map[string]slog.Value

func (d dict) prematch(k string){
	d[k] = slog.StringValue(missingAttr)
}

func (d dict) match(a Attr){
	if _, found := d[a.Key]; found {
		d[a.Key] = a.Value
	}
}

func (d dict) insert(a Attr) {
	d[a.Key] = a.Value
}

func (d dict) clear() {
	for k := range d {
		delete(d, k)
	}
}

// INTERPOLATE

func (s *splicer) interpolate(msg string) {
	// interpolation loop
	var clip []byte
	var found bool
	for {
		if msg, clip, found = s.text.scanKey(msg); !found {
			break
		}
		s.interpAttr(clip)
	}
}

func (s *splicer) interpAttr(clip text) {
	key, verb := splitVerb(clip)

	if len(key) == 0 {
		s.interpUnkeyed(verb)
	} else {
		s.interpKeyed(key, verb)
	}
}

func (s *splicer) interpUnkeyed(verb text) {
	var arg any
	if len(s.list) > 0 {
		arg = s.list[0]
		s.list = s.list[1:]
	} else {
		s.text.appendString(missingArg)
		return
	}

	if a, isAttr := arg.(Attr); isAttr {
		s.text.appendValue(a.Value, verb)
		return
	}

	s.text.appendArg(arg, verb)
}

func (s *splicer) interpKeyed(key, verb text) {
	v, ok := s.dict[string(key)]

	// TODO: should be unreachable, but I kept reaching it
	if !ok {
		s.text.appendString(missingAttr)
		return
	}

	s.text.appendValue(v, verb)
}
