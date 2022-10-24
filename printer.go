package logf

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// COLORS

const (
	cOpen      = "\x1b["
	cClose     = "\x1b[0m"
	cDim       = "2m"
	cBold      = "1m"
	cBoldGreen = "32;1m"
	cGreen     = "32m"
	cBlue      = "36;1m"
)

// Print configuration
var Print struct {
	Time struct {
		None    bool
		Layout  string
		Start   time.Time
		Elapsed bool
	}
	Level  slog.Leveler
	Tag    bool
	Attrs  bool
	Colors bool
	Source bool
}

func init() {
	Print.Time.None = false
	Print.Time.Layout = "15:04:05"
	Print.Time.Start = time.Now()
	Print.Time.Elapsed = false

	Print.Level = INFO

	Print.Tag = true

	Print.Attrs = true

	Print.Colors = true

	Print.Source = true
}

type printer struct {
	mu     *sync.Mutex
	seg    []Attr
	prefix string
	tag    string
}

func newPrinter() *printer {
	return &printer{
		mu: new(sync.Mutex),
	}
}

func (p *printer) withAttrs(seg []Attr) handler {
	scoped := scopeSegment(p.prefix, seg)

	return &printer{
		mu:     p.mu,
		seg:    concat(p.seg, scoped),
		prefix: p.prefix,
		tag:    p.tag,
	}
}

func (p *printer) withGroup(name string) handler {
	return &printer{
		mu:     p.mu,
		seg:    p.seg,
		prefix: name + ".",
		tag:    name,
	}
}

func (p *printer) attrs() []Attr {
	return p.seg
}

func (p *printer) level() slog.Level {
	return INFO
}

func (p *printer) handle(
	s *splicer,
	level slog.Level,
	msg string,
	err error,
	depth int,
) error {
	if !Print.Time.None {
		p.printTime(s, time.Now())
		s.writeByte(' ')
	}

	if Print.Tag {
		p.printTag(s, p.tag)
	}

	if Print.Attrs && !Print.Colors {
		s.writeByte('"')
	}

	if Print.Colors {
		s.writeString(cOpen + cBold)
	}

	s.interpolate(msg)

	if Print.Colors {
		s.writeString(cClose)
	}

	if Print.Attrs && !Print.Colors {
		s.writeByte('"')
	}

	if Print.Attrs {
		p.printSegment(s, p.seg)
		p.printSegment(s, s.export)
	}

	if Print.Source {
		p.printSource(s, depth)
	}

	s.writeByte('\n')

	p.mu.Lock()
	defer p.mu.Unlock()

	os.Stdout.WriteString(s.msg())
	return nil
}

func (printer) printTime(s *splicer, t time.Time) {
	s.writeString(cOpen + cDim)
	defer s.writeString(cClose)

	if Print.Time.Elapsed {
		d := time.Since(Print.Time.Start).Round(time.Millisecond)
		s.text = appendDuration(s.text, d)
	} else {
		s.writeTimeVerb(t, Print.Time.Layout)
	}
}

func (printer) printTag(s *splicer, tag string) {
	if Print.Colors {
		s.writeString(cOpen + cBoldGreen)
		defer s.writeString(cClose)
	}

	if len(tag) > 0 {
		s.writeString(tag)
		s.writeByte(' ')
	}
}

func (printer) printSegment(s *splicer, seg []Attr) {
	if Print.Colors {
		s.writeString(cOpen + cGreen)
		defer s.writeString(cClose)
	}

	for _, a := range seg {
		if len(a.Key) == 0 {
			continue
		}

		s.writeByte(' ')
		s.writeString(a.Key)
		s.writeByte('=')
		s.writeValueNoVerb(a.Value)
	}
}

func (printer) printString(s *splicer, str string) {
	if len(str) > 0 {
		s.writeString(str)
	}
}

func (printer) printSource(s *splicer, depth int) {
	s.writeByte(' ')

	if Print.Colors {
		s.writeString(cOpen + cDim)
		defer s.writeString(cClose)
	}

	// yank source from runtime
	u := [1]uintptr{}
	runtime.Callers(4+depth, u[:])
	pc := u[0]
	src, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	// print file/line
	s.writeString(src.File)
	s.writeByte(':')
	s.text = strconv.AppendInt(s.text, int64(src.Line), 10)
}
