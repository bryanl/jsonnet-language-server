package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHighlight(t *testing.T) {
	file := "file.jsonnet"

	source := "local x=1; x"
	pos := jpos.New(1, 7)

	nc := NewNodeCache()
	locations, err := Highlight(file, source, pos, nc)
	require.NoError(t, err)

	expected := []jpos.Location{
		jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
		jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
	}
	assert.Equal(t, expected, locations)
}
