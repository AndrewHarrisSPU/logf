package logf

import (
	"context"

	"golang.org/x/exp/slog"
)

const labelKey = "label"
var noLabel = Attr{}

type Logger struct {
	h     handler
	level slog.Level
	depth int
}

// FromContext employs [slog.FromContext] to obtain a Logger from the given context.
// Precisely, the function returns the result of:
//
//	UsingHandler(slog.FromContext(ctx).Handler())
func FromContext(ctx context.Context) *Logger {
	return UsingHandler(slog.FromContext(ctx).Handler())
}

// UsingHandler returns a Logger employing the given slog.Handler
//
// If the given handler is not of a type native to logf, a new [Handler] is constructed, encapsulating the given handler.
func UsingHandler(h slog.Handler) *Logger {
	if h, isLogfHandler := h.(handler); isLogfHandler {
		return &Logger{h: h}
	}

	return &Logger{
		h: &Handler{
			enc:       h,
			addSource: true,
		},
	}
}

// Level returns a logger emitting at the given level.
// Level does not modify its receiver in any way.
func (l Logger) Level(level slog.Level) *Logger {
	l.level = level
	return &l
}

// Depth is used to modulate source file/line retrieval.
// Depth does not modify its receiver in any way.
func (l Logger) Depth(depth int) *Logger {
	l.depth = depth
	return &l
}

// With appends attributes held in a [Logger]'s handler.
// Arguments are converted to attributes with [Attrs].
func (l *Logger) With(args ...any) *Logger {
	*l = Logger{
		h:     l.h.WithAttrs(Attrs(args...)).(handler),
		level: l.level,
		depth: l.depth,
	}
	return l
}

// Group calls [slog.Logger.WithGroup] on a [Logger]'s handler.
func (l *Logger) Group(name string) *Logger {
	*l = Logger{
		h:     l.h.WithGroup(name).(handler),
		level: l.level,
		depth: l.depth,
	}
	return l
}

// Label sets the label text of the [Logger].
// Label text is helpful for identifying loggers in a [TTY].
// A label is not an attribute and does not affect logged output in any other way.
func (l *Logger) Label(label string) *Logger {
	*l = Logger{
		h:     l.h.withLabel(label),
		level: l.level,
		depth: l.depth,
	}
	return l
}

// Handler returns a handler associated with the Logger.
func (l *Logger) Handler() slog.Handler {
	return l.h.(slog.Handler)
}

func (l Logger) LogValue() slog.Value {
	return l.h.LogValue()
}

// LOGGING METHODS

// Msg interpolates a message string, and logs it.
func (l Logger) Msg(msg string, args ...any) {
	if !l.h.Enabled(l.level) {
		return
	}

	l.h.handle(l.level.Level(), msg, nil, l.depth, args)
}

// Err logs a message, appending the error string to the message text.
func (l Logger) Err(msg string, err error, args ...any) {
	if !l.h.Enabled(l.level) {
		return
	}

	l.h.handle(l.level.Level(), msg, err, l.depth, args)
}

// Msgf applies interpolation and formatting.
// The resulting string is returned, rather than logged.
func (l Logger) Msgf(msg string, args ...any) string {
	s, _ := l.h.fmt(msg, nil, args)
	return s
}

// Errf applies interpolation and formatting to wrap an error.
// The resulting error is returned, rather than logged.
func (l Logger) Errf(msg string, err error, args ...any) error {
	_, e := l.h.fmt(msg, err, args)
	return e
}
