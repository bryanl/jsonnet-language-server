package token

import (
	"bytes"
	"fmt"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/ksonnet/ksonnet-lib/ksonnet-gen/printer"
	"github.com/pkg/errors"
)

// SignatureResponse is the response from SignatureHelper.
type SignatureResponse struct {
	Label      string
	Parameters []string
}

// SignatureHelper retrieves the signature for a function at a position.
func SignatureHelper(source string, pos jpos.Position, nodeCache *NodeCache) (*SignatureResponse, error) {
	node, err := ReadSource("snippet.jsonnet", source, nil)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, pos)
	if err != nil {
		return nil, err
	}

	apply, ok := found.(*ast.Apply)
	if !ok {
		return nil, nil
	}

	es, err := eval(node, apply, nodeCache)
	if err != nil {
		return nil, err
	}

	s := newScope(nodeCache)
	s.addEvalScope(es)

	var se *ScopeEntry
	var name string
	switch n := apply.Target.(type) {
	case *ast.Var:
		name = string(n.Id)
		se, err = s.Get(name)
		if err != nil {
			return nil, err
		}
	case *ast.Index:
		_, path := resolveIndex(n)
		se, err = s.GetInPath(path)
		if err != nil {
			return nil, err
		}

		name = string(path[len(path)-1])
	}

	funNode, ok := se.Node.(*ast.Function)
	if !ok {
		return nil, errors.New("node was not a function")
	}

	var label bytes.Buffer
	fmt.Fprintf(&label, "%s(", name)

	var params []string

	required := funNode.Parameters.Required
	for i, p := range required {
		label.WriteString(string(p))
		if i < len(required)-1 {
			label.WriteString(", ")
		}

		params = append(params, string(p))
	}

	optional := funNode.Parameters.Optional
	if len(optional) > 0 && len(required) > 0 {
		label.WriteString(", ")
	}

	for i, p := range optional {
		var nodeBuf bytes.Buffer
		if err = printer.Fprint(&nodeBuf, p.DefaultArg); err != nil {
			return nil, err
		}

		fmt.Fprintf(&label, "%s=%s", string(p.Name), nodeBuf.String())

		if i < len(optional)-1 {
			label.WriteString(", ")
		}

		params = append(params, string(p.Name))

	}

	label.WriteString(")")

	sr := &SignatureResponse{
		Label:      label.String(),
		Parameters: params,
	}

	return sr, nil
}
