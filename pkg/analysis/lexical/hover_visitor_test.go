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
	cases := []struct {
		name          string
		filename      string
		loc           ast.Location
		expectedToken interface{}
		expectedLoc   ast.LocationRange
		isErr         bool
	}{
		{
			name:          "hover object",
			filename:      "example1.jsonnet",
			loc:           createLoc(2, 9),
			expectedToken: ast.Identifier("name"),
			expectedLoc:   createFileRange("example1.jsonnet", 2, 7, 2, 10),
		},
		{
			name:          "hover var",
			filename:      "example2.jsonnet",
			loc:           createLoc(11, 13),
			expectedToken: &ast.Var{},
			expectedLoc:   createFileRange("example2.jsonnet", 11, 13, 11, 14),
		},
		{
			name:          "hover index 1",
			filename:      "example2.jsonnet",
			loc:           createLoc(11, 17),
			expectedToken: &ast.Index{},
			expectedLoc:   createFileRange("example2.jsonnet", 11, 15, 11, 21),
		},
		{
			name:          "hover index 2",
			filename:      "example2.jsonnet",
			loc:           createLoc(11, 26),
			expectedToken: &ast.Index{},
			expectedLoc:   createFileRange("example2.jsonnet", 11, 23, 11, 29),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := jlstesting.Testdata(t, "lexical", tc.filename)
			r := strings.NewReader(source)

			hv, err := newHoverVisitor(tc.filename, r, tc.loc)
			require.NoError(t, err)

			err = hv.Visit()
			require.NoError(t, err)

			got, err := hv.TokenAtLocation()
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.IsType(t, tc.expectedToken, got.Token)
			assertLocationRange(t, tc.expectedLoc, got.Loc)
		})
	}

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

func createFileRange(name string, r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		FileName: name,
		Begin:    createLoc(r1l, r1c),
		End:      createLoc(r2l, r2c),
	}
}

func assertLocationRange(t *testing.T, expected, actual ast.LocationRange) {
	assert.Equal(t, expected.FileName, actual.FileName)
	assert.Equal(t, expected.Begin, actual.Begin,
		"range begin expected = %s; actual = %s",
		expected.Begin.String(), actual.Begin.String())
	assert.Equal(t, expected.End, actual.End,
		"range end expected = %s; actual = %s",
		expected.End.String(), actual.End.String())
}
