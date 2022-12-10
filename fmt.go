package logf

import (
	"fmt"
	"errors"
)

func logFmt(l Logger, f string, args []any) string {
	h, ok := l.Handler().(handler)
	if !ok {
		return f
	}

	var as []Attr
	var scope string
	var replace func(Attr) Attr
	switch h := h.(type) {
	case *Handler:
		as = h.attrs		
		scope = h.scope
		replace = h.replace
	case *TTY:
		as = h.attrs		
		scope = h.scope
		replace = h.fmtr.sink.replace
	}

	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	s.joinAttrs(as, scope, replace)
	for _, a := range Attrs(args...){
		s.joinOne(a, scope, replace)
	}
	s.ipol(f)

	return s.line()
}

func logFmtErr(l Logger, f string, err error, args []any) error {
	h, ok := l.Handler().(handler)
	if !ok {
		return err
	}

	var as []Attr
	var scope string
	var replace func(Attr) Attr
	switch h := h.(type) {
	case *Handler:
		as = h.attrs		
		scope = h.scope
		replace = h.replace
	case *TTY:
		as = h.attrs		
		scope = h.scope
		replace = h.fmtr.sink.replace
	}

	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	s.joinAttrs(as, scope, replace)
	for _, a := range Attrs(args...) {
		s.joinOne(a, "", nil)
	}
	s.ipol(f)

	if err == nil {
		return errors.New(s.line())
	}

	if len(s.text) > 0 {
		s.WriteString(": ")
	}
	s.WriteString("%w")
	return fmt.Errorf(s.line(), err)	
}

func Fmt(f string, args ...any) string {
	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	for _, a := range Attrs(args...) {
		s.joinOne(a, "", nil)
	}
	s.ipol(f)

	return s.line()
}

func FmtError(f string, err error, args ...any) error {
	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	for _, a := range Attrs(args...) {
		s.joinOne(a, "", nil)
	}
	s.ipol(f)

	if err == nil {
		return errors.New(s.line())
	}

	if len(s.text) > 0 {
		s.WriteString(": ")
	}
	s.WriteString("%w")
	return fmt.Errorf(s.line(), err)
}