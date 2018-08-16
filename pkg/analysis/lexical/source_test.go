package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLines(t *testing.T) {
	data := "1\n2\n3\n4\n5\n"

	got, err := ExtractLines([]byte(data), 2, 4)
	require.NoError(t, err)

	expected := "2\n3\n4\n"

	assert.Equal(t, expected, string(got))
}

func TestExtractUntil(t *testing.T) {
	data := jlstesting.Testdata(t, "local.jsonnet")

	cases := []struct {
		name     string
		loc      ast.Location
		expected string
		isErr    bool
	}{
		{
			name:     "single line",
			loc:      createLoc(1, 7),
			expected: "local ",
		},
		{
			name:     "multiple lines",
			loc:      createLoc(2, 7),
			expected: "local foo1 = \"bar\", foo2=\"bar\";\nlocal ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractUntil(data, tc.loc)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, string(got))

		})
	}
}

func TestExtractCount(t *testing.T) {
	cases := []struct {
		name     string
		data     string
		count    int
		expected string
		isErr    bool
	}{
		{
			name:     "extract 3",
			data:     "1234567890",
			count:    3,
			expected: "123",
		},
		{
			name:     "extract 0",
			data:     "1234567890",
			count:    0,
			expected: "",
		},
		{
			name:     "extract 3 last",
			data:     "1234567890",
			count:    -3,
			expected: "890",
		},
		{
			name:  "extract more characters than length",
			data:  "123",
			count: 15,
			isErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(tc.data)
			got, err := ExtractCount(data, tc.count)
			if tc.isErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			expected := []byte(tc.expected)

			assert.Equal(t, expected, got)
		})
	}
}

func TestFindLocation(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		count    int
		expected ast.Location
		isErr    bool
	}{
		{
			name:     "single line",
			in:       "123456",
			count:    3,
			expected: createLoc(1, 3),
		},
		{
			name:     "single line",
			in:       "123\n456",
			count:    4,
			expected: createLoc(2, 1),
		},
		{
			name:  "past end of file",
			in:    "123\n456",
			count: 10,
			isErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FindLocation([]byte(tc.in), tc.count)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)

		})
	}

}
