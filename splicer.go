package logf

import (
	"context"
	"sync"

	"golang.org/x/exp/slog"
)

const (
	corruptKind         = "!corrupt-kind"
	missingAttr         = "!missing-attr"
	missingArg          = "!missing-arg"
	missingKey          = "!missing-key"
	missingRightBracket = "!missing-right-bracket"
)

var missingAttrValue = slog.StringValue("!missing-attr")

// Splicers have a well-defined lifecycle per logging call:
// 1. init
// - returns a fresh/cleared splicer from a pool.
// - a number of maps and splices have capacity but no elements.
// 2. scan
// - scans message for interpolation sites, each keyed interpolation added to dictionary
// - partitions arguments into unkeyed-values / keyed-attrs
// 3. join
// - attrs' values from a Handler, context, etc. are matched into interpolation dictionary
// - an exported attr list is built in order: handler, context, keyed-attrs
// 4. interpolate
// - while reading message, final text is written
// 5. export
// - after interpolation, message text and exported attrs are available
// 6. free
// - before returning to pool, zero out and clear internal slices and dict
type splicer struct {
	// final spliced output is written to text
	text []byte

	// holds parts of interpolated message that need escaping
	// also holds stack of keys when interpolating groups
	scratch []byte

	// holds ordered list of unkeyed arguments
	list []any

	// holds map of keyed interpolation symbols
	dict map[string]slog.Value

	// holds ordered list of exported attrs
	export []Attr
}

func newSplicer() *splicer {
	return spool.Get().(*splicer)
}

var spool = sync.Pool{
	New: func() any {
		return &splicer{
			text:    make([]byte, 0, 1024),
			scratch: make([]byte, 0, 1024),
			dict:    make(map[string]slog.Value, 5),
			list:    make([]any, 0, 5),
			export:  make([]Attr, 0, 5),
		}
	},
}

// contains heuristics for killing splicers that are too large
// TODO: think through implications
func (s *splicer) free() {
	const maxTextSize = 16 << 10
	const maxAttrSize = 128

	ok := cap(s.text)+cap(s.scratch) < maxTextSize
	ok = ok && (len(s.dict)+cap(s.list)+cap(s.export)) < maxAttrSize

	if ok {
		s.clear()
		spool.Put(s)
	}
}

// atm, clearing on "free" when cap/length is not over limits
// still researching sync.Pool, I think this is sane
func (s *splicer) clear() {
	// clear byte buffers
	s.text = s.text[:0]
	s.scratch = s.scratch[:0]

	// zero out and clear reference-holding components
	for i := range s.list {
		s.list[i] = any(nil)
	}
	s.list = s.list[:0]

	for i := range s.export {
		s.export[i] = Attr{}
	}
	s.export = s.export[:0]

	for k := range s.dict {
		delete(s.dict, k)
	}
}

// get a message.
func (s *splicer) msg() (msg string) {
	return string(s.text)
}

// INTERPOLATE

func (s *splicer) interpolate(msg string) {
	var clip []byte
	var found bool
	for {
		if msg, clip, found = s.writeUntilKey(msg); !found {
			break
		}
		s.interpolateAttr(clip)
	}
}

func (s *splicer) interpolateAttr(clip []byte) {
	key, verb := splitKeyVerb(clip)

	if len(key) == 0 {
		s.interpolateUnkeyed(verb)
	} else {
		s.interpolateKeyed(key, verb)
	}
}

func (s *splicer) interpolateUnkeyed(verb []byte) {
	var arg any
	if len(s.list) > 0 {
		arg = s.list[0]
		s.list = s.list[1:]
	} else {
		s.writeString(missingArg)
		return
	}

	if a, isAttr := arg.(Attr); isAttr {
		s.writeValue(a.Value, verb)
		return
	}

	s.writeArg(arg, verb)
}

func (s *splicer) interpolateKeyed(key, verb []byte) {
	v, ok := s.dict[string(key)]

	// should be unreachable, but I kept reaching it
	if !ok {
		s.writeString(missingAttr)
		return
	}

	s.writeValue(v, verb)
}

// after interpolation, freeze unsafely yields a string containing an interpolated message.
// but, it's catastrophically bad to read the string after free has been called.
// func (s *splicer) freezeUnsafe() (msg string) {
// 	textHeader := (*reflect.SliceHeader)(unsafe.Pointer(&s.text))
// 	msgHeader := (*reflect.StringHeader)(unsafe.Pointer(&msg))
// 	msgHeader.Data, msgHeader.Len = textHeader.Data, textHeader.Len
// 	return
// }

// JOIN / MATCH

// read each of seg / ctx / remaining args (order matters)
// update interpolation dictionary and export list
func (s *splicer) join(seg []Attr, ctx context.Context, args []any) {
	for _, a := range seg {
		s.match(a)
	}

	if ctx != nil {
		if as, ok := ctx.Value(segmentKey{}).([]Attr); ok {
			for _, a := range as {
				s.match(a)
				s.export = append(s.export, a)
			}
		}
	}

	for _, arg := range s.list {
		if a, ok := arg.(Attr); ok {
			s.match(a)
		}
	}

	ex := Segment(args...)
	for _, a := range ex {
		s.match(a)
	}
	s.export = append(s.export, ex...)
}

// root of matching invocation
func (s *splicer) match(a Attr) {
	if _, found := s.dict[a.Key]; found {
		s.dict[a.Key] = a.Value
	}
	if a.Value.Kind() == slog.GroupKind {
		// store a marker that deliminates s.scratch state before all matchRec operations
		gpos := len(s.scratch)

		// push attr key
		s.scratch = append(s.scratch, a.Key...)
		s.scratch = append(s.scratch, '.')

		s.matchRec(a.Value.Group(), gpos)

		// pop attr key
		s.scratch = s.scratch[:gpos]
	}
}

// recursive matching invocation
func (s *splicer) matchRec(group []Attr, gpos int) {
	// store a marker that deliminates s.scratch state per attr operation
	apos := len(s.scratch)

	for _, a := range group {
		// push attr key
		s.scratch = append(s.scratch, a.Key...)

		// match
		key := string(s.scratch[gpos:])
		if _, found := s.dict[key]; found {
			s.dict[key] = a.Value
		}

		// recursively matchRec, one deeper level
		// keep gpos invariant through matchRec
		if a.Value.Kind() == slog.GroupKind {
			s.scratch = append(s.scratch, '.')
			s.matchRec(a.Value.Group(), gpos)
		}

		// pop attr key
		s.scratch = s.scratch[:apos]
	}
}
