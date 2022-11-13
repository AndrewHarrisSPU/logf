package main

import (
	"sync"

	"github.com/AndrewHarrisSPU/logf"
)

type list struct {
	wg sync.WaitGroup
	mu *sync.Mutex
	ns []int
}

func (l *list) Map(filter func(int) bool) func(func(*int) bool) bool {
	l.wg.Add(1)

	return func(yield func(x *int) bool) bool {
		l.mu.Lock()
		for i := range l.ns {
			n := &l.ns[i]
			if !filter(*n) {
				continue
			}
			if !yield(n) {
				break
			}
		}
		l.mu.Unlock()
		l.wg.Done()
		return false
	}
}

func main() {
	l := list{
		mu: new(sync.Mutex),
		ns: []int{1, 2, 3, 4, 5},
	}

	mapOdd := l.Map(func(x int) bool {
		return x%2 != 0
	})

	mapEven := l.Map(func(x int) bool {
		return x%2 == 0
	})

	go func() {
		for mapOdd(func(n *int) bool {
			*n = *n + 1
			return true
		}) {
		}
	}()

	go func() {
		for mapEven(func(n *int) bool {
			*n = *n * *n
			return true
		}) {
		}
	}()

	l.wg.Wait()

	log := logf.New().
		Layout("message").
		Logger()

	log.Msg("{}", l.ns)
}
