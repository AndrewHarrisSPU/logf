package logf

import (
	"context"

	"golang.org/x/exp/slog"
)

type Logger struct {
	*slog.Logger
}

// UsingHandler returns a Logger employing the given slog.Handler
//
// If the given handler is not of a type native to logf, a new [Handler] is constructed, encapsulating the given handler.
func UsingHandler(h slog.Handler) Logger {
	if h, isLogfHandler := h.(handler); isLogfHandler {
		return newLogger(h)
	}

	lh := &Handler{
		enc:       h,
		addSource: true,
	}

	return newLogger(lh)
}

// FromContext employs [slog.FromContext] to obtain a Logger from the given context.
// Precisely, the function returns the result of:
//
//	UsingHandler(slog.FromContext(ctx).Handler())
func FromContext(ctx context.Context) Logger {
	return UsingHandler(slog.FromContext(ctx).Handler())
}

func newLogger(h handler) Logger {
	return Logger{slog.New(h)}
}

// With appends attributes held in a [Logger]'s handler.
// Arguments are converted to attributes with [Attrs].
func (l Logger) With(args ...any) Logger {
	return Logger{
		l.Logger.With(args...),
	}
}

func (l Logger) WithGroup(name string) Logger {
	return Logger{
		l.Logger.WithGroup(name),
	}
}

func (l Logger) WithContext(ctx context.Context) Logger {
	return Logger{
		l.Logger.WithContext(ctx),
	}
}

func (l Logger) Tag(name string) Logger {
	h, ok := l.Handler().(handler)
	if !ok {
		return l
	}

	return newLogger(h.withTag(name))
}

func (l Logger) Debugf(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Debug(msg, args...)
}

func (l Logger) Infof(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Info(msg, args...)
}

func (l Logger) Warnf(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Warn(msg, args...)
}

func (l Logger) Errorf(msg string, err error, args ...any) {
	err = logFmtErr(l, msg, err, args)
	l.Error("", err, args...)
}

func (l Logger) Fmt(msg string, args ...any) string {
	return logFmt(l, msg, args)
}

func (l Logger) WrapErr(msg string, err error, args ...any) error {
	return logFmtErr(l, msg, err, args)
}

/*

type Logger struct {
	internal
}

type internal struct {
	*slog.Logger
}

func (l Logger) h() handler {
	return l.Handler().(handler)
}


// UsingHandler returns a Logger employing the given slog.Handler
//
// If the given handler is not of a type native to logf, a new [Handler] is constructed, encapsulating the given handler.
func UsingHandler(h slog.Handler) Logger {
	if h, isLogfHandler := h.(handler); isLogfHandler {
		return newLogger(h)
	}

	lh := &Handler{
		enc:       h,
		addSource: true,
	}

	return newLogger(lh)
}

func newLogger(h handler) Logger {
	return Logger{internal{slog.New(h)}}
}



// Group calls [slog.Logger.WithGroup] on a [Logger]'s handler.
func (l Logger) WithGroup(name string) Logger {
	h := l.h().WithGroup(name).(handler)
	return newLogger(h)
}

func (l Logger) WithContext(ctx context.Context) Logger {
	sl := l.internal.WithContext(ctx)
	return Logger{internal{sl}}
}

// Tag configures a tag that appears in [TTY] log output.
// Tags set by this method override; only one is set per logger.
func (l Logger) Tag(tag string) Logger {
	h := l.h().withTag(tag)
	return newLogger(h)
}

// LogValue returns the set of [Attr]s accrued by the Logger's handler.
func (l Logger) LogValue() slog.Value {
	return l.h().LogValue()
}
/*
// Fmt applies interpolation to the given message string.
// The resulting string is returned, rather than logged.
func (l Logger) Fmt(msg string, args ...any) string {
	s := l.h().ipol(msg, args)
	defer s.free()

	return s.line()
}

// NewErr applies interpolation and formatting to wrap an error.
// If the given err is nil, a new error is constructed.
// The resulting error is returned, rather than logged.
func (l Logger) NewErr(msg string, err error, args ...any) error {
	s := l.h().ipol(msg, args)
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
*/