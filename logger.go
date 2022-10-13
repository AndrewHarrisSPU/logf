package logf

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"
)

type Logger struct {
	h     *Handler
	level slog.Leveler
}

// CONSTRUCTION

// Pass options to New, get a Logger!
func New(options ...Option) Logger {
	return Logger{
		h:     NewHandler(options...),
		level: slog.InfoLevel,
	}
}

// Level is intended for chaining calls, e.g.:
// log.Level(INFO+1).Msg("") logs at INFO+1
func (l Logger) Level(level slog.Leveler) Logger {
	return Logger{
		h:     l.h,
		level: level,
	}
}

// With extends the structure held in the Logger.
// Arguments are munged through Segment.
func (l Logger) With(args ...any) Logger {
	return Logger{
		h:     l.h.with(Segment(args...)),
		level: l.level,
	}
}

func (l Logger) WithScope(name string) Logger {
	return Logger{
		h:     l.h.withScope(name),
		level: l.level,
	}
}

// LOGGING METHODS

// Msg logs a message
func (l Logger) Msg(msg string, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, nil, args)
	l.h.handle(s, l.level.Level(), msg, nil, 0)
}

// Err logs a message with an appended error
func (l Logger) Err(msg string, err error, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, nil, args)
	l.h.handle(s, l.level.Level(), msg, err, 0)
}

// Fmt interpolates like [Logger.Msg] or [Logger.Err].
// The result is not written to a log,
func (l Logger) Fmt(msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.seg, nil, args)

	s.interpolate(msg)

	if err != nil && len(msg) > 0 {
		s.text.appendString(": %w")
		err = fmt.Errorf(s.msg(), err)
		msg = err.Error()
	}

	return msg, err
}

// SEGMENT

func NewAttr(key string, value any) Attr {
	return slog.Any(key, value)
}

// Segment munges arguments to Attrs, returning a slice of attrs 'seg'.
//   - Pairs of (string, any) result in an Attr, appended to seg.
//   - Attrs are appended to seg.
//   - []Attrs, contexts, and logf's Loggers, CtxLoggers, Handlers are flattened and appended seg.
func Segment(args ...any) (seg []Attr) {
	for len(args) > 0 {
		switch arg := args[0].(type) {
		case string:
			if len(args) == 1 {
				seg = append(seg, slog.String(arg, missingArg))
				return
			}
			seg = append(seg, NewAttr(arg, args[1]))
			args = args[2:]
		case Attr:
			seg = append(seg, arg)
			args = args[1:]
		case []Attr:
			seg = append(seg, arg...)
			args = args[1:]
		case context.Context:
			if ctxSeg, ok := arg.Value(segmentKey{}).([]Attr); ok {
				seg = append(seg, ctxSeg...)
			}
			args = args[1:]
		case Logger:
			seg = append(seg, arg.h.seg...)
			args = args[1:]
		case CtxLogger:
			seg = append(seg, arg.h.seg...)
			args = args[1:]
		case *Handler:
			seg = append(seg, arg.seg...)
			args = args[1:]
		default:
			seg = append(seg, slog.Any(missingKey, arg))
			args = args[1:]
		}
	}
	return
}

func scopeSegment(prefix string, seg []Attr) []Attr {
	if prefix == "" {
		return seg
	}

	pseg := make([]Attr, 0, len(seg))
	for _, a := range seg {
		pseg = append(pseg, Attr{
			Key:   prefix + a.Key,
			Value: a.Value,
		})
	}
	return pseg
}
