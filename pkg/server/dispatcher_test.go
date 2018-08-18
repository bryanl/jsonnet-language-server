package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatch(t *testing.T) {
	done := make(chan bool)

	wasDispatched := false
	fn := func(v interface{}) {
		assert.Equal(t, "msg", v)
		wasDispatched = true

		done <- true
	}

	d := NewDispatcher()

	cancel := d.Watch(fn)
	require.Len(t, d.keys, 1)

	d.Dispatch("msg")
	<-done
	require.True(t, wasDispatched)

	cancel()
	require.Len(t, d.keys, 0)
}
