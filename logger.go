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

func New(options ...Option) Logger {
	return Logger{
		h:     newHandler(options...),
		level: slog.InfoLevel,
	}
}

func (l Logger) Level(level slog.Leveler) Logger {
	return Logger{
		h:     l.h,
		level: level,
	}
}

func (l Logger) With(args ...any) Logger {
	return Logger{
		h:     l.h.with(segment(args)),
		level: l.level,
	}
}

// LOGGING METHODS

func (l Logger) Msg(msg string, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(nil, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, nil, 0)
}

func (l Logger) Err(msg string, err error, args ...any) {
	if l.level.Level() < l.h.ref.Level() {
		return
	}

	s := newSplicer()
	defer s.free()

	s.join(nil, l.h.seg, args)
	l.h.handle(s, l.level.Level(), msg, err, 0)
}

func (l Logger) Fmt(msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	s.join(nil, l.h.seg, args)
	s.interpolate(msg)

	if err != nil && len(s.text) > 0 {
		err = fmt.Errorf(string(s.text)+": %w", err)
	}

	s.text.appendError(err)

	return string(s.text), err
}

// SEGMENT

func segment(args []any) (seg []Attr) {
	for len(args) > 0 {
		switch arg := args[0].(type) {
		case string:
			if len(args) == 1 {
				seg = append(seg, slog.String(missingKey, arg))
				return
			}
			seg = append(seg, slog.Any(arg, args[1]))
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
