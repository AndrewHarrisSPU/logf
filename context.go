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

func (l Logger) Contextual() CtxLogger {
	return CtxLogger{
		h:     l.h,
		level: l.level,
	}
}

// Not much different than Logger (maybe it should wrap a Logger?)

func (l CtxLogger) Msg(ctx context.Context, msg string, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, nil, 0)
}

func (l CtxLogger) Err(ctx context.Context, msg string, err error, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, err, 0)
}

func (l CtxLogger) Fmt(ctx context.Context, msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	s.join(ctx, l.h.seg, args)
	s.interpolate(msg)

	if err != nil && len(s.text) > 0 {
		err = fmt.Errorf(string(s.text)+": %w", err)
	}

	s.text.appendError(err)

	return string(s.text), err
}

func (l CtxLogger) Level(level slog.Leveler) CtxLogger {
	return CtxLogger{
		h:     l.h,
		level: level,
	}
}

func (l CtxLogger) With(args ...any) CtxLogger {
	return CtxLogger{
		h:     l.h.with(segment(args)),
		level: l.level,
	}
}
