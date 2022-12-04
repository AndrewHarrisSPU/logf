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
		log: log.
			Level(logf.DEBUG).
			Tag("gopher").
			With("id", i),
		id:  i,
		sum: 0,
	}
}

func (g gopher) add(ns <-chan int, sums chan<- int) {
	go func() {
		for n := range ns {
			g.sum += n
			g.log.Msg("{id}: {sum}", "sum", g.sum)
		}
		g.log.Msg("{id} done")
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
		Ref(logf.DEBUG).
		Layout( "level", "tags", "message", "attrs").
		Level(logf.LevelBar).
		TTY()

	log := tty.
		Logger().
		Tag("main")

	ns, sums := make(chan int), make(chan int)

	for i := 0; i < gophersN; i++ {
		newGopher(log, i).add(ns, sums)
	}

	for i := 1; i < rangeN; i++ {
		<-time.NewTimer(time.Millisecond * 10).C
		if i%3 == 0 {
			log.Level(logf.INFO).Msg("FIZZ", "fizz", i)
		}
		if i%5 == 0 {
			log.Level(logf.INFO+1).Msg("BUZZ", "buzz", i)
		}
		ns <- i
	}
	close(ns)

	var total int
	for i := 0; i < gophersN; i++ {
		total += <-sums
	}

	tty.WriteString( log.Msgf("{}", total))
}
