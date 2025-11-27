package firmware

import (
	"sync"
)

type Notifier[T any] struct {
	cond  *sync.Cond
	value *T
}

func NewNotifier[T any]() *Notifier[T] {
	return &Notifier[T]{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (n *Notifier[T]) Notify(value *T) {
	v := *value

	n.cond.L.Lock()
	n.value = &v
	n.cond.Broadcast()
	n.cond.L.Unlock()
}

func (n *Notifier[T]) Wait() *T {
	n.cond.L.Lock()
	n.cond.Wait()
	v := *n.value
	n.cond.L.Unlock()

	return &v
}
