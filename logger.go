package logf

import (
	"context"

	"golang.org/x/exp/slog"
)

type Logger struct {
	h     handler
	level slog.Level
	depth int
}

func FromContext(ctx context.Context) Logger {
	return UsingHandler(slog.FromContext(ctx).Handler())
}

func UsingHandler(h slog.Handler) Logger {
	if tty, isTTY := h.(*TTY); isTTY {
		return tty.Logger()
	}

	if logfh, isLogfHandler := h.(*Handler); isLogfHandler {
		return Logger{h: logfh}
	}

	return Logger{
		h: &Handler{
			enc: h,
		},
	}
}

// Level is intended for chaining calls, e.g.:
// log.Level(INFO+1).Msg("") logs at INFO+1
func (l Logger) Level(level slog.Level) Logger {
	l.level = level
	return l
}

// Depth is used to modulate source file/line retrieval.
func (l Logger) Depth(depth int) Logger {
	l.depth = depth
	return l
}

// With extends the structure held in the Logger.
// Arguments are munged through Segment.
func (l Logger) With(args ...any) Logger {
	l.h = l.h.withAttrs(segment(args...))
	return l
}

// Label
func (l Logger) Label(name string) Logger {
	l.h = l.h.withGroup(name)
	return l
}

// LOGGING METHODS

// Msg interpolates a message string, and logs it.
func (l Logger) Msg(msg string, args ...any) {
	if !l.h.enabled(l.level) {
		return
	}

	l.h.handle(l.level.Level(), msg, nil, l.depth, args)
}

// Err logs a message, appending the error string to the message text.
func (l Logger) Err(msg string, err error, args ...any) {
	if !l.h.enabled(l.level) {
		return
	}

	l.h.handle(l.level.Level(), msg, err, l.depth, args)
}

// Fmt interpolates like [Logger.Msg] or [Logger.Err].
// The result is not written to a log, but returned.
// The returned string is the interpolation of msg.
// With a nil error, Fmt emits a nil error.
// Otherwise, the returned error stringifies to the returned string.
// but is wrapped with [fmt.Errorf] (preserving [errors.Is], [errors.As] behavior).
func (l Logger) Fmt(msg string, err error, args ...any) (string, error) {
	return l.h.fmt(msg, err, args)
}

func (l Logger) Handler() slog.Handler {
	switch h := l.h.(type) {
	case *Handler:
		return h
	case *TTY:
		return h
	default:
		panic("unknown handler type")
	}
}
