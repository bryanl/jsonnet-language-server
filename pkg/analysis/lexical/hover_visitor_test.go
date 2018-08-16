package lexical

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_hoverVisitor(t *testing.T) {
	source := jlstesting.Testdata(t, "lexical", "example1.jsonnet")
	r := strings.NewReader(source)
	loc := createLoc(2, 9)

	hv, err := newHoverVisitor("example1.jsonnet", r, loc)
	require.NoError(t, err)

	err = hv.Visit()
	require.NoError(t, err)

	got, err := hv.TokenAtLocation()
	require.NoError(t, err)

	lLoc := createRange(2, 7, 2, 10)
	lLoc.FileName = filepath.Join("example1.jsonnet")

	expected := &locate.Locatable{
		Token: ast.Identifier("name"),
		Loc:   lLoc,
	}

	assert.Equal(t, expected.Token, got.Token)
	assert.Equal(t, expected.Loc, got.Loc)
}
