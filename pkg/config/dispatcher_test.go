package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatch(t *testing.T) {
	done := make(chan bool)

	wasDispatched := false
	fn := func(ctx context.Context, v interface{}) error {
		assert.Equal(t, "msg", v)
		wasDispatched = true

		done <- true
		return nil
	}

	d := NewDispatcher()

	cancel := d.Watch(fn)
	require.Len(t, d.keys, 1)

	ctx := context.Background()
	d.Dispatch(ctx, "msg")
	<-done
	require.True(t, wasDispatched)

	cancel()
	require.Len(t, d.keys, 0)
}
