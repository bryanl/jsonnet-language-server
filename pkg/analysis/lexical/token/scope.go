package token

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/static"
	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ScopeEntry is a scope entry.
type ScopeEntry struct {
	Detail        string
	Documentation string
	Node          ast.Node
}

// Scope is scope.
type Scope struct {
	store map[string]ScopeEntry
}

func newScope() *Scope {
	return &Scope{
		store: make(map[string]ScopeEntry),
	}
}

// Keys lists keys in the scope.
func (sm *Scope) Keys() []string {
	var keys []string
	for k := range sm.store {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (sm *Scope) Keywords() []string {
	return []string{"assert", "else", "error", "false", "for",
		"function", "if", "import", "importstr", "in", "local",
		"null", "tailstrict", "then", "self", "super", "true"}
}

func (sm *Scope) GetInPath(path []string) (*ScopeEntry, error) {
	id, path := path[0], path[1:]

	e, err := sm.Get(id)
	if err != nil {
		return nil, err
	}

	if len(path) == 0 {
		return e, nil
	}

	node, err := findInObject(e.Node, path)
	if err != nil {
		return nil, err
	}

	text := astext.TokenName(node)

	return &ScopeEntry{
		Node:   node,
		Detail: text,
	}, nil
}

func findInObject(node ast.Node, path []string) (ast.Node, error) {
	o, ok := node.(*ast.DesugaredObject)
	if !ok {
		return nil, errors.New("node was not an object")
	}

	id, path := path[0], path[1:]

	for i := range o.Fields {
		field := o.Fields[i]

		name, ok := field.Name.(*ast.LiteralString)
		if !ok {
			return nil, errors.New("field name was not a string")
		} else if name.Value != id {
			continue
		}

		if len(path) == 0 {
			local, ok := field.Body.(*ast.Local)
			if !ok {
				return nil, errors.New("field body wasn't a local")
			}

			logrus.Info("found body")
			return local.Body, nil
		}

		return findInObject(field.Body, path)
	}

	return nil, errors.Errorf("unable to find field %q", id)
}

// Get retrieves an entry by name from the scope.
func (sm *Scope) Get(key string) (*ScopeEntry, error) {
	se, ok := sm.store[key]
	if !ok {
		return nil, errors.Errorf("scope does not contain %q", key)
	}

	return &se, nil
}

func (sm *Scope) add(key ast.Identifier, node ast.Node) {
	id := string(key)
	sm.store[id] = ScopeEntry{
		Detail: id,
		Node:   node,
	}
}

// LocationScope finds the free variables for a location.
func LocationScope(filename, source string, loc jlspos.Position) (*Scope, error) {
	node, err := Parse(filename, source)
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

	logrus.Infof("locating scope at %s", loc.String())
	found, err := locate(node, loc.ToJsonnet())
	if err != nil {
		return nil, err
	}

	sm := newScope()
	es := eval(node, found)
	for k, v := range es {
		sm.add(k, v)
	}

	return sm, nil
}

type scopeCatalog struct {
	ids      ast.IdentifierSet
	store    map[string]ast.Node
	parent   *scopeCatalog
	children map[ast.Node]*scopeCatalog
}

func newScopeCatalog(ids ...ast.Identifier) *scopeCatalog {
	return &scopeCatalog{
		ids:      ast.NewIdentifierSet(ids...),
		store:    make(map[string]ast.Node),
		children: make(map[ast.Node]*scopeCatalog),
	}
}

func (sc *scopeCatalog) Clone(node ast.Node) *scopeCatalog {
	child := &scopeCatalog{
		ids:      sc.ids.Clone(),
		store:    make(map[string]ast.Node),
		children: make(map[ast.Node]*scopeCatalog),
		parent:   sc,
	}

	sc.children[node] = child

	for k, v := range sc.store {
		child.store[k] = v
	}

	return child
}

func resolveIndex(i *ast.Index, path []string) (ast.Identifier, []string) {
	if i.Target != nil {
		switch v := i.Target.(type) {
		case *ast.Index:
			path = append(path, string(*i.Id))
			resolveIndex(v, path)
		case *ast.Var:
			return v.Id, path
		}
	} else if i.Id != nil {
		// not sure what do here, so panic
		panic("unable to handle index with index")
	}

	panic("index target and index were nil")
}

func (sc *scopeCatalog) Add(i ast.Identifier, node ast.Node) bool {
	switch v := node.(type) {
	case *ast.Index:
		fmt.Println("started with", i)
		path := []string{}
		i, _ = resolveIndex(v, path)
		fmt.Println("got", i)
	case *ast.Local:
		for _, bind := range v.Binds {
			if bind.Variable == i {
				node = bind.Body
			}
		}
	case *ast.Var:
		fmt.Printf("found var and it points to %s\n", string(v.Id))
	default:
		fmt.Printf("Not sure how to add id of type %T\n", node)
	}

	id := string(i)
	if pc, file, line, ok := runtime.Caller(1); ok {
		funcName := runtime.FuncForPC(pc).Name()
		fmt.Printf("adding [%s] -> %T at %s:%v:%s\n",
			string(i), node, filepath.Base(file), line, filepath.Base(funcName))
	}

	sc.store[id] = node
	isAdded := sc.ids.Add(i)
	return isAdded
}

func (sc *scopeCatalog) Contains(i ast.Identifier) bool {
	return sc.ids.Contains(i)
}

func (sc *scopeCatalog) FreeVariables() ast.Identifiers {
	return sc.ids.ToOrderedSlice()
}
