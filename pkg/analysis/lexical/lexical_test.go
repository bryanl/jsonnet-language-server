package lexical

import (
	"strings"
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/sourcegraph/go-langserver/pkg/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoverAtLocation(t *testing.T) {
	data := jlstesting.Testdata(t, "lexical", "example2.jsonnet")

	r := strings.NewReader(data)
	got, err := HoverAtLocation("example2.jsonnet", r, 1, 13)
	require.NoError(t, err)

	expected := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Value: "(literal) name: string",
			},
		},
		Range: newRange(2, 7, 2, 10),
	}

	assert.Equal(t, expected, got)
}

func newPosition(l, c int) lsp.Position {
	return lsp.Position{Line: l - 1, Character: c - 1}
}

func newRange(sl, sc, el, ec int) lsp.Range {
	return lsp.Range{
		Start: newPosition(sl, sc),
		End:   newPosition(el, ec),
	}
}
