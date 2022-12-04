package logf

import (
	"time"

	"golang.org/x/exp/slog"
)

// handler minor
// fmt allows the Logger.Fmt method
// handle is oriented towards a later-binding Record contruction
type handler interface {
	slog.Handler
	slog.LogValuer
	withTag(string) handler
	fmt(string, []any) *splicer
	handle(*splicer, slog.Level, string, error, int, []any) error
}

// Handler encapsulates a [slog.Handler] and maintains additional state required for message interpolation.
type Handler struct {
	tag   Attr
	attrs []Attr

	scope string
	enc   slog.Handler

	replace   func(Attr) Attr
	addSource bool
}

// LogValue returns a [slog.Value], of [slog.GroupKind].
// The group of [Attr]s is the collection of attributes present in log lines handled by the [Handler].
func (h *Handler) LogValue() Value {
	return slog.GroupValue(h.attrs...)
}

// SlogHandler returns the [slog.Handler] encapsulated by a [Handler]
func (h *Handler) SlogHandler() slog.Handler {
	return h.enc
}

// See [slog.Handler.Enabled].
func (h *Handler) Enabled(level Level) bool {
	return h.enc.Enabled(level)
}

// See [slog.Handler.WithAttrs].
func (h *Handler) WithAttrs(as []Attr) slog.Handler {
	scopedSeg := scopeAttrs(h.scope, as, h.replace)

	return &Handler{
		tag:       h.tag,
		attrs:     concat(h.attrs, scopedSeg),
		scope:     h.scope,
		enc:       h.enc.WithAttrs(as),
		replace:   h.replace,
		addSource: h.addSource,
	}
}

// See [slog.Handler.WithGroup].
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		tag:       h.tag,
		attrs:     h.attrs,
		scope:     h.scope + name + ".",
		enc:       h.enc.WithGroup(name),
		replace:   h.replace,
		addSource: h.addSource,
	}
}

func (h *Handler) withTag(tag string) handler {
	return &Handler{
		tag:       slog.String("#", tag),
		attrs:     h.attrs,
		scope:     h.scope,
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
	s.joinOne("", h.tag, nil)
	s.join(h.scope, h.attrs, nil, h.replace)
	s.ipol(r.Message)
	r.Message = s.line()

	return h.enc.Handle(r)
}

func (h *Handler) handle(
	s *splicer,
	level slog.Level,
	msg string,
	err error,
	depth int,
	args []any,
) error {
	defer s.free()

	if h.tag.Key != "" {
		s.joinOne("", h.tag, nil)
	}
	s.join(h.scope, h.attrs, args, h.replace)

	if s.ipol(msg) {
		if err != nil {
			s.writeError(err)
		}
		msg = s.line()
	} else if err != nil {
		s.WriteString(msg)
		s.WriteString(": ")
		s.WriteString(err.Error())
		msg = s.line()
	}

	if h.addSource {
		depth += 4
	} else {
		depth = 0
	}

	r := slog.NewRecord(time.Now(), level, msg, depth, nil)
	r.AddAttrs(s.export...)

	return h.enc.Handle(r)
}

func (h *Handler) fmt( msg string, args []any) *splicer {
	s := newSplicer()

	s.join(h.scope, h.attrs, s.scan(msg, args), h.replace)
	s.ipol(msg)

	return s
}