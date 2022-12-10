package logf

import (
	// "time"

	"golang.org/x/exp/slog"
)

// handler minor
// fmt allows the Logger.Fmt method
// handle is oriented towards a later-binding Record contruction
type handler interface {
	slog.Handler
	slog.LogValuer
	group() Attr
	withTag(string) handler
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

// A Grouper is like [slog.LogValuer] in that a Grouper produces structure when expanded.
// Unlike a [slog.LogValuer] that expands to a group of [Attr]s, the key associated with a Grouper is set by the Grouper.
type loggingValuer interface {
	group() Attr
}

func (h *Handler) group() Attr {
	return slog.Group("", h.attrs...)
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

func (h *Handler) Handle(r slog.Record) error {
	return h.enc.Handle(r)
}
