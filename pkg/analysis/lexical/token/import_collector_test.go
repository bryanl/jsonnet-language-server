package token

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportCollector_Collect(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		expected []string
		isErr    bool
	}{
		{
			name:     "no tokens",
			filename: "importcollector1.jsonnet",
			expected: []string{},
		},
		{
			name:     "imports a file",
			filename: "importcollector2.jsonnet",
			expected: []string{"importcollector1.jsonnet"},
		},
		{
			name:     "import has imports",
			filename: "importcollector3.jsonnet",
			expected: []string{"importcollector1.jsonnet",
				"importcollector2.jsonnet"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			abs, err := filepath.Abs(filepath.Join("testdata"))
			require.NoError(t, err)

			sourceFile := filepath.Join(abs, tc.filename)

			libPaths := []string{abs}
			ic := NewImportCollector(libPaths)

			files, err := ic.Collect(sourceFile)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, files)
		})
	}

}
