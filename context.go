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
	h     handler
	level slog.Level
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
func (l LoggerCtx) Level(level slog.Level) LoggerCtx {
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

// See [Logger.Scope]
func (l LoggerCtx) Scope(name string) LoggerCtx {
	l.h = l.h.withGroup(name)
	return l
}

// See [Logger.Msg]
func (l LoggerCtx) Msg(ctx context.Context, msg string, args ...any) {
	if l.level < l.h.level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), ctx, args)
	l.h.handle(s, l.level.Level(), msg, nil, l.depth)
}

// See [Logger.Err]
func (l LoggerCtx) Err(ctx context.Context, msg string, err error, args ...any) {
	if l.level < l.h.level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), ctx, args)
	l.h.handle(s, l.level.Level(), msg, err, l.depth)
}

// See [Logger.Fmt]
func (l LoggerCtx) Fmt(ctx context.Context, msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), nil, args)

	s.interpolate(msg)

	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.msg(), err)
		msg = err.Error()
	}

	return msg, err
}
