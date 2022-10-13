package logf

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// minimal is an encoder.
// It writes Records to an io.Writer.
type minimal struct {
	w  io.Writer
	mu *sync.Mutex

	// Handler state
	ref slog.Leveler
	seg []Attr

	// configuration
	layout    string
	start     time.Time
	zeroTime  bool
	elapsed   bool
	export    bool
	addSource bool
}

func usingMinimal(elapsed, export bool) option[slog.Handler] {
	m := new(minimal)

	m.mu = new(sync.Mutex)
	m.layout = time.Kitchen
	m.elapsed = elapsed
	m.export = export
	m.start = time.Now()

	return func(cfg *config) {
		m.w = cfg.w
		m.ref = cfg.ref
		m.addSource = cfg.addSource

		cfg.h = m
	}
}

func (m *minimal) Enabled(level slog.Level) bool {
	return m.ref.Level() <= level
}

func (m *minimal) With(seg []Attr) slog.Handler {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &minimal{
		w:   m.w,
		mu:  m.mu,
		seg: concat(m.seg, seg),

		layout:    m.layout,
		start:     m.start,
		zeroTime:  m.zeroTime,
		elapsed:   m.elapsed,
		export:    m.export,
		addSource: m.addSource,
	}
}

// TODO
func (m *minimal) WithScope(name string) slog.Handler {
	return &minimal{
		w:   m.w,
		mu:  m.mu,
		seg: m.seg,

		layout:    m.layout,
		start:     m.start,
		zeroTime:  m.zeroTime,
		elapsed:   m.elapsed,
		export:    m.export,
		addSource: m.addSource,
	}
}

func (m *minimal) Handle(r slog.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := r.Message()

	// elide zero content messages
	if len(msg) == 0 && !m.export && !m.addSource {
		return nil
	}

	m.writeTime(r)
	m.writeMessage(msg)
	m.writeSource(r)
	if m.export {
		m.writeSegment()
		r.Attrs(m.writeAttr)
	}
	m.writeNewline()

	return nil
}

func (m *minimal) writeTime(r slog.Record) {
	switch {
	case m.zeroTime:
		return
	case m.elapsed:
		s := fmt.Sprintf("%-6s", time.Since(m.start).Round(time.Millisecond).String())
		io.WriteString(m.w, s)
	default:
		io.WriteString(m.w, r.Time().Format(m.layout))
	}
}

func (m *minimal) writeMessage(msg string) {
	if len(msg) > 0 {
		m.writeSpace()
		io.WriteString(m.w, msg)
	}
}

func (m *minimal) writeSource(r slog.Record) {
	if !m.addSource {
		return
	}
	file, line := r.SourceLine()
	if len(file) == 0 {
		return
	}
	src := fmt.Sprintf("%s:%d", file, line)

	m.writeSpace()
	io.WriteString(m.w, src)
}

func (m *minimal) writeSegment() {
	for _, a := range m.seg {
		m.writeSpace()
		io.WriteString(m.w, a.String())
	}
}

func (m *minimal) writeAttr(a Attr) {
	m.writeSpace()
	io.WriteString(m.w, a.String())
}

func (m *minimal) writeSpace() {
	m.w.Write([]byte(" "))
}

func (m *minimal) writeNewline() {
	m.w.Write([]byte("\n"))
}
