package main

import (
	"flag"
	"io"
	"math/rand"
	"time"

	"github.com/AndrewHarrisSPU/logf"
	// "golang.org/x/exp/slog"
)

type gopher struct {
	log logf.Logger
	id  uint
	sum uint
}

func newGopher(log logf.Logger, i uint) gopher {
	return gopher{
		log: log.Label("gopher").With("id", i),
		id:  i,
		sum: 0,
	}
}

// func (g gopher) LogValue() slog.Value {
// 	return slog.Uint64("sum", uint64(g.sum))
// }

func (g gopher) add(ns <-chan uint, sums chan<- uint) {
	go func() {
		for n := range ns {
			g.sum += n
			g.log.Level(logf.DEBUG+1).Msg("", "sum", g.sum)
		}
		g.log.Msg("done")
		sums <- g.sum
	}()
}

var gophersN = flag.Uint("gophers", 10, "number of gophers")
var rangeN = flag.Uint("range", 101, "set end of summation range")

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Parse()

	tty := logf.New().
		Level(logf.INFO+1).
		Elapsed(true).
		Spin(logf.DEBUG, 5).
		TTY()

	log := tty.Logger()

	ns, sums := make(chan uint), make(chan uint)

	if *gophersN >= *rangeN {
		*gophersN = *rangeN
	}

	var i uint
	for i = 0; i < *gophersN; i++ {
		newGopher(log, i).add(ns, sums)
	}

	for i = 1; i < *rangeN; i++ {
		<-time.NewTimer(time.Millisecond * 100).C
		if i%9 == 0 {
			log.Level(logf.INFO).Msg("fizz squared")
		}
		if i%25 == 0 {
			io.WriteString(tty, "buzz squared\n")
		}
		ns <- i
	}
	close(ns)

	var total uint
	for i = 0; i < *gophersN; i++ {
		total += <-sums
	}

	log.Level(logf.INFO+1).Msg("{}", total)

	tty.Write(nil)
}
