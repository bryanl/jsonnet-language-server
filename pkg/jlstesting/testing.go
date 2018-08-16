package jlstesting

import (
	"io/ioutil"
	"path/filepath"

	"github.com/stretchr/testify/require"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Skip(args ...interface{})
	SkipNow()
	Skipf(format string, args ...interface{})
	Skipped() bool
}

func Testdata(t TestingT, elem ...string) string {
	name := filepath.Join(append([]string{"testdata"}, elem...)...)
	data, err := ioutil.ReadFile(name)
	require.NoError(t, err)
	return string(data)
}
