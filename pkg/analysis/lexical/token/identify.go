package token

import (
	"bytes"
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/static"
	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/ksonnet/ksonnet-lib/ksonnet-gen/printer"
	"github.com/pkg/errors"
)

// IdentifyConfig is configuration for Identify.
type IdentifyConfig struct {
	JsonnetLibPaths []string
	ExtVar          map[string]string
	ExtCode         map[string]string
	TLACode         map[string]string
	TLAVar          map[string]string
}

// VM create a jsonnet VM using IdentifyConfig.
func (ic *IdentifyConfig) VM() *jsonnet.VM {
	vm := jsonnet.MakeVM()

	importer := &jsonnet.FileImporter{
		JPaths: ic.JsonnetLibPaths,
	}

	vm.Importer(importer)

	for k, v := range ic.ExtVar {
		vm.ExtVar(k, v)
	}
	for k, v := range ic.ExtCode {
		vm.ExtCode(k, v)
	}
	for k, v := range ic.TLACode {
		vm.TLACode(k, v)
	}
	for k, v := range ic.TLAVar {
		vm.TLAVar(k, v)
	}

	return vm
}

// Identify identifies what is at a position.
func Identify(filename, source string, pos jlspos.Position, nodeCache *NodeCache, config IdentifyConfig) (fmt.Stringer, error) {
	node, err := Parse(filename, source, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parse source")
	}

	if err = DesugarFile(&node); err != nil {
		return nil, errors.Wrap(err, "desugar node")
	}

	err = static.Analyze(node)
	if err != nil {
		return nil, errors.Wrap(err, "analyze node")
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

func (i *identifier) identify(n ast.Node) (fmt.Stringer, error) {
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

		return astext.NewItem(evaluated), nil

	case *ast.Index:
		return i.index(n)
	case *ast.Local:
		return i.local(n)
	case *ast.Var:
		return i.variable(n)
	case *ast.Function, *ast.Object:
		return astext.NewItem(n), nil
	case nil, *astext.Partial:
		return IdentifyNoMatch, nil
	case *ast.Array, *ast.DesugaredObject, *ast.Import,
		*ast.LiteralBoolean, *ast.LiteralNumber, *ast.LiteralString:
		return astext.NewItem(n), nil
	default:
		panic(fmt.Sprintf("unable to identify %T", n))
	}
}

func (i *identifier) index(idx *ast.Index) (fmt.Stringer, error) {
	v, path := resolveIndex(idx)

	vSe, err := i.scope.Get(string(v.Id))
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

		return astext.NewItem(evaluated), nil
	}

	se, err := i.scope.GetInPath(path)
	if err != nil {
		return nil, err
	}

	return i.identify(se.Node)
}

func (i *identifier) variable(v *ast.Var) (fmt.Stringer, error) {
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
			return astext.NewItem(x), nil
		}
	}

	return IdentifyNoMatch, nil
}

var (
	// IdentifyNoMatch is a no match.
	IdentifyNoMatch = &emptyItem{}
)

func (i *identifier) local(local *ast.Local) (fmt.Stringer, error) {
	for _, bind := range local.Binds {
		if i.pos.IsInJsonnetRange(bind.VarLoc) {
			switch n := bind.Body.(type) {
			case *ast.Import:
				ne, err := i.nodeCache.Get(n.File.Value)
				if err == nil {
					return astext.NewItem(ne.Node), nil
				}
			case *ast.LiteralString, *ast.LiteralBoolean, *ast.LiteralNumber,
				*ast.LiteralNull, *ast.Function:
				return astext.NewItem(bind.Body), nil
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
				return astext.NewItem(evaluated), nil
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

	fmt.Println(buf.String())

	// evaluate node and manifest value to node
	evaluated, err := vm.EvaluateToNode("snippet.jsonnet", buf.String())
	if err != nil {
		return nil, err
	}

	return evaluated, nil
}

type emptyItem struct{}

var _ fmt.Stringer = (*emptyItem)(nil)

func (ei *emptyItem) String() string {
	return ""
}
