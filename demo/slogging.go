package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	tty := logf.New().
		AddSource(true).
		Elapsed(true).
		Ref(logf.DEBUG).
		Stream(logf.INFO, logf.WARN).
		StreamSizes(1, 1).
		StreamRefresh(20).
		TTY()

	slog.SetDefault(slog.New(tty.WithAttrs(logf.Attrs("place", "world"))))
	ctx := slog.NewContext(context.Background(), slog.Default())

	// random log traffic
	go ping(logf.DEBUG, 100)
	go ping(logf.INFO, 1_000)
	go ping(logf.WARN, 4_000)
	go halfway(tty, 5_000)
	go deadline(ctx, 10_000)

	<-time.NewTimer(10 * time.Second).C
	tty.Close()
}

func deadline(ctx context.Context, interval int) {
	log := slog.FromContext(ctx)
	d := time.Duration(rand.Intn(interval))
	ctx, _ = context.WithTimeout(ctx, d*time.Millisecond)
	<-ctx.Done()
	log.Error("", ctx.Err())
}

func halfway(tty *logf.TTY, interval int) {
	<-time.NewTimer(time.Duration(interval) * time.Millisecond).C
	tty.WriteString("halfway!")
}

func ping(level slog.Level, interval int) {
	i := 0
	for {
		d := time.Duration(rand.Intn(interval))
		<-time.NewTimer(d * time.Millisecond).C
		slog.Log(level, "Hello, {place}", "i", i)
		i++
	}
}
