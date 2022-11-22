package logf

import (
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

// handler minor
// fmt allows the Logger.Fmt method
// handle is oriented towards a later-binding Record contruction
type handler interface {
	slog.Handler
	slog.LogValuer
	withLabel(string) handler
	fmt(string, error, []any) (string, error)
	handle(slog.Level, string, error, int, []any) error
}

// Handler encapsulates a [slog.Handler] and maintains additional state required for message interpolation.
type Handler struct {
	attrs     []Attr
	scope     string
	label     Attr
	enc       slog.Handler
	replace   func(Attr) Attr
	addSource bool
}

// LogValue returns a [slog.Value], of [slog.GroupKind].
// The group of [Attr]s is the collection of attributes present in log lines handled by the [Handler].
func (h *Handler) LogValue() slog.Value {
	return slog.GroupValue(h.attrs...)
}

// Encoder returns the [slog.Handler] encapsulated by a [Handler]
func (h *Handler) Encoder() slog.Handler {
	return h.enc
}

// See [slog.Handler.Enabled].
func (h *Handler) Enabled(level slog.Level) bool {
	return h.enc.Enabled(level)
}

// See [slog.Handler.WithAttrs].
func (h *Handler) WithAttrs(as []Attr) slog.Handler {
	scopedSeg := scopeAttrs(h.scope, as, h.replace)

	return &Handler{
		attrs:     concat(h.attrs, scopedSeg),
		scope:     h.scope,
		label:     h.label,
		enc:       h.enc.WithAttrs(as),
		replace:   h.replace,
		addSource: h.addSource,
	}
}

// See [slog.Handler.WithGroup].
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		attrs:     h.attrs,
		scope:     h.scope + name + ".",
		label:     h.label,
		enc:       h.enc.WithGroup(name),
		replace:   h.replace,
		addSource: h.addSource,
	}
}

func (h *Handler) withLabel(label string) handler {
	return &Handler{
		attrs:     h.attrs,
		scope:     h.scope,
		label:     slog.String(labelKey, label),
		enc:       h.enc,
		replace:   h.replace,
		addSource: h.addSource,
	}
}

// Handle performs interpolation on a [slog.Record] message.
// The record is then handled by an encapsulated [slog.Handler].
func (h *Handler) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.scan(r.Message, nil)
	s.join(h.scope, h.attrs, nil, h.replace)
	s.ipol(r.Message)
	r.Message = s.line()

	return h.enc.Handle(r)
}

func (h *Handler) handle(
	level slog.Level,
	msg string,
	err error,
	depth int,
	args []any,
) error {
	s := newSplicer()
	defer s.free()

	s.join(h.scope, h.attrs, s.scan(msg, args), h.replace)
	if s.ipol(msg) {
		if err != nil {
			s.writeError(err)
		}
		msg = s.line()
	} else if err != nil {
		s.writeString(msg)
		s.writeString(": ")
		s.writeString(err.Error())
		msg = s.line()
	}

	if h.addSource {
		depth += 4
	} else {
		depth = 0
	}

	r := slog.NewRecord(time.Now(), level, msg, depth, nil)
	if h.label != noLabel {
		r.AddAttrs(h.label)
	}

	r.AddAttrs(s.export...)

	return h.enc.Handle(r)
}

func (h *Handler) fmt(
	msg string,
	err error,
	args []any,
) (string, error) {
	// shortcut: no msg, err -> return labled error string, err
	if err != nil && len(msg) == 0 && h.label != noLabel {
		err = fmt.Errorf("%s: %w", h.label, err)
		msg = err.Error()
		return msg, err
	}

	// shortcut: no err, no msg -> return
	if err == nil && len(msg) == 0 {
		return msg, err
	}

	// interpolate...
	s := newSplicer()
	defer s.free()

	s.join(h.scope, h.attrs, s.scan(msg, args), h.replace)
	s.ipol(msg)

	// err -> return error string, err
	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.line(), err)
		msg = err.Error()
		return msg, err
	}

	// no err -> return msg string, nil
	if len(msg) > 0 {
		msg = s.line()
		return msg, err
	}

	return msg, err
}
