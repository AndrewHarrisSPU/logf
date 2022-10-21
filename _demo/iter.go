package main

import (
	"errors"

	"github.com/AndrewHarrisSPU/logf"
)

var print = logf.New().Print

// iteration with session states
type State struct {
	Err error
}

// all instances of ok{} are identical
var Ok = State{nil}

type IterErr[T any] interface {
	Next(State) (T, State)
}

func takeN[T any](it IterErr[T], n int) []T {
	ts := make([]T, 0, n)
	i := 0
	for t, state := it.Next(Ok); state.Err == nil; t, state = it.Next(state) {
		ts = append(ts, t)
		i++
		if i >= n {
			state = State{errOverLimit}
			print("takeN", state.Err)
			continue
		}
	}
	return ts
}

// counter

type counter struct {
	n   int
	lim int
}

var errOverLimit = errors.New("over limit")

func (c *counter) Next(curr State) (i int, next State) {
	switch curr {
	case Ok:
		c.n++
		if c.n > c.lim {
			next = State{errOverLimit}
			print("counter", next.Err)
			return
		}
		return c.n, Ok
	default:
		next = curr
		return
	}
}

// main

func main() {
	c := &counter{lim: 42}
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
	print("{}", takeN[int](c, 10))
}
