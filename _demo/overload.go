package main

import (
	"math"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

var print = logf.New().Label("overload")

type Agent struct {
	First string
	Last  string
}

func (a Agent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("", "Agent"),
		slog.String("first", a.First),
		slog.String("last", a.Last),
	)
}

func main() {
	print.Msg("Ya, it's possible to overload {}", "print")
	print.Msg("pi is {:%.2f}", math.Pi)

	mulder := Agent{"Fox", "Mulder"}
	print.Msg("{}", mulder)

	print.Msg("{:%+v}", struct {
		first string
		last  string
	}{first: "Fox", last: "Mulder"})
}
