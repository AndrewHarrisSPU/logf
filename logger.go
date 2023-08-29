package logf

import (
	"log/slog"
)

// Logger embeds a [slog.Logger], and offers additional formatting methods:
//   - Leveled / formatting: [Logger.Debugf], [Logger.Infof], [Logger.Warnf], [Logger.Errorf]
//   - Formatting to a string or an error: [Logger.Fmt], [Logger.WrapErr]
//   - Logger tagging: [Logger.Tag]
//
// The following methods are available on a Logger by way of embedding:
//   - Leveled logging methods: [slog.Logger.Debug], [slog.Logger.Info], [slog.Logger.Warn], [slog.Logger.Error]
//   - General logging methods: [slog.Logger.Log], [slog.Logger.LogAttrs], [slog.Logger.LogDepth], [slog.Logger.LogAttrsDepth]
//   - [slog.Logger.Handler]
//
// The following methds are overriden to return [Logger]s rather than [*slog.Logger]s:
//   - [slog.Logger.With]
//   - [slog.Logger.WithGroup]
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

func newLogger(h handler) Logger {
	return Logger{slog.New(h)}
}

// See [slog.Logger.With]
func (l Logger) With(args ...any) Logger {
	return Logger{
		l.Logger.With(args...),
	}
}

// See [slog.Logger.WithGroup]
func (l Logger) WithGroup(name string) Logger {
	return Logger{
		l.Logger.WithGroup(name),
	}
}

func (l Logger) Log(level slog.Level, msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Logger.Log(nil, level, msg, args...)
}

// Debugf interpolates the msg string and logs at DEBUG.
func (l Logger) Debugf(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Debug(msg, args...)
}

// Infof interpolates the msg string and logs at INFO.
func (l Logger) Infof(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Info(msg, args...)
}

// Warnf interpolates the msg string and logs at WARN.
func (l Logger) Warnf(msg string, args ...any) {
	msg = logFmt(l, msg, args)
	l.Warn(msg, args...)
}

// Error is log slog.Error, but specifically asks for an error.
func (l Logger) Error(msg string, err error, args ...any) {
	args = append(args, slog.Any("err", err))
	l.Logger.Error(msg, args...)
}

// Errorf interpolates the msg string and logs at ERROR.
func (l Logger) Errorf(msg string, err error, args ...any) {
	args = append(args, slog.Any("err", err))
	msg = logFmt(l, msg, args)
	err = logFmtErr(l, msg, err, args)

	l.Logger.Error(msg, args...)
}

// Fmt interpolates the f string and returns the result.
func (l Logger) Fmt(f string, args ...any) string {
	return logFmt(l, f, args)
}

// WrapErr interpolates the f string, and returns an error.
// If geven a nil error, the resulting error.Error() string is the result of interpolating f.
// If given a non-nil error, the result includes the given error's string, and matches [errors.Is]/[errors.As] behavior, as with [fmt.Errorf]
func (l Logger) WrapErr(f string, err error, args ...any) error {
	return logFmtErr(l, f, err, args)
}
