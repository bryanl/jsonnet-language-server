package token

import (
	"fmt"
	"sort"
	"strings"

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

func (sm *Scope) addEvalScope(es *evalScope) {
	for k, v := range es.store {
		sm.add(k, v)
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

// Keywords returns jsonnet keywords.
func (sm *Scope) Keywords() []string {
	return []string{"assert", "else", "error", "false", "for",
		"function", "if", "import", "importstr", "in", "local",
		"null", "tailstrict", "then", "self", "super", "true"}
}

// GetInPath returns an entry given a path.
func (sm *Scope) GetInPath(path []string) (*ScopeEntry, error) {
	id, path := path[0], path[1:]

	if id == "std" {
		return resolveStd(path)
	}

	e, err := sm.Get(id)
	if err != nil {
		return nil, errors.Wrapf(err, "current path [%s]",
			strings.Join(path, ","))
	}

	if len(path) == 0 {
		return e, nil
	}

	node, err := sm.findInPath(e.Node, path)
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
	o, ok := node.(*ast.Object)
	if !ok {
		return nil, errors.Errorf("not a regular object: %T", node)
	}

	id, path := path[0], path[1:]

	var fieldNames []string

	for i := range o.Fields {
		field := o.Fields[i]

		var name string
		switch field.Kind {
		case ast.ObjectFieldID:
			if field.Id == nil {
				return nil, errors.New("field id shouldn't be nil")
			}
			name = string(*field.Id)
		case ast.ObjectFieldStr:
			if field.Expr1 == nil {
				return nil, errors.New("field id should be a string")
			}
			name = astext.TokenValue(field.Expr1)
		}

		fieldNames = append(fieldNames, name)

		if name != id {
			continue
		}

		if len(path) == 0 {
			return field.Expr2, nil
		}

		return findInObject(field.Expr2, path)
	}

	return nil, errors.Errorf("unable to find field %q in [%s]",
		id, strings.Join(fieldNames, ","))

}

func findInDesugaredObject(node ast.Node, path []string) (ast.Node, error) {
	o, ok := node.(*ast.DesugaredObject)
	if !ok {
		return nil, errors.Errorf("not a desugared object: %T", node)
	}

	id, path := path[0], path[1:]

	var fieldNames []string

	for i := range o.Fields {
		field := o.Fields[i]

		name, ok := field.Name.(*ast.LiteralString)
		if !ok {
			return nil, errors.New("field name was not a string")
		}

		fieldNames = append(fieldNames, name.Value)

		if name.Value != id {
			continue
		}

		var body ast.Node
		// field body can be a local or a desugared object
		switch n := field.Body.(type) {
		case *ast.Local:
			body = n.Body
		case *ast.DesugaredObject:
			body = n
		default:
			return n, nil
		}

		if len(path) == 0 {
			return body, nil
		}

		return findInDesugaredObject(body, path)
	}

	return nil, errors.Errorf("desugared: unable to find field %q in [%s]",
		id, strings.Join(fieldNames, ","))
}

func (sm *Scope) findInPath(node ast.Node, path []string) (ast.Node, error) {
	switch node := node.(type) {
	case *ast.DesugaredObject:
		return findInDesugaredObject(node, path)
	case *ast.Object:
		return findInObject(node, path)
	case *ast.Index:
		v, indexPath := resolveIndex(node)
		if v == nil {
			logrus.Infof("findInPath for index. v is nil. got indexPath [%s]",
				strings.Join(indexPath, ","))
		}
		o, err := sm.Get(string(v.Id))
		if err != nil {
			return nil, err
		}

		path = append(indexPath[1:], path...)
		return sm.findInPath(o.Node, path)
	default:
		return nil, errors.Errorf("not an object %T: [%s]",
			node, strings.Join(path, ","))
	}
}

// Get retrieves an entry by name from the scope.
func (sm *Scope) Get(key string) (*ScopeEntry, error) {
	se, ok := sm.store[key]
	if !ok {
		var keys []string
		for k := range sm.store {
			keys = append(keys, k)
		}
		return nil, errors.Errorf("scope does not contain %q (%s)",
			key, strings.Join(keys, ","))
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

func ReadSource(filename, source string, ch chan<- ParseDiagnostic) (ast.Node, error) {
	node, err := Parse(filename, source, ch)
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

	return node, nil
}

// LocationScope finds the free variables for a location.
func LocationScope(filename, source string, loc jlspos.Position, nodeCache *NodeCache) (*Scope, error) {
	node, err := ReadSource(filename, source, nil)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, loc)
	if err != nil {
		return nil, err
	}

	es, err := eval(node, found, nodeCache)
	if err != nil {
		return nil, err
	}

	sm := newScope(nodeCache)
	sm.addEvalScope(es)

	return sm, nil
}

func resolveIndex(i *ast.Index) (*ast.Var, []string) {
	var cur ast.Node = i
	var v *ast.Var
	count := 0
	done := false
	var path []string
	for count < 100 && !done {
		switch c := cur.(type) {
		case *ast.Apply:
			cur = c.Target
		case *ast.Index:
			if c.Index != nil {
				s, ok := c.Index.(*ast.LiteralString)
				if ok {
					path = append([]string{s.Value}, path...)
				}

			}
			cur = c.Target
		case *ast.Self:
			path = append([]string{"self"}, path...)
			done = true
		case *ast.Var:
			v = c
			path = append([]string{string(c.Id)}, path...)
			done = true
		default:
			panic(fmt.Sprintf("unable to resolve a %T in index", c))
		}

		count++
	}

	return v, path
}

func resolveStd(path []string) (*ScopeEntry, error) {
	return nil, errors.Errorf("resolveStd for %s not implemented",
		strings.Join(path, "."))
}
