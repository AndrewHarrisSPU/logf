package main

import (
	"github.com/AndrewHarrisSPU/logf"
)

var print = logf.New().Msg

func Append[T any](xs []T, yss ...[]T) []T {
	for _, ys := range yss {
		xs = append(xs, ys...)
	}
	return xs
}

func Collect[T any](yss ...[]T) []T {
	var n int
	for _, ys := range yss {
		n += len(ys)
	}

	xs := make([]T, n)
	var i int
	for _, ys := range yss {
		for _, y := range ys {
			xs[i] = y
			i++
		}
	}
	return xs
}

func main() {
	// yss := [][]rune {
	// 	[]rune{ 'H', 'e', 'l', 'l', 'o' },
	// 	[]rune{ ',', ' ' },
	// 	[]rune{ 'W', 'o', 'r', 'l', 'd' },
	// 	[]rune{ '!' },
	// }

	// print( string( Append( []rune{}, yss...)))

	// print( "{}", len( append( []*int{}, nil )))

	// print( "{}", len( append( []*int{}, []*int{}... )))

	var xs, ys []*int
	print("{}", len(append([]*int{}, append(xs, nil)...)))
	print("{}", len(append([]*int{}, append(xs, ys...)...)))

	// print( "{}", len( append( [][]*int{}, xs, ys )))

	// print( "{}", len( Collect[int](nil, nil, nil)))
}
