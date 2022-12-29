package main

import (
	"math/rand"
	"time"

	"github.com/AndrewHarrisSPU/logf"
)

type gopher struct {
	log logf.Logger
	sum int
}

func newGopher(log logf.Logger, i int) gopher {
	return gopher{
		log: log.
			With("#", "euler-gopher", "id", i),
		sum: 0,
	}
}

func (g gopher) add(ns <-chan int, sums chan<- int) {
	go func() {
		for n := range ns {
			g.sum += n
			g.log.Debugf("{id}: {sum}", "sum", g.sum)
		}
		g.log.Infof("{id} done")
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
		Layout("level", "tag", "message").
		Tag("#", "dim").
		Level(logf.LevelBullet).
		TTY()

	tty.SetRef(logf.DEBUG)

	tty.Filter("euler-gopher", "fizzbuzz")

	log := tty.
		Logger().
		With("#", "fizzbuzz")

	ns, sums := make(chan int), make(chan int)

	for i := 0; i < gophersN; i++ {
		newGopher(log, i).add(ns, sums)
	}

	for i := 1; i < rangeN; i++ {
		<-time.NewTimer(time.Millisecond * 10).C
		if i%3 == 0 {
			log.Info("FIZZ", "fizz", i)
		}
		if i%5 == 0 {
			log.Log(logf.INFO+1, "BUZZ", "buzz", i)
		}
		ns <- i
	}
	close(ns)

	var total int
	for i := 0; i < gophersN; i++ {
		total += <-sums
	}

	tty.WriteString(log.Fmt("(counted to {})", total))
}
