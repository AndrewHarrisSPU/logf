package logf

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/exp/slog"
)

type Logger struct {
	h     handler
	level slog.Level
	depth int
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
		return Logger{h: h}
	}

	return Logger{
		h: &Handler{
			enc:       h,
			addSource: true,
		},
	}
}

// Level returns a logger emitting at the given level.
func (l Logger) Level(level slog.Level) Logger {
	l.level = level
	return l
}

// Depth is used to modulate source file/line retrieval.
func (l Logger) Depth(depth int) Logger {
	l.depth = depth
	return l
}

// With appends attributes held in a [Logger]'s handler.
// Arguments are converted to attributes with [Attrs].
func (l Logger) With(args ...any) Logger {
	return Logger{
		// h:     l.h.WithAttrs(Attrs(args...)).(handler),
		h: l.h.WithAttrs(parseAttrs(args)).(handler),
		level: l.level,
		depth: l.depth,
	}
}

// Group calls [slog.Logger.WithGroup] on a [Logger]'s handler.
func (l Logger) Group(name string) Logger {
	return Logger{
		h:     l.h.WithGroup(name).(handler),
		level: l.level,
		depth: l.depth,
	}
}

// Tag configures a tag that appears in [TTY] log output.
// Tags set by this method override; only one is set per logger.
func (l Logger) Tag(tag string) Logger {
	return Logger{
		h:     l.h.withTag(tag),
		level: l.level,
		depth: l.depth,
	}
}

// Handler returns a handler associated with the Logger.
func (l Logger) Handler() slog.Handler {
	return l.h.(slog.Handler)
}


// LogValue returns the set of [Attr]s accrued by the Logger's handler.
func (l Logger) LogValue() slog.Value {
	return l.h.LogValue()
}

// LOGGING METHODS

// Msg logs the given message string. No interpolation is performed.
func (l Logger) Msg(msg string, args ...any) {
	if !l.h.Enabled(l.level) {
		return
	}

	s := newSplicer()

	l.h.handle(s, l.level, msg, nil, l.depth, args)
}

// Msgf performs interpolation on the given message string, and logs.
func (l Logger) Msgf(msg string, args ...any) {
	if !l.h.Enabled(l.level) {
		return
	}

	s := newSplicer()
	args = s.scan(msg, args)

	l.h.handle(s, l.level, msg, nil, l.depth, args)
}

// Err logs a message, appending the error string to the message text.
// Interpolation is performed on the given message string.
func (l Logger) Err(msg string, err error, args ...any) {
	if !l.h.Enabled(l.level) {
		return
	}

	s := newSplicer()

	l.h.handle(s, l.level, msg, err, l.depth, args)
}

// Fmt applies interpolation to the given message string.
// The resulting string is returned, rather than logged.
func (l Logger) Fmt(msg string, args ...any) string {
	s := l.h.fmt(msg, args)
	defer s.free()

	return s.line()
}

// NewErr applies interpolation and formatting to wrap an error.
// If the given err is nil, a new error is constructed.
// The resulting error is returned, rather than logged.
func (l Logger) NewErr(msg string, err error, args ...any) error {
	s := l.h.fmt(msg, args)
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
