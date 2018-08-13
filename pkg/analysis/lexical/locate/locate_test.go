package locate

import (
	"fmt"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_findLocation2(t *testing.T) {
	cases := []struct {
		name   string
		source string
		pos    int
		loc    ast.Location
		isErr  bool
	}{
		{
			name:   "single row",
			source: "12345",
			pos:    0,
			loc:    createLoc(1, 1),
		},
		{
			name:   "multi line 1",
			source: "12\n34\n5",
			pos:    2,
			loc:    createLoc(2, 1),
		},
		{
			name:   "pos is past end of source",
			source: "1",
			pos:    2,
			isErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println("===")
			got, err := findLocation2(tc.source, tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.loc, got)
		})
	}
}
