package logf

import (
	"time"

	"golang.org/x/exp/slog"
)

type Handler struct {
	seg       []Attr
	ref       slog.Leveler
	enc       slog.Handler
	addSource bool
}

func NewHandler(options ...Option) *Handler {
	return newHandler(newConfig(options))
}

func newHandler(cfg *config) *Handler {
	return &Handler{
		seg:       make([]Attr, 0, 5),
		ref:       cfg.ref,
		enc:       cfg.h,
		addSource: cfg.addSource,
	}
}

func (h *Handler) Enabled(level slog.Level) bool {
	return h.ref.Level() <= level
}

func (h *Handler) With(seg []Attr) slog.Handler {
	return h.with(seg)
}

func (h *Handler) with(seg []Attr) *Handler {
	return &Handler{
		seg:       concat(h.seg, seg),
		ref:       h.ref,
		enc:       h.enc.With(seg),
		addSource: h.addSource,
	}
}

func (h *Handler) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.join(nil, h.seg, nil)
	r.Attrs(func(a Attr) {
		s.list.insert(a)
	})

	return h.handle(s, r.Level(), r.Message(), nil, 1)
}

func (h *Handler) handle(
	s *splicer,
	level slog.Level,
	msg string,
	err error,
	depth int,
) error {
	s.interpolate(msg)

	if err != nil {
		s.text.appendError(err)
	}

	if h.addSource {
		depth += 5
	}

	r := slog.NewRecord(time.Now(), level, s.freeze(), depth)
	s.list.export(&r)

	return h.enc.Handle(r)
}
