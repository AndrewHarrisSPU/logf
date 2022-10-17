package main

import (
	"math"

	"github.com/AndrewHarrisSPU/logf"
	"golang.org/x/exp/slog"
)

var print = logf.Print

type Agent struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

func (a Agent) LogValue() slog.Value {
	return slog.GroupValue([]slog.Attr{slog.String("first", a.First), slog.String("last", a.Last)}...)
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

	print("{:%+v}", []string{"Fox", "Mulder"})
}
