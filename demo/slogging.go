package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	tty := logf.NewDefault().
		AddSource(true).
		Layout("level", "time", "tags", "message", "source").
		Time("dim", logf.TimeShort).
		Source("dim", logf.SourceShort).
		Tag("place", "dim magenta").
		Tag("i", "bright magenta").
		Tag("done", "bright magenta").
		TTY()

	tty.SetRef(logf.DEBUG)

	tty.FilterRecords(func(r slog.Record) bool {
		var ok bool
		r.Attrs(func(a slog.Attr) {
			ok = ok || a.Key == "done"
		})
		return ok
	})

	ctx := slog.NewContext(context.Background(), slog.Default().With("place", "slogoverse"))
	d := 5_000 * time.Millisecond
	ctx, _ = context.WithTimeout(ctx, d)

	// random log traffic
	wg := new(sync.WaitGroup)
	wg.Add(3)

	go ping(ctx, wg, logf.DEBUG, 100)
	go ping(ctx, wg, logf.INFO, 1_000)
	go ping(ctx, wg, logf.WARN, 500)
	deadline(ctx)

	wg.Wait()
}

func deadline(ctx context.Context) {
	log := slog.FromContext(ctx)
	<-ctx.Done()
	log.Error("", ctx.Err())
}

func ping(ctx context.Context, wg *sync.WaitGroup, level slog.Level, interval int) {
	log := slog.FromContext(ctx)

	d := time.Duration(interval)
	tick := time.NewTicker(d * time.Millisecond).C
	i := 0
	for {
		select {
		case <-tick:
			log.Log(level, "tick", "i", i)
			i++
		case <-ctx.Done():
			log.Log(level, "", "done", "bye!")
			wg.Done()
			return
		}
	}
}
