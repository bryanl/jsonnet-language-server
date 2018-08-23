package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDocument_Range(t *testing.T) {
	source := "123456789\n123456789"
	td := &TextDocument{
		text: source,
	}

	got, err := td.Truncate(2, 3)
	require.NoError(t, err)

	expected := "123456789\n123"
	assert.Equal(t, expected, got)
}
