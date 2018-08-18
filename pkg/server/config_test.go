package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Update(t *testing.T) {
	cases := []struct {
		name     string
		update   map[string]interface{}
		key      func(*Config) interface{}
		expected interface{}
		isErr    bool
	}{
		{
			name: "update jsonnet lib paths",
			update: map[string]interface{}{
				"jsonnet.libPaths": []string{"new"},
			},
			key: func(c *Config) interface{} {
				return c.JsonnetLibPaths
			},
			expected: []string{"new"},
		},
		{
			name: "update jsonnet lib paths with []interface",
			update: map[string]interface{}{
				"jsonnet.libPaths": []interface{}{"new"},
			},
			key: func(c *Config) interface{} {
				return c.JsonnetLibPaths
			},
			expected: []string{"new"},
		},
		{
			name: "invalid setting type",
			update: map[string]interface{}{
				"jsonnet.libPaths": "new",
			},
			isErr: true,
		},
		{
			name: "unknown setting",
			update: map[string]interface{}{
				"unknown": []string{"new"},
			},
			isErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewConfig()
			err := c.Update(tc.update)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.key(c))
		})
	}
}

func TestConfig_Update_watcher(t *testing.T) {
	update := map[string]interface{}{
		CfgJsonnetLibPaths: []string{"new"},
	}

	c := NewConfig()

	done := make(chan bool)

	wasDispatched := false
	fn := func(v interface{}) {
		wasDispatched = true
		assert.Equal(t, update[CfgJsonnetLibPaths], v)

		done <- true
	}

	cancel := c.Watch(CfgJsonnetLibPaths, fn)
	c.Update(update)

	<-done
	require.True(t, wasDispatched)
	cancel()
}
