package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

var space string = "     "
var beams string = " *~- "
var width int = len(beams) * 10

func main() {
	h := spinHandler{start: time.Now()}
	log := logf.New(logf.Using.Handler(&h)).With("Scully", "👩‍🦰")
	ufo := errors.New("🛸")

	done := make(chan struct{})

	go func() {
		for i := 0; i < width; i++ {
			lpad := strings.Repeat(space, 10)[:i]
			rpad := strings.Repeat(beams, 10)[:width-i]
			<-time.NewTimer(60 * time.Millisecond).C
			log.Err("{}{Scully}{}", ufo, lpad, rpad)
		}
		done <- struct{}{}
	}()

	go func() {
		for {
			<-time.NewTimer(400 * time.Millisecond).C
			log.Level(logf.INFO+1).Msg("{}: oh no! {Scully}!", "👦🏻")
		}
	}()

	<-done
	log.Err("{Scully} was abducted", ufo)
}

const (
	xStore     = "\033[s"
	xLoad      = "\033[u"
	xLineClear = "\033[K"
)

type spinHandler struct {
	mu            sync.Mutex
	start         time.Time
	level         slog.Level
	clearNextLine bool
}

func (h *spinHandler) Enabled(level slog.Level) bool {
	return h.level >= level
}

func (h *spinHandler) With([]slog.Attr) slog.Handler {
	return h
}

func (h *spinHandler) Handle(r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clearLine()
	h.write(xStore)

	if r.Level() <= h.level {
		h.clearNextLine = true
	}

	h.elapsed()

	h.write(" ")
	h.write(r.Message())

	h.write("\n")
	return nil
}

func (h *spinHandler) write(s string) {
	io.WriteString(os.Stdout, s)
}

func (h *spinHandler) elapsed() {
	d := time.Since(h.start).Round(time.Millisecond).String()
	h.write(fmt.Sprintf("%-8s", d))
}

func (h *spinHandler) clearLine() {
	if h.clearNextLine {
		h.write(xLoad)
		h.write(xLineClear)
		h.write(xLoad)
		h.clearNextLine = false
	}
}
