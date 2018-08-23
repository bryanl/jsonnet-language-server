package config

import "github.com/bryanl/jsonnet-language-server/pkg/lsp"

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
