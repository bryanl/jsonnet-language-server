package token

import (
	"bytes"
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/ksonnet/ksonnet-lib/ksonnet-gen/printer"
	"github.com/pkg/errors"
)

// Item is something that can identified.
type Item struct {
	token interface{}
}

var _ Identity = (*Item)(nil)

// NewItem creates an instance of Item.
func NewItem(token interface{}) *Item {
	return &Item{
		token: token,
	}
}

func (i *Item) String() string {
	return astext.TokenName(i.token)
}

// Signature is the item's signature if it is a function.
func (i *Item) Signature() *Signature {
	return nil
}

// Signature is a function signature.
type Signature struct {
	label         string
	documentation string
	parameters    []string
}

// Label is the function signature label.
func (s *Signature) Label() string {
	return s.label
}

// Documentation is documentation for the function.
func (s *Signature) Documentation() string {
	return s.documentation
}

// Parameters are parameters for the function.
func (s *Signature) Parameters() []string {
	return s.parameters
}

type Identity interface {
	Signature() *Signature
	String() string
}

// Identify identifies what is at a position.
func Identify(source string, pos jlspos.Position, nodeCache *NodeCache, config IdentifyConfig) (Identity, error) {
	node, err := ReadSource(config.path, source, nil)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, pos)
	if err != nil {
		return nil, errors.Wrap(err, "locate node at position")
	}

	es, err := eval(node, found, nodeCache)
	if err != nil {
		return nil, errors.Wrap(err, "find scope for node")
	}

	scope := newScope(nodeCache)
	scope.addEvalScope(es)

	i := identifier{
		sourceNode: node,
		pos:        pos,
		scope:      scope,
		es:         es,
		nodeCache:  nodeCache,
		config:     config,
	}

	return i.identify(found)
}

type identifier struct {
	sourceNode ast.Node
	pos        jlspos.Position
	scope      *Scope
	es         *evalScope
	nodeCache  *NodeCache
	config     IdentifyConfig
}

func (i *identifier) clone() identifier {
	return identifier{
		pos:       i.pos,
		scope:     i.scope,
		es:        i.es,
		nodeCache: i.nodeCache,
		config:    i.config,
	}
}

func (i *identifier) identify(n ast.Node) (Identity, error) {
	switch n := n.(type) {

	case *ast.Apply:
		stub, err := buildEvalStub(n, i.scope)
		if err != nil {
			return nil, err
		}

		evaluated, err := evaluateNode(stub, i.config.VM())
		if err != nil {
			return nil, errors.Wrap(err, "evaluate apply")
		}

		return NewItem(evaluated), nil

	case *ast.Index:
		return i.index(n)
	case *ast.Local:
		return i.local(n)
	case *ast.Var:
		return i.variable(n)
	case *ast.Function, *ast.Object:
		return NewItem(n), nil
	case nil, *astext.Partial:
		return IdentifyNoMatch, nil
	case *ast.Array, *ast.DesugaredObject, *ast.Import,
		*ast.LiteralBoolean, *ast.LiteralNumber, *ast.LiteralString:
		return NewItem(n), nil
	default:
		panic(fmt.Sprintf("unable to identify %T", n))
	}
}

func (i *identifier) index(idx *ast.Index) (Identity, error) {
	path := resolveIndex(idx)

	vSe, err := i.scope.Get(path[0])
	if err != nil {
		return nil, err
	}

	_, ok := vSe.Node.(*ast.Index)

	if path[0] == "std" || ok {
		stub, err := buildEvalStub(idx, i.scope)
		if err != nil {
			return nil, err
		}

		evaluated, err := evaluateNode(stub, i.config.VM())
		if err != nil {
			return nil, errors.Wrap(err, "evaluate node in index")
		}

		return NewItem(evaluated), nil
	}

	se, err := i.scope.GetInPath(path)
	if err != nil {
		return nil, err
	}

	return i.identify(se.Node)
}

func (i *identifier) variable(v *ast.Var) (Identity, error) {
	es, err := eval(i.sourceNode, v, i.nodeCache)
	if err != nil {
		return nil, err
	}

	x, ok := es.store[v.Id]
	if ok {
		switch v := x.(type) {
		case *ast.Index:
			ptr := i.clone()
			return ptr.identify(v)
		default:
			return NewItem(x), nil
		}
	}

	return IdentifyNoMatch, nil
}

var (
	// IdentifyNoMatch is a no match.
	IdentifyNoMatch = &emptyItem{}
)

func (i *identifier) local(local *ast.Local) (Identity, error) {
	for _, bind := range local.Binds {
		if i.pos.IsInJsonnetRange(bind.VarLoc) {
			switch n := bind.Body.(type) {
			case *ast.Import:
				ne, err := i.nodeCache.Get(n.File.Value)
				if err == nil {
					return NewItem(ne.Node), nil
				}
			case *ast.LiteralString, *ast.LiteralBoolean, *ast.LiteralNumber,
				*ast.LiteralNull, *ast.Function:
				return NewItem(bind.Body), nil
			case *ast.Self:
				return IdentifyNoMatch, nil
			default:
				stub, err := buildEvalStub(n, i.scope)
				if err != nil {
					return nil, err
				}

				evaluated, err := evaluateNode(stub, i.config.VM())
				if err != nil {
					return nil, errors.Wrap(err, "evaluate node")
				}

				// report on evaluated node
				return NewItem(evaluated), nil
			}
		}
	}

	return IdentifyNoMatch, nil
}

func buildEvalStub(n ast.Node, scope *Scope) (*ast.Local, error) {
	stub := &ast.Local{
		Binds: ast.LocalBinds{},
	}
	for _, id := range scope.Keys() {
		if id == "std" {
			continue
		}
		se, err := scope.Get(id)
		if err != nil {
			return nil, err
		}

		bind := ast.LocalBind{
			Variable: ast.Identifier(id),
			Body:     se.Node,
		}

		stub.Binds = append(stub.Binds, bind)

	}

	stub.Body = n
	do, ok := n.(*ast.DesugaredObject)
	if ok {
		for _, field := range do.Fields {
			bodyLocal, ok := field.Body.(*ast.Local)
			if ok {
				for _, bind := range bodyLocal.Binds {
					if string(bind.Variable) == "$" {
						continue
					}

					stub.Binds = append(stub.Binds, bind)
				}
			}
		}
	}

	// if binds are empty, add a null value so
	// it can be printed properly.
	if len(stub.Binds) == 0 {
		stub.Binds = append(stub.Binds, ast.LocalBind{
			Variable: ast.Identifier("__unused"),
			Body:     &ast.LiteralNull{},
		})
	}

	return stub, nil
}

func evaluateNode(node ast.Node, vm *jsonnet.VM) (ast.Node, error) {
	// convert node to a snippet
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, node); err != nil {
		return nil, err
	}

	// evaluate node and manifest value to node
	evaluated, err := vm.EvaluateToNode("snippet.jsonnet", buf.String())
	if err != nil {
		return nil, err
	}

	return evaluated, nil
}

type emptyItem struct{}

var _ Identity = (*emptyItem)(nil)

func (ei *emptyItem) String() string {
	return ""
}

func (ei *emptyItem) Signature() *Signature {
	return nil
}
