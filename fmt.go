package logf

import (
	"errors"
	"fmt"
)

func logFmt(l Logger, f string, args []any) string {
	h, ok := l.Handler().(handler)
	if !ok {
		return f
	}

	var store Store
	var replace func([]string, Attr) Attr
	switch h := h.(type) {
	case *Handler:
		store = h.store
		replace = h.replace
	case *TTY:
		store = h.store
		replace = h.dev.replace
	}

	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	s.joinStore(store, replace)
	for _, a := range Attrs(args...) {
		s.joinLocal(store.scope, a, replace)
	}
	s.ipol(f)

	return s.line()
}

func logFmtErr(l Logger, f string, err error, args []any) error {
	h, ok := l.Handler().(handler)
	if !ok {
		return err
	}

	var store Store
	var replace func([]string, Attr) Attr
	switch h := h.(type) {
	case *Handler:
		store = h.store
		replace = h.replace
	case *TTY:
		store = h.store
		replace = h.dev.replace
	}

	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	s.joinStore(store, replace)
	for _, a := range Attrs(args...) {
		s.joinLocal(store.scope, a, replace)
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

// Fmt interpolates the f string with the given arguments.
// The arguments parse as with [Attrs].
func Fmt(f string, args ...any) string {
	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	for _, a := range Attrs(args...) {
		s.joinLocal(nil, a, nil)
	}
	s.ipol(f)

	return s.line()
}

// WrapErr interpolates the f string with the given arguments and error.
// The arguments parse as with [Attrs].
// The returned error matches [errors.Is]/[errors.As] behavior, as with [fmt.Errorf].
func WrapErr(f string, err error, args ...any) error {
	s := newSplicer()
	defer s.free()

	s.scanMessage(f)
	for _, a := range Attrs(args...) {
		s.joinLocal(nil, a, nil)
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
