package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureHelper(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		pos      jpos.Position
		expected *SignatureResponse
	}{
		{
			name:   "single required",
			source: "local id(x) = x; id()",
			pos:    jpos.New(1, 21),
			expected: &SignatureResponse{
				Label:      "id(x)",
				Parameters: []string{"x"},
			},
		},
		{
			name:   "multiple required",
			source: "local id(x,y) = x; id()",
			pos:    jpos.New(1, 23),
			expected: &SignatureResponse{
				Label:      "id(x, y)",
				Parameters: []string{"x", "y"},
			},
		},
		{
			name:   "optional",
			source: "local id(x=1) = x; id()",
			pos:    jpos.New(1, 23),
			expected: &SignatureResponse{
				Label:      "id(x=1)",
				Parameters: []string{"x"},
			},
		},
		{
			name:   "multiple optional",
			source: "local id(x=1,y=1) = x+y; id()",
			pos:    jpos.New(1, 30),
			expected: &SignatureResponse{
				Label:      "id(x=1, y=1)",
				Parameters: []string{"x", "y"},
			},
		},
		{
			name:   "optional/required",
			source: "local id(x,y=1) = x+y; id()",
			pos:    jpos.New(1, 27),
			expected: &SignatureResponse{
				Label:      "id(x, y=1)",
				Parameters: []string{"x", "y"},
			},
		},
		{
			name:   "index",
			source: "local o={id(x)::x}; o.id()",
			pos:    jpos.New(1, 26),
			expected: &SignatureResponse{
				Label:      "id(x)",
				Parameters: []string{"x"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nodeCache := NewNodeCache()

			sr, err := SignatureHelper(tc.source, tc.pos, nodeCache)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, sr)
		})
	}
}
