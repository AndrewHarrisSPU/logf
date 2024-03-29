package logf

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"log/slog"
)

const (
	corruptKind = "!corrupt-kind"
	missingAttr = "!missing-attr"
	missingArg  = "!missing-arg"
	missingKey  = "!missing-key"
)

// LIFECYCLE

// Splicers have a well-defined lifecycle per logging call:
// 1. init
// - returns a fresh/cleared splicer from a pool.
// - splicer components likely have allocated capacity but no elements.
// 2. scan
// - scans message for interpolation sites; each keyed interpolation added to dictionary
// - partitions arguments into unkeyed-values / keyed-attrs
// 3. join
// - attrs from a handler etc. are matched into interpolation dictionary
// - an exported attr list is built (order is observed; last seen wins)
// 4. interpolate
// - while reading message, final text is written
// 5. export
// - after interpolation, splicer text is availiable via s.line(), exports via s.export
// 6. free
// - before returning to pool, zero out and clear internal slices and dict
type splicer struct {
	// final spliced output is written to text
	text []byte

	// holds parts of interpolated message that need escaping
	// also holds stack of keys when interpolating groups
	scratch []byte

	matchStack []string

	// holds map of keyed interpolation symbols
	dict map[string]slog.Value

	// holds ordered list of exported attrs
	export []Attr

	// holds number of unkeyed attrs
	iUnkeyed int
}

func newSplicer() *splicer {
	return spool.Get().(*splicer)
}

var spool = sync.Pool{
	New: func() any {
		return &splicer{
			text:       make([]byte, 0, 1024),
			scratch:    make([]byte, 0, 1024),
			matchStack: make([]string, 0, 16),
			dict:       make(map[string]slog.Value, 5),
			export:     make([]Attr, 0, 5),
		}
	},
}

// contains heuristics for killing splicers that are too large
// TODO: think through implications
func (s *splicer) free() {
	const maxTextSize = 16 << 10
	const maxAttrSize = 128
	const maxStackSize = 128

	ok := cap(s.text)+cap(s.scratch) < maxTextSize
	ok = ok && (len(s.dict)+cap(s.export)) < maxAttrSize
	ok = ok && (len(s.matchStack)) < maxStackSize

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
	for i := range s.export {
		s.export[i] = Attr{}
	}
	s.export = s.export[:0]

	for k := range s.dict {
		delete(s.dict, k)
	}

	s.iUnkeyed = 0
}

// return spliced text
func (s *splicer) line() string {
	return string(s.text)
}

// JOIN / MATCH
func (s *splicer) joinStore(store Store, replace replaceFunc) {
	store.Attrs(func(scope []string, a Attr) {
		s.match(scope, a, replace)
	})
}

func (s *splicer) joinLocal(stack []string, a Attr, replace replaceFunc) {
	if replace != nil {
		a = replace(stack, a)
	}

	s.export = append(s.export, a)
	s.matchLocal(stack, a, replace)
	s.match(stack, a, replace)
}

func (s *splicer) matchLocal(stack []string, a Attr, replace replaceFunc) {
	if replace != nil {
		a = replace(stack, a)
	}

	if _, found := s.dict[a.Key]; found {
		s.dict[a.Key] = a.Value
	}

	if a.Value.Kind() == slog.KindGroup {
		stack = append(stack, a.Key)

		for _, a := range a.Value.Group() {
			s.match(stack, a, replace)
		}
	}
}

func (s *splicer) match(stack []string, a Attr, replace replaceFunc) {
	if replace != nil {
		a = replace(stack, a)
	}

	var key string
	if len(stack) > 0 {
		key = strings.Join(stack, ".")
		key += "."
	}
	key += a.Key

	// properly scoped
	if _, found := s.dict[key]; found {
		s.dict[key] = a.Value
	}

	if a.Value.Kind() == slog.KindGroup {
		stack = append(stack, a.Key)

		for _, a := range a.Value.Group() {
			s.match(stack, a, replace)
		}
	}
}

