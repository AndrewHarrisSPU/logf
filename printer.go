package logf

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/exp/slog"
)

// COLORS

const (
	cOpen  = "\x1b["
	cClose = "\x1b[0m"
	cDim   = "2m"
)

var pkgPrinter printer

// Print configuration
var Print struct {
	Time struct {
		None    bool
		Layout  string
		Start   time.Time
		Elapsed bool
	}
	Level  slog.Leveler
	Attrs  bool
	Colors bool
	Source bool
}

type printer struct{}

func init() {
	Print.Time.None = false
	Print.Time.Layout = "03:04:05"
	Print.Time.Start = time.Now()
	Print.Time.Elapsed = true

	Print.Level = INFO

	Print.Attrs = true

	Print.Colors = true

	Print.Source = true
}

func (p *printer) print(s *splicer, msg string, depth int, seg []Attr) error {
	if !Print.Time.None {
		p.printTime(s, time.Now())
		s.writeByte(' ')
	}

	if Print.Source {
		p.printSource(s)
	}

	if Print.Attrs && !Print.Colors {
		s.writeByte('"')
	}

	s.interpolate(msg)

	if Print.Attrs && !Print.Colors {
		s.writeByte('"')
	}

	if Print.Attrs {
		s.writeString(cOpen + cDim)
		p.printSegment(s, seg)
		p.printSegment(s, s.export)
		s.writeString(cClose)
	}

	println(s.msg())
	return nil
}

func (p *printer) printTime(s *splicer, t time.Time) {
	s.writeString(cOpen + cDim)
	defer s.writeString(cClose)

	if Print.Time.Elapsed {
		d := time.Since(Print.Time.Start).Round(time.Millisecond)
		s.text = appendDuration(s.text, d)
	} else {
		s.writeTimeVerb(t, Print.Time.Layout)
	}
}

func (p *printer) printSegment(s *splicer, seg []Attr) {
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

func (p *printer) printString(s *splicer, str string) {
	if len(str) > 0 {
		s.writeString(str)
	}
}

func (p *printer) printSource(s *splicer) {
	s.writeString(cOpen + cDim)
	defer s.writeString(cClose)

	u := [1]uintptr{}
	runtime.Callers(4, u[:])
	pc := u[0]

	src, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	dir, file := filepath.Split(src.File)
	s.writeString(filepath.Base(dir))
	s.writeRune(os.PathSeparator)
	s.writeString(file)
	s.writeByte(':')
	s.text = strconv.AppendInt(s.text, int64(src.Line), 10)

	s.writeByte(' ')
}
