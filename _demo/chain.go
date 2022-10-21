package main

import (
	"context"
	// "errors"

	"github.com/AndrewHarrisSPU/logf"
)

type cont[T any] struct {
	t   T
	ctx context.Context
	err error
}

func Apply[T any](fns ...contFunc[T]) (T, error) {
	s := &cont[T]{}

	for i, fn := range fns {
		fn(s)
		logf.Print("{:%2d}: {}", i, s.t)
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

	logf.Print("{}", t) // (1)(1) )
}

/*
type ChainFunc[T any] func(state[T]) func(any)

type state[T any] struct {
	t T
	ctx context.Context
	err error
}



type anyfunc[T any] any

type Funcs[T any] []anyfunc[T]

func NewFunc[T any, S Signature[T]](fn S) anyfunc[T] {
	return anyfunc[T](fn)
}

type Signature[T any] interface {
	func(context.Context, T) T |
	func(context.Context, T) (T, error) |
	func(context.Context, T, error) T |
	func(context.Context, T, error)(T, error)|
	func(T) T |
	func(T) (T,error) |
	func(T,error) (T,error) |
	func(T,error) T
}

func chainRewriteSignature[T any](fn anyfunc[T]) func(state[T]) func(any) {
	z = a
	for _, fn := range fns {
		if err != nil {
			return
		}
		switch fn := any(fn).(type) {
		case func(context.Context, T) T:
			z = fn(ctx, z)
		case func(context.Context, T) (T, error):
			z, err = fn(ctx, z)
		case func(context.Context, T, error) T:
			z = fn(ctx, z, err)
		case func(context.Context, T, error)(T, error):
			z, err = fn(ctx, z, err)
		case func(T) T:
			z = fn(z)
		case func(T) (T, error):
			z, err = fn(z)
		case func(T, error) T:
			z = fn(z, err)
		case func(T,error) (T,error):
			z, err = fn(z, err)
		}
	}
	return
}

func main() {
	add1 := NewFunc[int]( func(n int) int {
		return n + 1
	})

	div0 := NewFunc[int]( func(n int)(int, error) {
		return n, errors.New("!divide-by-zero")
	})

	fns := Funcs[int]{ add1, add1, add1, div0 }

	t, err := Chain(context.Background(), 5, fns)
	logf.Print( "{}, {}", t, err )
}
*/
