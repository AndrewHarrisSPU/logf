package main

import (
	"os"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

func main() {
	sl := slog.New(slog.NewJSONHandler(os.Stdout))
	sl = sl.WithScope("outer").With("x", 1)

	log := logf.New(logf.Using.Handler(sl.Handler()))
	log = log.WithScope("inner").With("x", 2)

	// not captured because log doesn't know anything about sl's Handler state
	log.Msg("outer.x {outer.x}")
	log.Msg("outer.inner.x {outer.inner.x}")

	// captured, without outer - log's Handler only knows about "inner" scope
	log.Msg("inner.x {inner.x}")

	log = log.WithScope("local").With("x", 3)
	log.Msg("inner.x {inner.x}")
	log.Msg("local.x {local.x}")
}
