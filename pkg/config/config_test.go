package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_UpdateClientConfiguration(t *testing.T) {
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
			c := New()
			err := c.UpdateClientConfiguration(tc.update)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.key(c))
		})
	}
}

func TestConfig_UpdateClientConfiguration_watcher(t *testing.T) {
	update := map[string]interface{}{
		JsonnetLibPaths: []string{"new"},
	}

	c := New()

	done := make(chan bool)

	wasDispatched := false
	fn := func(v interface{}) error {
		wasDispatched = true
		assert.Equal(t, update[JsonnetLibPaths], v)

		done <- true
		return nil
	}

	cancel := c.Watch(JsonnetLibPaths, fn)
	c.UpdateClientConfiguration(update)

	<-done
	require.True(t, wasDispatched)
	cancel()
}

func TestConfig_StoreTextDocumentItem_watcher(t *testing.T) {
	c := New()

	tdi := TextDocument{
		uri:  "file:///new",
		text: "text",
	}

	done := make(chan bool)

	wasDispatched := false
	fn := func(got interface{}) error {
		wasDispatched = true
		assert.Equal(t, tdi, got)
		done <- true

		return nil
	}

	cancel := c.Watch(TextDocumentUpdates, fn)
	c.StoreTextDocumentItem(tdi)

	<-done
	require.True(t, wasDispatched)
	cancel()
}

func TestConfig_StoreTextDocumentItem(t *testing.T) {
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
			file := TextDocument{
				uri:  tc.uri,
				text: "text",
			}

			c := New()
			require.Len(t, c.textDocuments, 0)

			err := c.StoreTextDocumentItem(file)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, c.textDocuments, 1)
			text, err := c.Text(tc.uri)

			require.NoError(t, err)
			assert.Equal(t, "text", text.String())
		})
	}
}

func TestConfig_String(t *testing.T) {
	c := New()

	update := map[string]interface{}{
		JsonnetLibPaths: []string{"/path"},
	}

	err := c.UpdateClientConfiguration(update)
	require.NoError(t, err)

	got := c.String()

	expected := "{\"JsonnetLibPaths\":[\"/path\"]}"
	assert.Equal(t, expected, got)
}
