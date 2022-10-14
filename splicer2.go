package logf

import (
	"golang.org/x/exp/slog"
)

type dict map[string]slog.Value

func (d dict) prematch(k string) {
	d[k] = slog.StringValue(missingAttr)
}

func (d dict) insert(a Attr) {
	d[a.Key] = a.Value
}

func (d dict) clear() {
	for k := range d {
		delete(d, k)
	}
}

// root of matching invocation
func (s *splicer) match(a Attr) {
	if _, found := s.dict[a.Key]; found {
		s.dict[a.Key] = a.Value
	}
	if a.Value.Kind() == slog.GroupKind {
		// store a marker that deliminates s.scratch state before matchRec operations
		gpos := len(s.scratch)

		// push attr key
		s.scratch = append(s.scratch, a.Key...)
		s.scratch = append(s.scratch, '.')

		s.matchRec(a, gpos)

		// pop attr key
		s.scratch = s.scratch[:gpos]
	}
}

// recursive matching invocation
func (s *splicer) matchRec(group Attr, gpos int) {
	// store a marker that deliminates s.scratch state per attr operation
	apos := len(s.scratch)

	// iterate group elements and match
	for _, a := range group.Value.Group() {
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
			s.matchRec(a, gpos)
		}

		// pop attr key
		s.scratch = s.scratch[:apos]
	}
}
