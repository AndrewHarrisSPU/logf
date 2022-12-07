package logf

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

	// holds map of keyed interpolation symbols
	dict map[string]slog.Value

	// holds ordered list of exported attrs
	export []Attr

	// holds number of unkeyed attrs
	nUnkeyed int
	iUnkeyed int

	// false if scanning indicates no interpolation
	interpolates bool
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
	ok = ok && (len(s.dict)+cap(s.export)) < maxAttrSize

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
	s.interpolates = false
}

// return spliced text
func (s *splicer) line() string {
	return string(s.text)
}

// JOIN / MATCH

func (s *splicer) joinAttrList(as []Attr) {
	for _, a := range as {
		s.export = append(s.export, a)
	}
}

func (s *splicer) joinList(args []any) {
	for len(args) > 0 {
		args = parseAttr(&s.export, args)
	}
}

// used for joining Record attrs
func (s *splicer) joinOne(a Attr) {
	s.export = append(s.export, a)
}

func (s *splicer) matchAll(scope string, as []Attr, replace func(Attr) Attr) {
	for _, a := range as {
		s.match(scope, a, replace)
	}
	for _, a := range s.export {
		s.match(scope, a, replace)
	}
}

// root of matching invocation
// here, an attr key is known to be needed for interpolation
// match attempts to puts the right value in the dictionary
func (s *splicer) match(scope string, a Attr, replace func(Attr) Attr) {
	// match if raw attr key is found in dictionary
	if _, found := s.dict[a.Key]; found {
		if replace != nil {
			a = replace(a)
		}
		s.dict[a.Key] = a.Value
		return
	}

	// match if given prefix + attr key is found in dictionary
	if _, found := s.dict[scope+a.Key]; found {
		if replace != nil {
			a = replace(a)
		}
		s.dict[scope+a.Key] = a.Value
		return
	}

	// if lv, ok := a.Value.Any().(slog.LogValuer); ok {
	// v := lv.LogValue().Resolve()
	// if v.Kind() == slog.GroupKind {
	// 	gpos := len(s.scratch)
	// 	s.scratch = append(s.scratch, a.Key...)
	// 	s.scratch = append(s.scratch, '.')

	// 	s.matchRec(v.Group(), gpos, replace)

	// 	s.scratch = s.scratch[:gpos]
	// }
	// 	return
	// }

	if a.Value.Kind() == slog.GroupKind {
		// store a marker that deliminates s.scratch state before subsequent matchRec operations
		gpos := len(s.scratch)

		// push attr key
		s.scratch = append(s.scratch, a.Key...)
		s.scratch = append(s.scratch, '.')

		s.matchRec(a.Value.Group(), gpos, replace)

		// pop attr key
		s.scratch = s.scratch[:gpos]
	}
}

// recursive matching invocation
func (s *splicer) matchRec(group []Attr, gpos int, replace func(Attr) Attr) {
	// store a marker that deliminates s.scratch state per attr operation
	apos := len(s.scratch)

	for _, a := range group {
		// push attr key
		s.scratch = append(s.scratch, a.Key...)

		// match
		key := string(s.scratch[gpos:])
		if _, found := s.dict[key]; found {
			if replace != nil {
				a = replace(a)
			}
			s.dict[key] = a.Value
		}

		// if lv, ok := a.Value.Any().(slog.LogValuer); ok {
		// 	v := lv.LogValue().Resolve()
		// 	if v.Kind() == slog.GroupKind {
		// 		s.scratch = append(s.scratch, '.')
		// 		s.matchRec(v.Group(), gpos, replace)
		// 	}
		// }

		// recursively matchRec, one deeper level
		// (keep gpos invariant through matchRec)
		if a.Value.Kind() == slog.GroupKind {
			s.scratch = append(s.scratch, '.')
			s.matchRec(a.Value.Group(), gpos, replace)
		}

		// pop attr key
		s.scratch = s.scratch[:apos]
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

func (s *splicer) writeArg(arg any, verb []byte) {
	switch arg := arg.(type) {
	case Attr:
		s.WriteValue(arg.Value, verb)
	case []Attr:
		s.writeGroup(arg)
	case slog.LogValuer:
		s.WriteValue(arg.LogValue(), verb)
	default:
		s.WriteValue(slog.AnyValue(arg), verb)
	}
}

func (s *splicer) WriteValue(v slog.Value, verb []byte) {
	if len(verb) > 0 {
		s.writeValueVerb(v, string(verb))
	} else {
		s.writeValueNoVerb(v)
	}
}

func (s *splicer) writeValueNoVerb(v slog.Value) {
	switch v.Kind() {
	case slog.StringKind:
		s.WriteString(v.String())
	case slog.BoolKind:
		s.text = strconv.AppendBool(s.text, v.Bool())
	case slog.Float64Kind:
		s.text = strconv.AppendFloat(s.text, v.Float64(), 'g', -1, 64)
	case slog.Int64Kind:
		s.text = strconv.AppendInt(s.text, v.Int64(), 10)
	case slog.Uint64Kind:
		s.text = strconv.AppendUint(s.text, v.Uint64(), 10)
	case slog.DurationKind:
		s.text = appendDuration(s.text, v.Duration())
	case slog.TimeKind:
		s.text = appendTimeRFC3339Millis(s.text, v.Time())
	case slog.GroupKind:
		s.writeGroup(v.Group())
	case slog.LogValuerKind:
		s.writeValueNoVerb(v.Resolve())
	case slog.AnyKind:
		fmt.Fprintf(s, "%v", v.Any())
	default:
		panic(corruptKind)
	}
}

func (s *splicer) writeValueVerb(v slog.Value, verb string) {
	switch v.Kind() {
	case slog.StringKind:
		fmt.Fprintf(s, verb, v.String())
	case slog.BoolKind:
		fmt.Fprintf(s, verb, v.Bool())
	case slog.Float64Kind:
		fmt.Fprintf(s, verb, v.Float64())
	case slog.Int64Kind:
		fmt.Fprintf(s, verb, v.Int64())
	case slog.Uint64Kind:
		fmt.Fprintf(s, verb, v.Uint64())
	case slog.DurationKind:
		s.writeDurationVerb(v.Duration(), verb)
	case slog.TimeKind:
		s.writeTimeVerb(v.Time(), verb)
	case slog.GroupKind:
		s.writeGroup(v.Group())
	case slog.LogValuerKind:
		s.writeValueVerb(v.Resolve(), verb)
	case slog.AnyKind:
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

func (s *splicer) writeError(extended int, err error) {
	if extended > 0 {
		s.WriteString(": ")
	}
	s.WriteString(err.Error())
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
