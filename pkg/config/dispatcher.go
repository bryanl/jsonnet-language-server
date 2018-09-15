package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/tracing"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// DispatchFn is a function that will be dispatched.
type DispatchFn func(context.Context, interface{}) error

// DispatchCancelFn is a function that cancels a dispatched function.
type DispatchCancelFn func()

// Dispatcher implements a dispatcher pattern.
type Dispatcher struct {
	keys map[string]DispatchFn

	mu sync.Mutex
}

// NewDispatcher creates an instance of Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		keys: make(map[string]DispatchFn),
	}
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// Dispatch dispatches a value to all the watchers.
func (d *Dispatcher) Dispatch(ctx context.Context, v interface{}) {
	span, ctx := tracing.ChildSpan(ctx, "dispatcher")

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, fn := range d.keys {
		go func(fn DispatchFn) {
			if err := fn(ctx, v); err != nil {
				span.LogFields(
					log.Error(err),
				)

				st, ok := err.(stackTracer)
				if ok {
					for _, f := range st.StackTrace() {
						fmt.Fprintf(os.Stderr, "%v\n", f)
					}
				}
			}
		}(fn)
	}
}

// Watch configures a watcher.
func (d *Dispatcher) Watch(fn DispatchFn) DispatchCancelFn {
	d.mu.Lock()
	defer d.mu.Unlock()

	u := uuid.NewV4()
	d.keys[u.String()] = fn

	cancel := func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		delete(d.keys, u.String())

		return
	}

	return cancel
}
