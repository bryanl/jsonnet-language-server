package lexical

import (
	"io"
	"io/ioutil"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/pkg/errors"
)

// SourceVisitor visitors items in a reader.
type SourceVisitor struct {
	*NodeVisitor

	Source []byte
}

// NewSourceVisitor creates an instance of SourceVisitor.
func NewSourceVisitor(filename string, r io.Reader, pv PreVisit) (*SourceVisitor, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading source")
	}

	node, err := jsonnet.SnippetToAST(filename, string(data))
	if err != nil {
		return nil, errors.Wrap(err, "parsing source")
	}

	v := NewNodeVisitor(node, nil, Env{}, pv)

	return &SourceVisitor{
		NodeVisitor: v,
		Source:      data,
	}, nil
}
