package main

import (
	"context"

	"github.com/AndrewHarrisSPU/logf"
)

var print = logf.New().Label("chain")

type cont[T any] struct {
	t   T
	ctx context.Context
	err error
}

func Apply[T any](fns ...contFunc[T]) (T, error) {
	s := &cont[T]{}

	for i, fn := range fns {
		fn(s)
		print.Msg("step {:%d}: {}", i, s.t)
	}

	return s.t, s.err
}

type contFunc[T any] func(*cont[T])

func mungeFunc[T any, SIG interface {
	func(context.Context, T) T |
		func(context.Context, T) (T, error) |
		func(context.Context, T, error) T |
		func(context.Context, T, error) (T, error) |
		func(T) T |
		func(T) (T, error) |
		func(T, error) (T, error) |
		func(T, error) T
}](fn SIG) contFunc[T] {
	switch fn := any(fn).(type) {
	case func(context.Context, T) T:
		return contFunc[T](func(s *cont[T]) {
			s.t = fn(s.ctx, s.t)
		})
	case func(context.Context, T) (T, error):
		return contFunc[T](func(s *cont[T]) {
			s.t, s.err = fn(s.ctx, s.t)
		})
	case func(context.Context, T, error) T:
		return contFunc[T](func(s *cont[T]) {
			s.t = fn(s.ctx, s.t, s.err)
		})
	case func(context.Context, T, error) (T, error):
		return contFunc[T](func(s *cont[T]) {
			s.t, s.err = fn(s.ctx, s.t, s.err)
		})
	case func(T) T:
		return contFunc[T](func(s *cont[T]) {
			s.t = fn(s.t)
		})
	case func(T) (T, error):
		return contFunc[T](func(s *cont[T]) {
			s.t, s.err = fn(s.t)
		})
	case func(T, error) T:
		return contFunc[T](func(s *cont[T]) {
			s.t = fn(s.t, s.err)
		})
	case func(T, error) (T, error):
		return contFunc[T](func(s *cont[T]) {
			s.t, s.err = fn(s.t, s.err)
		})
	default:
		panic("what type?")
	}
}

func main() {
	add1 := mungeFunc[int](func(n int) int {
		n++
		return n
	})

	mulN := func(m int) contFunc[int] {
		return mungeFunc[int](func(n int) int {
			return m * n
		})
	}

	t, _ := Apply[int](add1, add1, mulN(20), add1)

	print.Msg("{}", t)
}
