package logf

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/exp/slog"
)

type Logger struct {
	*slog.Logger
	h handler
}

// FromContext employs [slog.FromContext] to obtain a Logger from the given context.
// Precisely, the function returns the result of:
//
//	UsingHandler(slog.FromContext(ctx).Handler())
func FromContext(ctx context.Context) Logger {
	return UsingHandler(slog.FromContext(ctx).Handler())
}

// UsingHandler returns a Logger employing the given slog.Handler
//
// If the given handler is not of a type native to logf, a new [Handler] is constructed, encapsulating the given handler.
func UsingHandler(h slog.Handler) Logger {
	if h, isLogfHandler := h.(handler); isLogfHandler {
		return Logger{slog.New(h), h}
	}

	lh := &Handler{
		enc:       h,
		addSource: true,
	}

	return newLogger(lh)

	// if h, isLogfHandler := h.(handler); isLogfHandler {
	// 	return Logger{h: h}
	// }

	// return Logger{
	// 	h: &Handler{
	// 		enc:       h,
	// 		addSource: true,
	// 	},
	// }
}

func newLogger(h handler) Logger {
	return Logger{slog.New(h), h}
}

// With appends attributes held in a [Logger]'s handler.
// Arguments are converted to attributes with [Attrs].
func (l Logger) With(args ...any) Logger {
	h := l.h.WithAttrs(parseAttrs(args)).(handler)
	return newLogger(h)
}

// Group calls [slog.Logger.WithGroup] on a [Logger]'s handler.
func (l Logger) Group(name string) Logger {
	h := l.h.WithGroup(name).(handler)
	return newLogger(h)
}

// Tag configures a tag that appears in [TTY] log output.
// Tags set by this method override; only one is set per logger.
func (l Logger) Tag(tag string) Logger {
	h := l.h.withTag(tag)
	return newLogger(h)
}

// Handler returns a handler associated with the Logger.
func (l Logger) Handler() slog.Handler {
	return l.h
}

// LogValue returns the set of [Attr]s accrued by the Logger's handler.
func (l Logger) LogValue() slog.Value {
	return l.h.LogValue()
}

// LOGGING METHODS
/*
func (l Logger) Log(level slog.Level, msg string, args ...any){
	if !l.h.Enabled(level) {
		return
	}

	s := newSplicer()

	l.h.handle(s, level, msg, nil, 0, args)
}

func (l Logger) Debug(msg string, args ...any) {
	if !l.h.Enabled(DEBUG) {
		return
	}

	s := newSplicer()

	l.h.handle(s, DEBUG, msg, nil, 0, args)
}

// Info performs interpolation on the given message string, and logs.
func (l Logger) Info(msg string, args ...any) {
	if !l.h.Enabled(INFO) {
		return
	}

	s := newSplicer()

	l.h.handle(s, INFO, msg, nil, 0, args)
}

func (l Logger) Warn(msg string, args ...any) {
	if !l.h.Enabled(WARN) {
		return
	}

	s := newSplicer()

	l.h.handle(s, WARN, msg, nil, 0, args)
}

func (l Logger) Error(msg string, err error, args ...any) {
	if !l.h.Enabled(ERROR) {
		return
	}

	s := newSplicer()

	l.h.handle(s, ERROR, msg, err, 0, args)
}

func (l Logger) LogDepth(depth int, level slog.Level, msg string, args ...any){
	if !l.h.Enabled(level) {
		return
	}

	s := newSplicer()

	l.h.handle(s, level, msg, nil, depth, args)
}
*/
// Fmt applies interpolation to the given message string.
// The resulting string is returned, rather than logged.
func (l Logger) Fmt(msg string, args ...any) string {
	s := l.h.fmt(msg, args)
	// s := l.h.fmt(msg, args)
	defer s.free()

	return s.line()
}

// NewErr applies interpolation and formatting to wrap an error.
// If the given err is nil, a new error is constructed.
// The resulting error is returned, rather than logged.
func (l Logger) NewErr(msg string, err error, args ...any) error {
	s := l.h.fmt(msg, args)

	// s := l.h.fmt(msg, args)
	defer s.free()

	if err == nil {
		return errors.New(s.line())
	}

	if len(s.text) > 0 {
		s.WriteString(": ")
	}
	s.WriteString("%w")
	return fmt.Errorf(s.line(), err)
}
