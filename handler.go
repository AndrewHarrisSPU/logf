package logf

import (
	"context"
	"log/slog"
)

// handler minor
// fmt allows the Logger.Fmt method
// handle is oriented towards a later-binding Record contruction
type handler interface {
	slog.Handler
	slog.LogValuer
}

type Handler struct {
	enc   slog.Handler
	store Store

	label     Attr
	replace   replaceFunc
	addSource bool
}

func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.enc.Enabled(ctx, l)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return h.enc.Handle(ctx, r)
}

func (h *Handler) WithAttrs(as []Attr) slog.Handler {
	h2 := &Handler{
		enc:       h.enc.WithAttrs(as),
		store:     h.store.WithAttrs(as),
		replace:   h.replace,
		addSource: h.addSource,
	}
	_, h2.label = detectLabel(as, h.label)

	return h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		enc:       h.enc.WithGroup(name),
		store:     h.store.WithGroup(name),
		label:     h.label,
		replace:   h.replace,
		addSource: h.addSource,
	}
}

// iterates out through stored handlerFrames, LIFO
func (h *Handler) LogValue() Value {
	return h.store.LogValue()
}
