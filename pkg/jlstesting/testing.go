package jlstesting

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Testdata(t *testing.T, elem ...string) string {
	name := filepath.Join(append([]string{"testdata"}, elem...)...)
	data, err := ioutil.ReadFile(name)
	require.NoError(t, err)
	return string(data)
}
