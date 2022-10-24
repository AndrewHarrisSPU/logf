package logf

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"
)

type Logger struct {
	h     handler
	level slog.Level
	depth int
}

// CONSTRUCTION

// Pass options to New, get a Logger!
func New(options ...Option) Logger {
	cfg := makeConfig(options...)

	var l Logger

	if cfg.usePrinter {
		l.h = newPrinter()
	} else {
		l.h = newHandler(cfg)
	}

	return l
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
	l.h = l.h.withAttrs(Segment(args...))
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
	if l.level < l.h.level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), nil, args)
	l.h.handle(s, l.level.Level(), msg, nil, l.depth)
}

// Err logs a message, appending the error string to the message text.
func (l Logger) Err(msg string, err error, args ...any) {
	if l.level < l.h.level() {
		return
	}

	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), nil, args)
	l.h.handle(s, l.level.Level(), msg, err, l.depth)
}

// Fmt interpolates like [Logger.Msg] or [Logger.Err].
// The result is not written to a log, but returned.
// The returned string is the interpolation of msg.
// With a nil error, Fmt emits a nil error.
// Otherwise, the returned error stringifies to the returned string.
// but is wrapped with [fmt.Errorf] (preserving [errors.Is], [errors.As] behavior).
func (l Logger) Fmt(msg string, err error, args ...any) (string, error) {
	s := newSplicer()
	defer s.free()

	args = s.scan(msg, args)
	s.join(l.h.attrs(), nil, args)
	s.interpolate(msg)

	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.msg(), err)
		msg = err.Error()
	} else {
		msg = s.msg()
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
		case slog.LogValuer:
			v := arg.LogValue()
			if v.Kind() == slog.GroupKind {
				seg = append(seg, v.Group()...)
			}
			args = args[1:]
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
		case error:
			seg = append(seg, slog.String("err", arg.Error()))
			args = args[1:]
		case Logger:
			seg = append(seg, arg.h.attrs()...)
			args = args[1:]
		case LoggerCtx:
			seg = append(seg, arg.h.attrs()...)
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
