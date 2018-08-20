package server

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_updateClientConfiguration(t *testing.T) {
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
				return c.JsonnetLibPaths()
			},
			expected: []string{"new"},
		},
		{
			name: "update jsonnet lib paths with []interface",
			update: map[string]interface{}{
				"jsonnet.libPaths": []interface{}{"new"},
			},
			key: func(c *Config) interface{} {
				return c.JsonnetLibPaths()
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
			err := c.updateClientConfiguration(tc.update)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.key(c))
		})
	}
}

func TestConfig_updateClientConfiguration_watcher(t *testing.T) {
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
	c.updateClientConfiguration(update)

	<-done
	require.True(t, wasDispatched)
	cancel()
}

func TestConfig_storeTextDocumentItem(t *testing.T) {
	cases := []struct {
		name  string
		uri   string
		isErr bool
	}{
		{
			name: "valid uri",
			uri:  "file:///file.jsonnet",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := lsp.TextDocumentItem{
				URI:  tc.uri,
				Text: "text",
			}

			c := NewConfig()
			require.Len(t, c.textDocuments, 0)

			err := c.storeTextDocumentItem(file)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, c.textDocuments, 1)
			text, err := c.Text(tc.uri)

			require.NoError(t, err)
			assert.Equal(t, "text", text)
		})
	}
}

func TestConfig_String(t *testing.T) {
	c := NewConfig()

	update := map[string]interface{}{
		CfgJsonnetLibPaths: []string{"/path"},
	}

	err := c.updateClientConfiguration(update)
	require.NoError(t, err)

	got := c.String()

	expected := "{\"JsonnetLibPaths\":[\"/path\"]}"
	assert.Equal(t, expected, got)
}
