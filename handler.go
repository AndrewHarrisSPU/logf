package logf

import (
	"time"

	"golang.org/x/exp/slog"
)

// Handler satisfies the [slog.Handler] interface.
//
// When used with a logf [Logger], {keyed} and unkeyed {} interpolation tokens are handled.
// When used with another logger API (e.g., [slog.Logger]) only {keyed} interpolation is possible.
//
// Handler is not an encoder, and forwards records to aonther slog.Handler for that purpose.
type Handler struct {
	seg    []Attr
	prefix string

	ref       slog.Leveler
	enc       slog.Handler
	addSource bool
}

// NewHandler constructs a new Handler from a list of Options
// Options are available in the package variable [Using].
func NewHandler(options ...Option) *Handler {
	return newHandler(makeConfig(options...))
}

func newHandler(cfg config) *Handler {
	h := &Handler{
		seg:       make([]Attr, 0),
		ref:       cfg.ref,
		enc:       cfg.h,
		addSource: cfg.addSource,
	}
	return h
}

// Enabled indicates whether a [Handler] is enabled for a given level
func (h *Handler) Enabled(level slog.Level) bool {
	return h.ref.Level() <= level
}

// With extends the segment of [Attr]s associated with a [Handler]
func (h *Handler) WithAttrs(seg []Attr) slog.Handler {
	return h.withAttrs(seg).(*Handler)
}

// WithScope opens a namespace. Every subsequent Attr key is prefixed with the name.
func (h *Handler) WithGroup(name string) slog.Handler {
	return h.withGroup(name).(*Handler)
}

// Handle performs interpolation on a [slog.Record]'s message
// The result is passed to another [slog.Handler]
func (h *Handler) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.scan(r.Message, nil)
	s.join(h.seg, nil, nil)
	s.interpolate(r.Message)
	r.Message = s.msg()

	return h.enc.Handle(r)
}

// handler minor ...

type handler interface {
	handle(*splicer, slog.Level, string, error, int) error
	withAttrs([]Attr) handler
	withGroup(string) handler
	attrs() []Attr
	level() slog.Level
}

func (h *Handler) withAttrs(seg []Attr) handler {
	scopedSeg := scopeSegment(h.prefix, seg)

	return &Handler{
		seg:       concat(h.seg, scopedSeg),
		prefix:    h.prefix,
		ref:       h.ref,
		enc:       h.enc.WithAttrs(seg),
		addSource: h.addSource,
	}
}

func (h *Handler) withGroup(name string) handler {
	return &Handler{
		seg:       h.seg,
		prefix:    h.prefix + name + ".",
		ref:       h.ref,
		enc:       h.enc.WithGroup(name),
		addSource: h.addSource,
	}
}

func (h *Handler) attrs() []Attr {
	return h.seg
}

func (h *Handler) level() slog.Level {
	return h.ref.Level()
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
		s.writeError(err)
	}

	if h.addSource {
		depth += 4
	} else {
		depth = 0
	}

	r := slog.NewRecord(time.Now(), level, s.msg(), depth, nil)
	r.AddAttrs(s.export...)

	return h.enc.Handle(r)
}
