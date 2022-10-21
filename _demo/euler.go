package main

import (
	"flag"

	"github.com/AndrewHarrisSPU/logf"
	// "golang.org/x/exp/slog"
)

type gopher struct {
	log logf.Logger
	sum uint
}

func newGopher(log logf.Logger, i uint) gopher {
	return gopher{
		log: log.With(".", "gopher", "id", i),
	}
}

func (g gopher) add(ns <-chan uint, sums chan<- uint) {
	go func() {
		for n := range ns {
			g.sum += n
			// g.log.Level(logf.DEBUG).Print("{.} {id}", g.sum)
		}
		g.log.Level(logf.INFO+1).Print("{.} {id} done: {}", g.sum)
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

	log := logf.New().With(".", "Eulerian Gophers")

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

	log.Level(logf.INFO+2).Print("{.} done: {}", total)
}
