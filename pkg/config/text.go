package config

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

// TextDocument is a document's text and and metadata.
type TextDocument struct {
	uri        string
	languageID string
	version    int
	text       string
}

func NewTextDocument(uri, text string) TextDocument {
	return TextDocument{
		uri:  uri,
		text: text,
	}
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

func (td *TextDocument) Filename() (string, error) {
	return uri.ToPath(td.uri)
}

// Truncate returns text truncated at a position.
func (td *TextDocument) Truncate(line, col int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(td.text))
	scanner.Split(bufio.ScanBytes)

	var buf bytes.Buffer

	c := 0
	l := 1

	for scanner.Scan() {
		c++

		t := scanner.Text()

		_, err := buf.WriteString(t)
		if err != nil {
			return "", err
		}

		if l == line && c == col {
			break
		}

		if t == "\n" {
			l++
			c = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
}
