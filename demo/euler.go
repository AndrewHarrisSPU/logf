package main

import (
	"math/rand"
	"time"

	"github.com/AndrewHarrisSPU/logf"
)

type gopher struct {
	log *logf.Logger
	id  int
	sum int
}

func newGopher(log *logf.Logger, i int) gopher {
	return gopher{
		log: log.Level(logf.DEBUG).Label("gopher").With("id", i),
		id:  i,
		sum: 0,
	}
}

func (g gopher) add(ns <-chan int, sums chan<- int) {
	go func() {
		for n := range ns {
			g.sum += n
			g.log.Msg("got a number", "sum", g.sum)
		}
		g.log.Msg("done")
		sums <- g.sum
	}()
}

var (
	gophersN = 10
	rangeN   = 101
)

func main() {
	rand.Seed(time.Now().UnixNano())

	tty := logf.New().
		Elapsed(true).
		Ref(logf.DEBUG).
		Stream(logf.INFO, logf.INFO+1).
		TTY()

	log := tty.Logger().Label("main")

	ns, sums := make(chan int), make(chan int)

	for i := 0; i < gophersN; i++ {
		newGopher(log, i).add(ns, sums)
	}

	for i := 1; i < rangeN; i++ {
		<-time.NewTimer(time.Millisecond * 30).C
		if i%9 == 0 {
			log.Level(logf.INFO).Msg("mod fizz squared: {}", i)
		}
		if i%25 == 0 {
			msg := log.Msgf("mod buzz squared: {}", i)
			tty.WriteString(msg)
		}
		ns <- i
	}
	close(ns)

	var total int
	for i := 0; i < gophersN; i++ {
		total += <-sums
	}

	log.Level(logf.INFO+1).Msg("{}", total)
	tty.Close()
}
