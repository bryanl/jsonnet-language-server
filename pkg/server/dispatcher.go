package server

import (
	"sync"

	uuid "github.com/satori/go.uuid"
)

// Dispatcher implements a dispatcher pattern.
type Dispatcher struct {
	keys map[string]func(interface{})

	mu sync.Mutex
}

// NewDispatcher creates an instance of Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		keys: make(map[string]func(interface{})),
	}
}

// Dispatch dispatches a value to all the watchers.
func (d *Dispatcher) Dispatch(v interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, fn := range d.keys {
		go fn(v)
	}
}

// Watch configures a watcher.
func (d *Dispatcher) Watch(fn func(interface{})) func() {
	d.mu.Lock()
	defer d.mu.Unlock()

	u := uuid.Must(uuid.NewV4())
	d.keys[u.String()] = fn

	cancel := func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		delete(d.keys, u.String())

		return
	}

	return cancel
}
