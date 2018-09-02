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
)

type IdentifyConfig struct {
	JsonnetLibPaths []string
	ExtVar          map[string]string
	ExtCode         map[string]string
	TLACode         map[string]string
	TLAVar          map[string]string
}

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
		return nil, err
	}

	if err = DesugarFile(&node); err != nil {
		return nil, err
	}

	err = static.Analyze(node)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, pos)
	if err != nil {
		return nil, err
	}

	es, err := eval(node, found, nodeCache)
	if err != nil {
		return nil, err
	}

	scope := newScope(nodeCache)
	scope.addEvalScope(es)

	i := identifier{
		n:         found,
		pos:       pos,
		scope:     scope,
		es:        es,
		nodeCache: nodeCache,
		config:    config,
	}

	return i.identify()

	// switch n := found.(type) {
	// case *ast.Index:
	// 	return identifyIndex(n, scope)
	// case *ast.Local:
	// 	return identifyLocal(n, pos, nodeCache, config)
	// case *ast.Var:
	// 	return identifyVar(n, es)
	// case nil, *ast.Array, *ast.DesugaredObject, *ast.Import,
	// 	*ast.LiteralBoolean, *ast.LiteralNumber, *ast.LiteralString,
	// 	*astext.Partial:
	// 	return IdentifyNoMatch, nil

	// default:
	// 	panic(fmt.Sprintf("unable to identify %T", n))
	// }
}

type identifier struct {
	n         ast.Node
	pos       jlspos.Position
	scope     *Scope
	es        *evalScope
	nodeCache *NodeCache
	config    IdentifyConfig
}

func (i *identifier) clone(n ast.Node) identifier {
	return identifier{
		n:         n,
		pos:       i.pos,
		scope:     i.scope,
		es:        i.es,
		nodeCache: i.nodeCache,
		config:    i.config,
	}
}

func (i *identifier) identify() (fmt.Stringer, error) {
	switch n := i.n.(type) {

	case *ast.Index:
		return i.index(n)
	case *ast.Local:
		return i.local(n)
	case *ast.Var:
		return i.variable(n)
	case *ast.Apply, *ast.Object:
		return astext.NewItem(i.n), nil
	case nil, *ast.Array, *ast.DesugaredObject, *ast.Import,
		*ast.LiteralBoolean, *ast.LiteralNumber, *ast.LiteralString,
		*astext.Partial:
		return IdentifyNoMatch, nil

	default:
		panic(fmt.Sprintf("unable to identify %T", n))
	}
}

func (i *identifier) index(idx *ast.Index) (fmt.Stringer, error) {
	_, path := resolveIndex(idx)

	if path[0] == "std" {
		evaluated, err := evaluateNode(idx, i.config.VM())
		if err != nil {
			return nil, err
		}

		return astext.NewItem(evaluated), nil
	}

	se, err := i.scope.GetInPath(path)
	if err != nil {
		return nil, err
	}

	return astext.NewItem(se.Node), nil
}

func (i *identifier) variable(v *ast.Var) (fmt.Stringer, error) {
	x, ok := i.es.store[v.Id]
	if ok {
		switch v := x.(type) {
		case *ast.Index:
			ptr := i.clone(v)
			return ptr.identify()
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
			case *ast.Self:
				return IdentifyNoMatch, nil
			default:
				evaluated, err := evaluateNode(n, i.config.VM())
				if err != nil {
					return nil, err
				}

				// report on evaluated node
				return astext.NewItem(evaluated), nil
			}
		}
	}

	return IdentifyNoMatch, nil
}

func evaluateNode(node ast.Node, vm *jsonnet.VM) (ast.Node, error) {
	// convert node to a snippet
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, node); err != nil {
		return nil, err
	}

	// evaluate node and manifest value to node
	return vm.EvaluateToNode("snippet.jsonnet", buf.String())
}

type emptyItem struct{}

var _ fmt.Stringer = (*emptyItem)(nil)

func (ei *emptyItem) String() string {
	return ""
}
