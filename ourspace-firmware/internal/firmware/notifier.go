package firmware

import (
	"sync"
)

type Notifier struct {
	cond  *sync.Cond
	value []byte
}

func NewNotifier() *Notifier {
	return &Notifier{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (n *Notifier) Notify(value []byte) {
	n.cond.L.Lock()
	n.value = value
	n.cond.Broadcast()
	n.cond.L.Unlock()
}

func (n *Notifier) Wait() []byte {
	n.cond.L.Lock()
	n.cond.Wait()
	v := make([]byte, len(n.value))
	copy(v, n.value)
	n.cond.L.Unlock()

	return v
}
