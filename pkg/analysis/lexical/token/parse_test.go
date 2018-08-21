package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	err := Parse("file.jsonnet", "string")
	require.NoError(t, err)

	assert.True(t, false)
}
