package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/require"
)

func Test_completation_complete(t *testing.T) {
	l := &locate.Locatable{
		Scope: locate.Scope{
			"a": locate.Locatable{
				Token: &ast.LiteralString{Value: "a", Kind: ast.StringDouble},
			},
		},
	}

	loc := createLoc(2, 1)

	c, err := newCompletion(l)
	require.NoError(t, err)
	c.complete(loc)
}
