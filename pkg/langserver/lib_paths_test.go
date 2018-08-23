package langserver

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LibPaths_Files(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	paths := []string{
		"path1/file3.jsonnet",
		"path1/file2.libsonnet",
		"path2/file1.libsonnet",
	}

	for _, path := range paths {
		createFile(t, dir, path)
	}

	libPaths := []string{
		filepath.Join(dir, "path1"),
		filepath.Join(dir, "path2"),
	}

	lp := NewLibPaths(libPaths)

	got, err := lp.Files()
	require.NoError(t, err)

	expected := []string{"file1.libsonnet", "file2.libsonnet", "file3.jsonnet"}
	assert.Equal(t, expected, got)
}

func createFile(t *testing.T, base, path string) {
	dir, file := filepath.Split(path)

	err := os.MkdirAll(filepath.Join(base, dir), 0700)
	require.NoError(t, err)

	file = filepath.Join(base, dir, file)
	err = ioutil.WriteFile(file, []byte(""), 0600)
	require.NoError(t, err)
}
