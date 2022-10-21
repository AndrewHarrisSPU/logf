package logf

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"
)

type segmentKey struct{}

// A LoggerCtx demands contexts for logging calls.
// It's a better choice if contexts carry transient segments of Attrs.
type LoggerCtx struct {
	h     *Handler
	level slog.Leveler
	depth int
}

// Contextual returns a LoggerCtx that is otherwise identical to the Logger
func (l Logger) Contextual() LoggerCtx {
	return LoggerCtx{
		h:     l.h,
		level: l.level,
		depth: l.depth,
	}
}

// See [Logger.Level]
func (l LoggerCtx) Level(level slog.Leveler) LoggerCtx {
	l.level = level
	return l
}

// See [Logger.Depth]
func (l LoggerCtx) Depth(depth int) LoggerCtx {
	l.depth = depth
	return l
}

// See [Logger.With]
func (l LoggerCtx) With(args ...any) LoggerCtx {
	l.h = l.h.withAttrs(Segment(args...))
	return l
}

// See [Logger.WithScope]
func (l LoggerCtx) WithGroup(name string) LoggerCtx {
	l.h = l.h.withGroup(name)
	return l
}

// See [Logger.Msg]
func (l LoggerCtx) Msg(ctx context.Context, msg string, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, ctx, args)
	l.h.handle(s, l.level.Level(), msg, nil, 0)
}

// See [Logger.Err]
func (l LoggerCtx) Err(ctx context.Context, msg string, err error, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, ctx, args)
	l.h.handle(s, l.level.Level(), msg, err, 0)
}

// See [Logger.Fmt]
func (l LoggerCtx) Fmt(ctx context.Context, msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, nil, args)

	s.interpolate(msg)

	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.msg(), err)
		msg = err.Error()
	}

	return msg, err
}
