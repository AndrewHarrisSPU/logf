package main

import (
	"math"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

var print = logf.New().With("demo", "overload").Print

type Agent struct {
	First string
	Last  string
}

func (a Agent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("first", a.First),
		slog.String("last", a.Last),
	)
}

func main() {
	print("Ya, it's possible to overload {}", "print")
	print("pi is {:%.2f}", math.Pi)

	mulder := Agent{"Fox", "Mulder"}
	print("{}", mulder)

	print("{:%+v}", struct {
		first string
		last  string
	}{first: "Fox", last: "Mulder"})
}
