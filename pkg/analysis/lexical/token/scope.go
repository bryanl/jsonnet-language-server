package token

import (
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
	nodeCache *NodeCache
	store     map[string]ScopeEntry
}

func newScope(nc *NodeCache) *Scope {
	return &Scope{
		store:     make(map[string]ScopeEntry),
		nodeCache: nc,
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
		return nil, errors.Errorf("node was not an object")
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
func LocationScope(filename, source string, loc jlspos.Position, nodeCache *NodeCache) (*Scope, error) {
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
	found, err := locateNode(node, loc.ToJsonnet())
	if err != nil {
		return nil, err
	}

	sm := newScope(nodeCache)
	es, err := eval(node, found, nodeCache)
	if err != nil {
		return nil, err
	}

	for k, v := range es.store {
		sm.add(k, v)
	}

	return sm, nil
}
