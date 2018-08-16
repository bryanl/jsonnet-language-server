package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	idxID := ast.Identifier("nested2")
	idx := &ast.Index{
		Id:       &idxID,
		NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 11, 13, 11, 22)),
	}

	l := &Locatable{}

	source := jlstesting.Testdata(t, "index1.jsonnet")

	got, err := Index(idx, l, source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 11, 15, 11, 21)
	assert.Equal(t, expected, got)
}
