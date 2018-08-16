// +build integration

package lexical_test

import (
	"io"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sourcegraph/go-langserver/pkg/lsp"

	. "github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
)

var _ = Describe("Lexical", func() {
	var (
		sourceReader io.Reader
	)

	BeforeEach(func() {
		data := jlstesting.Testdata(GinkgoT(), "lexical", "example2.jsonnet")
		sourceReader = strings.NewReader(data)
	})

	Describe("Hover At Location", func() {
		var (
			hoverResponse *lsp.Hover
			hoverError    error

			line   int
			column int
		)

		JustBeforeEach(func() {
			hoverResponse, hoverError = HoverAtLocation("example2.jsonnet", sourceReader, line, column)
		})

		Context("import", func() {
			BeforeEach(func() {
				line = 1
				column = 13
			})

			It("creates a response", func() {
				Expect(hoverError).ToNot(HaveOccurred())

				expected := &lsp.Hover{
					Contents: []lsp.MarkedString{
						{
							Language: "jsonnet",
							Value:    "(import) data.jsonnet",
						},
					},
					Range: newRange(1, 11, 1, 32),
				}

				Expect(hoverResponse).To(Equal(expected))
			})
		})

		Context("item points object", func() {
			BeforeEach(func() {
				line = 11
				column = 13
			})

			It("creates a response", func() {
				Expect(hoverError).ToNot(HaveOccurred())

				expected := &lsp.Hover{
					Contents: []lsp.MarkedString{
						{
							Language: "jsonnet",
							Value:    "(object) {\n  (field) nested1:,\n}",
						},
					},
					Range: newRange(3, 7, 3, 7),
				}

				Expect(hoverResponse).To(Equal(expected))
			})
		})

		Context("object", func() {
			BeforeEach(func() {
				line = 3
				column = 7
			})

			It("creates a response", func() {
				Expect(hoverError).ToNot(HaveOccurred())

				expected := &lsp.Hover{
					Contents: []lsp.MarkedString{
						{
							Language: "jsonnet",
							Value:    "(object) {\n  (field) nested1:,\n}",
						},
					},
					Range: newRange(3, 7, 3, 7),
				}

				Expect(hoverResponse).To(Equal(expected))
			})
		})
	})
})

func newRange(sl, sc, el, ec int) lsp.Range {
	return lsp.Range{
		Start: newPosition(sl, sc),
		End:   newPosition(el, ec),
	}
}

func newPosition(l, c int) lsp.Position {
	return lsp.Position{Line: l - 1, Character: c - 1}
}
