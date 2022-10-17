package main

import (
	"errors"

	"github.com/AndrewHarrisSPU/logf"
)

var print = logf.New(logf.Using.Minimal(true, true))

// iteration with session states
type State interface {
	Next(State) State
	State() error
}

// next
type ok struct{}

// all instances of ok{} are identical
var Ok = ok{}

func (ok) Next(curr State) State {
	if curr.State() != nil {
		return curr.(stop)
	}
	return ok{}
}

func (ok) State() error {
	return nil
}

// stop
type stop struct {
	err error
}

var iterStop = errors.New("")

// Stop is constructed from an error
func Stop(err error) stop {
	if err == nil {
		return stop{iterStop}
	}
	return stop{err}
}

func (stop) Next(curr State) State {
	return stop{curr.State()}
}

func (s stop) State() error {
	return s.err
}

type IterErr[T any] interface {
	Next(State) (T, State)
}

func takeN[T any](it IterErr[T], n int) []T {
	ts := make([]T, 0, n)
	i := 0
	for t, state := it.Next(Ok); state.State() == nil; t, state = it.Next(state) {
		ts = append(ts, t)
		i++
		if i >= n {
			state = Stop(errOverLimit)
			print.Err("takeN", state.State())
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
	switch curr.State() {
	case nil:
		c.n++
		if c.n > c.lim {
			next = Stop(errOverLimit)
			print.Err("counter", next.State())
			return
		}
	default:
		next = Stop(curr.State())
		return
	}
	return c.n, Ok
}

// main

func main() {
	c := &counter{lim: 42}
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
	print.Msg("{}", takeN[int](c, 10))
}