// WRITES

func (s *splicer) Write(p []byte) (int, error) {
	s.text = append(s.text, p...)
	return len(p), nil
}

func (s *splicer) WriteByte(c byte) error {
	s.text = append(s.text, c)
	return nil
}

func (s *splicer) writeRune(r rune) {
	s.text = utf8.AppendRune(s.text, r)
}

func (s *splicer) WriteString(m string) (int, error) {
	s.text = append(s.text, m...)
	return len(m), nil
}

// TYPED WRITES

func (s *splicer) WriteValue(v slog.Value, verb []byte) {
	if len(verb) > 0 {
		s.writeValueVerb(v, string(verb))
	} else {
		s.writeValueNoVerb(v)
	}
}

func (s *splicer) writeValueNoVerb(v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		s.WriteString(v.String())
	case slog.KindBool:
		s.text = strconv.AppendBool(s.text, v.Bool())
	case slog.KindFloat64:
		s.text = strconv.AppendFloat(s.text, v.Float64(), 'g', -1, 64)
	case slog.KindInt64:
		s.text = strconv.AppendInt(s.text, v.Int64(), 10)
	case slog.KindUint64:
		s.text = strconv.AppendUint(s.text, v.Uint64(), 10)
	case slog.KindDuration:
		s.text = appendDuration(s.text, v.Duration())
	case slog.KindTime:
		s.text = appendTimeRFC3339Millis(s.text, v.Time())
	case slog.KindGroup:
		s.writeGroup(v.Group())
	case slog.KindLogValuer:
		s.writeValueNoVerb(v.Resolve())
	case slog.KindAny:
		fmt.Fprintf(s, "%v", v.Any())
	default:
		panic(corruptKind)
	}
}

func (s *splicer) writeValueVerb(v slog.Value, verb string) {
	switch v.Kind() {
	case slog.KindString:
		fmt.Fprintf(s, verb, v.String())
	case slog.KindBool:
		fmt.Fprintf(s, verb, v.Bool())
	case slog.KindFloat64:
		fmt.Fprintf(s, verb, v.Float64())
	case slog.KindInt64:
		fmt.Fprintf(s, verb, v.Int64())
	case slog.KindUint64:
		fmt.Fprintf(s, verb, v.Uint64())
	case slog.KindDuration:
		s.writeDurationVerb(v.Duration(), verb)
	case slog.KindTime:
		s.writeTimeVerb(v.Time(), verb)
	case slog.KindGroup:
		s.writeGroup(v.Group())
	case slog.KindLogValuer:
		s.writeValueVerb(v.Resolve(), verb)
	case slog.KindAny:
		fmt.Fprintf(s, verb, v.Any())
	default:
		panic(corruptKind)
	}
}

func (s *splicer) writeTimeVerb(t time.Time, verb string) {
	switch verb {
	case "epoch":
		s.text = strconv.AppendInt(s.text, t.Unix(), 10)
	case "RFC3339":
		s.text = t.AppendFormat(s.text, time.RFC3339)
	case "kitchen":
		s.text = t.AppendFormat(s.text, time.Kitchen)
	case "stamp":
		s.text = t.AppendFormat(s.text, time.Stamp)
	default:
		// TODO: might be slow /shrug
		s.text = t.AppendFormat(s.text, strings.Replace(verb, ";", ":", -1))
	}
}

func (s *splicer) writeDurationVerb(d time.Duration, verb string) {
	switch verb {
	case "epoch":
		s.text = strconv.AppendInt(s.text, int64(d), 10)
	default:
		fmt.Fprintf(s, verb, d.String())
	}
}

func (s *splicer) writeGroup(as []Attr) {
	next := byte('[')
	for _, a := range as {
		s.WriteByte(next)
		s.WriteString(a.Key)
		s.WriteByte('=')
		s.writeValueNoVerb(a.Value)
		next = ' '
	}
	s.WriteByte(']')
}
