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
// Whiile any slog.Handler may be an used as an encoder, Handler is unaware of Attr segments held by its encoder.
type Handler struct {
	seg    []Attr
	prefix string

	ref       slog.Leveler
	enc       slog.Handler
	addSource bool
}

/*
	Need seg not to have prefixes
	But need to join prefiexed seg yuck wtf

	scopePrefix       string   // for text: prefix of scopes opened in preformatting
	scopes            []string // all scopes
	nOpenScopes       int      // the number of scopes opened in in preformattedAttrs
*/

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
func (h *Handler) With(seg []Attr) slog.Handler {
	return h.with(seg)
}

func (h *Handler) WithScope(name string) slog.Handler {
	return h.withScope(name)
}

func (h *Handler) with(seg []Attr) *Handler {
	pseg := scopeSegment(h.prefix, seg)

	return &Handler{
		seg:       concat(h.seg, pseg),
		ref:       h.ref,
		enc:       h.enc.With(seg),
		addSource: h.addSource,
	}
}

func (h *Handler) withScope(name string) *Handler {
	return &Handler{
		seg:       h.seg,
		prefix:    h.prefix + name + ".",
		ref:       h.ref,
		enc:       h.enc.WithScope(name),
		addSource: h.addSource,
	}
}

// Handle performs interpolation on a [slog.Record]'s message
// The result is passed to another [slog.Handler]
func (h *Handler) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.scan(r.Message(), nil)
	s.join(h.seg, nil, nil)

	r.Attrs(func(a Attr) {
		s.match(a)
	})

	var depth int
	if h.addSource {
		depth = 5
	}

	s.interpolate(r.Message())

	r2 := slog.NewRecord(r.Time(), r.Level(), s.msg(), depth)
	r.Attrs(func(a Attr) {
		r2.AddAttrs(a)
	})

	return h.enc.Handle(r2)
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
		s.appendError(err)
	}

	if h.addSource {
		depth += 5
	}

	r := slog.NewRecord(time.Now(), level, s.msg(), depth)
	r.AddAttrs(s.export...)
	// s.list.export(&r)

	return h.enc.Handle(r)
}
