package token

import (
	"sort"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// ScopeEntry is a scope entry.
type ScopeEntry struct {
	Detail        string
	Documentation string
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

// Get retrieves an entry by name from the scope.
func (sm *Scope) Get(key string) (*ScopeEntry, error) {
	se, ok := sm.store[key]
	if !ok {
		return nil, errors.Errorf("scope does not contain %q", key)
	}

	return &se, nil
}

func (sm *Scope) addIdentifier(key ast.Identifier) {
	id := string(key)
	sm.store[id] = ScopeEntry{Detail: id}
}

// LocationScope finds the free variables for a location.
func LocationScope(filename, source string, loc ast.Location) (*Scope, error) {
	node, err := Parse(filename, source)
	if err != nil {
		partialNode, isPartial := isPartialNode(err)

		if !isPartial {
			return nil, err
		}

		node = partialNode
	}

	if err = analyze(node); err != nil {
		return nil, err
	}

	found, err := locate(node, loc)
	if err != nil {
		return nil, err
	}

	sm := newScope()

	for _, id := range found.FreeVariables() {
		sm.addIdentifier(id)
	}

	return sm, nil
}
