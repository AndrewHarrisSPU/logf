package main

import (
	"flag"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

type gopher struct {
	log logf.Logger
	id  uint
	sum uint
}

func newGopher(log logf.Logger, i uint) gopher {
	return gopher{
		log: log.Label("gopher"),
		id:  i,
		sum: 0,
	}
}

func (g gopher) LogValue() slog.Value {
	return slog.GroupValue([]slog.Attr{
		slog.Uint64("id", uint64(g.id)),
		slog.Uint64("sum", uint64(g.sum)),
	}...)
}

func (g gopher) add(ns <-chan uint, sums chan<- uint) {
	go func() {
		for n := range ns {
			g.sum += n
			g.log.Level(logf.DEBUG).Msg("{id} {sum}", g)
		}
		g.log.Msg("{id}: {sum}", g)
		sums <- g.sum
	}()
}

var gophersN = flag.Uint("gophers", 10, "number of gophers")
var rangeN = flag.Uint("range", 101, "set end of summation range")
var verbosity = flag.Int("verbosity", 0, "set verbosity (lower is more verbose)")
var structured = flag.Bool("structured", false, "emit structure")

func main() {
	flag.Parse()

	// cfg := []logf.Option{
	// 	logf.Using.Minimal(true, *structured),
	// 	logf.Using.Level(slog.Level(*verbosity)),
	// }

	// log := logf.New(cfg...).With(".", "Eulerian Gophers")

	log := logf.New().Label("Eulerian Gophers")

	ns, sums := make(chan uint), make(chan uint)

	if *gophersN >= *rangeN {
		*gophersN = *rangeN
	}

	var i uint
	for i = 0; i < *gophersN; i++ {
		newGopher(log, i).add(ns, sums)
	}

	for i = 1; i < *rangeN; i++ {
		ns <- i
	}
	close(ns)

	var total uint
	for i = 0; i < *gophersN; i++ {
		total += <-sums
	}

	log.Level(logf.INFO+2).Msg("sum: {}", total)
}
