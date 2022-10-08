package logf

import (
	"io"
	"os"
	"time"

	"golang.org/x/exp/slog"
)

type Handler struct {
	// replacement hack
	rep 	  func(Attr) Attr
	repstr 	  *string

	seg       []Attr
	ref       slog.Leveler
	enc       slog.Handler
	addSource bool
}

func NewHandler(options ...Option) *Handler {
	return newHandler(options...)
}

func newHandler(options ...Option) *Handler {
	// CONFIG PART
	cfg := new(config)

	// These depend on other configurations,
	// so evaluation is delayed
	var oSlog Option = option[slog.TextHandler](usingText)
	var oHandler option[slog.Handler]

	for _, o := range options {
		switch o := o.(type) {
		case option[slog.TextHandler], option[slog.JSONHandler]:
			oSlog = o
		case option[slog.Handler]:
			oHandler = o
		case option[io.Writer]:
			o(cfg)
		case option[slog.Leveler]:
			o(cfg)
		case option[source]:
			o(cfg)
		default:
			panic("unknown option type")
		}
	}

	if cfg.ref == nil {
		cfg.ref = slog.InfoLevel
	}

	if cfg.w == nil {
		cfg.w = os.Stdout
	}

	// HANDLER PART
	h := &Handler{
		repstr:    new(string),
		seg:       make([]Attr, 0),
		ref:       cfg.ref,
		addSource: cfg.addSource,
	}

	h.rep = func(a Attr) Attr {
		if a.Key == "msg" {
			return slog.String( "msg", *h.repstr )
		}
		return a
	}

	if oHandler != nil {
		oHandler(cfg)
		h.enc = cfg.h
	} else {
		// build a slog Handler
		scfg := slog.HandlerOptions{
			Level:     cfg.ref,
			AddSource: cfg.addSource,
			ReplaceAttr: h.rep,
		}

		switch oSlog.(type) {
		case option[slog.JSONHandler]:
			h.enc = scfg.NewJSONHandler(cfg.w)
		case option[slog.TextHandler]:
			h.enc = scfg.NewTextHandler(cfg.w)
		}
	}

	return h
}

func (h *Handler) Enabled(level slog.Level) bool {
	return h.ref.Level() <= level
}

func (h *Handler) With(seg []Attr) slog.Handler {
	return h.with(seg)
}

func (h *Handler) with(seg []Attr) *Handler {
	return &Handler{
		repstr:	   h.repstr,
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
	s.interpolate(r.Message())	
	*h.repstr = s.freeze()

	return h.enc.Handle(r)
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

	*h.repstr = s.freeze()

	r := slog.NewRecord(time.Now(), level, *h.repstr, depth)
	s.list.export(&r)

	return h.enc.Handle(r)
}
