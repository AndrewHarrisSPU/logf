package logf

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"
)

type segmentKey struct{}

// A CtxLogger demands contexts for logging calls.
// It's a better choice if contexts carry transient segments of Attrs.
type CtxLogger struct {
	h     *Handler
	level slog.Leveler
}

// Contextual returns a CtxLogger that is otherwise identical to the Logger
func (l Logger) Contextual() CtxLogger {
	return CtxLogger{
		h:     l.h,
		level: l.level,
	}
}

// See [Logger.Msg]
func (l CtxLogger) Msg(ctx context.Context, msg string, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, nil, 0)
}

// See [Logger.Err]
func (l CtxLogger) Err(ctx context.Context, msg string, err error, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, err, 0)
}

// See [Logger.Fmt]
func (l CtxLogger) Fmt(ctx context.Context, msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)

	s.interpolate(msg)

	if err != nil && len(msg) > 0 {
		s.text.appendString(": %w")
		err = fmt.Errorf(s.msg(), err)
		msg = err.Error()
	}

	return msg, err
}

// See [Logger.Level]
func (l CtxLogger) Level(level slog.Leveler) CtxLogger {
	return CtxLogger{
		h:     l.h,
		level: level,
	}
}

// See [Logger.With]
func (l CtxLogger) With(args ...any) CtxLogger {
	return CtxLogger{
		h:     l.h.with(Segment(args...)),
		level: l.level,
	}
}
