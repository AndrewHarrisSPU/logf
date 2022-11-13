package logf

import (
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

type Handler struct {
	seg       []Attr
	labels    string
	enc       slog.Handler
	replace   func(Attr) Attr
	addSource bool
}

// Enabled indicates whether a [Handler] is enabled for a given level
func (h *Handler) Enabled(level slog.Level) bool {
	return h.enc.Enabled(level)
}

// With extends the segment of [Attr]s associated with a [Handler]
func (h *Handler) WithAttrs(seg []Attr) slog.Handler {
	return h.withAttrs(seg).(*Handler)
}

// WithGroup opens a namespace. Every subsequent Attr key is prefixed with the name.
func (h *Handler) WithGroup(name string) slog.Handler {
	return h.withGroup(name).(*Handler)
}

// Handle performs interpolation on a [slog.Record]'s message
// The result is passed to another [slog.Handler]
func (h *Handler) Handle(r slog.Record) error {
	s := newSplicer()
	defer s.free()

	s.replace = h.replace
	s.scan(r.Message, nil)
	s.join(h.labels, h.seg, nil)
	s.ipol(r.Message)
	r.Message = s.line()

	return h.enc.Handle(r)
}

func (h *Handler) LogValue() slog.Value {
	return slog.GroupValue(h.seg...)
}

// handler minor ...

type handler interface {
	fmt(string, error, []any) (string, error)
	handle(slog.Level, string, error, int, []any) error
	withAttrs([]Attr) handler
	withGroup(string) handler
	enabled(slog.Level) bool
}

func (h *Handler) enabled(level slog.Level) bool {
	return h.Enabled(level)
}

func (h *Handler) withAttrs(seg []Attr) handler {
	scopedSeg := scopeSegment(h.labels, seg)

	return &Handler{
		seg:       concat(h.seg, scopedSeg),
		labels:    h.labels,
		enc:       h.enc.WithAttrs(seg),
		addSource: h.addSource,
		replace:   h.replace,
	}
}

func (h *Handler) withGroup(name string) handler {
	return &Handler{
		seg:       h.seg,
		labels:    h.labels + name + ".",
		enc:       h.enc.WithGroup(name),
		addSource: h.addSource,
		replace:   h.replace,
	}
}

func (h *Handler) attrs() []Attr {
	return h.seg
}

func (h *Handler) scope() string {
	return h.labels
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

	s.replace = h.replace
	s.join(h.labels, h.seg, s.scan(msg, args))
	s.ipol(msg)

	if err != nil {
		s.writeError(err)
	}

	if h.addSource {
		depth += 4
	} else {
		depth = 0
	}

	r := slog.NewRecord(time.Now(), level, s.line(), depth, nil)
	r.AddAttrs(s.export...)

	return h.enc.Handle(r)
}

func (h *Handler) fmt(
	msg string,
	err error,
	args []any,
) (string, error) {
	s := newSplicer()
	defer s.free()

	s.join(h.labels, h.seg, s.scan(msg, args))
	s.ipol(msg)

	if err != nil && len(msg) > 0 {
		s.writeString(": %w")
		err = fmt.Errorf(s.line(), err)
		msg = err.Error()
	} else {
		msg = s.line()
	}

	return msg, err
}

func scopeSegment(prefix string, seg []Attr) []Attr {
	if prefix == "" {
		return seg
	}

	pseg := make([]Attr, 0, len(seg))
	for _, a := range seg {
		pseg = append(pseg, Attr{
			Key:   prefix + a.Key,
			Value: a.Value,
		})
	}
	return pseg
}
