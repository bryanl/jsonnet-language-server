package token

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	err := Scope("file.jsonnet", `local a="a";local b="b";`, createLoc(1, 2))
	require.Error(t, err)
}
