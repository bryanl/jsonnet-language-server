// +build integration

package lexical_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLexical(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lexical Suite")
}
