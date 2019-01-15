package mp

import (
	"sync"

	"go.uber.org/atomic"
)

type Router struct {
	txrx         chan func()
	chanReturned *atomic.Bool
}

func NewRouter() *Router {
	return &Router{
		txrx:         make(chan func(), 128),
		chanReturned: atomic.NewBool(false),
	}
}

func (m *Router) Execute(fn func()) {
	m.txrx <- fn
}

func (m *Router) Rx() <-chan func() {
	if m.chanReturned.Toggle() {
		panic("this function can be called only once to avoid race conditions")
	}

	return m.txrx
}

func (m *Router) NewSignal() *Signal {
	return &Signal{
		tx: m.txrx,
	}
}

type Signal struct {
	tx chan<- func()

	mu    sync.RWMutex
	slots []Slot
}

func (m *Signal) Connect(slot Slot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.slots = append(m.slots, slot)
}

func (m *Signal) Emit(v interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, slot := range m.slots {
		slot := slot
		m.tx <- func() { slot(v) }
	}
}

type Slot = func(v interface{})
