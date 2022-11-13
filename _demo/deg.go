package main

import (
	"math"

	"github.com/AndrewHarrisSPU/logf"
)

var print = logf.New().Printer()

type coord struct {
	x, y, z float64
}

func main() {
	twig1 := print.Label("deg2rad").Msg
	deg2rad := func(deg int) {
		rad := float64(deg) * (math.Pi / 180.0)
		twig1("{:%3d} degrees -> {:%3.2f} radians", deg, rad)
	}

	twig2 := print.Label("rad2deg").Msg
	rad2deg := func(rad float64) {
		deg := (180 * rad) / math.Pi
		twig2("{:%3.2f} radians -> {:%3.0f} degrees", rad, deg)
	}

	twig3 := print.Label("stepper").Msg
	stepper := func(step int) {
		rad := (float64(step) / 16.0) * math.Pi
		deg := (180 * rad) / math.Pi
		twig3("step {:%2d}: {:%3.2f} radians, {:%3.0f} degrees", step, rad, deg)
	}

	twig4 := print.Label("cube").Msg
	coords := func(x, y, z float64) {
		twig4("[{} {} {}]", x, y, z)
	}

	twig5 := print.Label("lowercase").Msg
	lowercase := func(r rune) {
		twig5("{:%2d}: {:%c}", r-'a'+1, r)
	}

	for deg := 0; deg <= 90; deg += 10 {
		deg2rad(deg)
	}

	for rad := 0.0; rad <= (math.Pi / 2); rad += (math.Pi / 16) {
		rad2deg(rad)
	}

	for step := 0; step <= 8; step++ {
		stepper(step)
	}

	h := coord{1, 1, 1}
	t := coord{4, 4, 4}
	for z := h.z; z < t.z; z++ {
		for y := h.y; y < t.y; y++ {
			for x := h.x; x < t.x; x++ {
				coords(x, y, z)
			}
		}
	}

	for r := 'a'; r <= 'z'; r++ {
		lowercase(r)
	}
}
