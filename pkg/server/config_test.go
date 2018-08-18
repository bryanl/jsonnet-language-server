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
