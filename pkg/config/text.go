package config

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
)

// TextDocument is a document's text and and metadata.
type TextDocument struct {
	uri        string
	languageID string
	version    int
	text       string
}

// NewTextDocumentFromItem creates a TextDocument from a lsp TextDocumentItem.
func NewTextDocumentFromItem(tdi lsp.TextDocumentItem) TextDocument {
	return TextDocument{
		uri:        tdi.URI,
		languageID: tdi.LanguageID,
		text:       tdi.Text,
		version:    tdi.Version,
	}
}

// URI returns the URI for the text document.
func (td *TextDocument) URI() string {
	return td.uri
}

func (td *TextDocument) String() string {
	return td.text
}

// Truncate returns text truncated at a position.
func (td *TextDocument) Truncate(line, col int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(td.text))
	scanner.Split(bufio.ScanRunes)

	var buf bytes.Buffer

	c := 0
	l := 1

	for scanner.Scan() {
		c++

		t := scanner.Text()

		if t == "\n" {
			l++
			c = 0
		}

		_, err := buf.WriteString(t)
		if err != nil {
			return "", err
		}

		if l == line && c == col {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
