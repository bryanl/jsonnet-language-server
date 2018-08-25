package locate

import (
	"bytes"
	"fmt"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrUnresolvable means the loctable can't be resolved.
	ErrUnresolvable = errors.New("unresolvable")
)

// Scope is a map of options.
type Scope map[string]Locatable

// Keys lists the keys in the scope.
func (s Scope) Keys() []string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}

	return keys
}

type Resolved struct {
	Location    ast.LocationRange
	Token       interface{}
	Description string
}

type Locatable struct {
	Token  interface{}
	Loc    ast.LocationRange
	Parent *Locatable
	Scope  Scope
}

func (l *Locatable) Resolve(jPaths []string, cache *langserver.NodeCache) (*Resolved, error) {
	if l == nil {
		return nil, errors.Errorf("locatable is nil")
	}

	logrus.Debugf("resolving %T", l.Token)

	switch t := l.Token.(type) {
	case *ast.Var:
		return l.handleVar(t, jPaths, cache)
	case *ast.Index:
		return l.handleIndex(t, cache)
	case ast.Identifier:
		return l.handleDefault(cache, jPaths)
	case *ast.Identifier:
		return l.handleDefault(cache, jPaths)
	case *ast.Import:
		return l.handleImport(t)
	case ast.LocalBind:
		return l.handleLocalBind(t)
	case *ast.Function:
		return l.handleFunction(t)
	case ast.NamedParameter:
		return l.handleNamedParameter(t)
	case astext.RequiredParameter:
		return l.handleRequiredParameter(t)
	default:
		logrus.Errorf("locatable unable to resolve %T", l.Token)
		return nil, ErrUnresolvable
	}
}

func (l *Locatable) handleLocalBind(b ast.LocalBind) (*Resolved, error) {
	return &Resolved{}, nil
}

func (l *Locatable) handleImport(i *ast.Import) (*Resolved, error) {
	resolved := &Resolved{
		Description: astext.TokenName(i),
		Location:    l.Loc,
	}
	return resolved, nil
}

func (l *Locatable) handleIndex(i *ast.Index, cache *langserver.NodeCache) (*Resolved, error) {
	description, err := resolvedIndex(i, cache, l.Scope)
	if err != nil {
		return nil, err
	}

	result := &Resolved{
		Token:       l.Token,
		Description: description,
	}

	return result, nil
}

func (l *Locatable) handleNamedParameter(p ast.NamedParameter) (*Resolved, error) {
	description := fmt.Sprintf("(param) %s", string(p.Name))

	result := &Resolved{
		Location:    l.Loc,
		Token:       l.Token,
		Description: description,
	}

	return result, nil
}

func (l *Locatable) handleRequiredParameter(p astext.RequiredParameter) (*Resolved, error) {
	description := fmt.Sprintf("(param) %s", string(p.ID))

	result := &Resolved{
		Location:    l.Loc,
		Token:       l.Token,
		Description: description,
	}

	return result, nil
}

func (l *Locatable) handleDefault(cache *langserver.NodeCache, jPaths []string) (*Resolved, error) {
	var name string
	var err error

	switch t := l.Parent.Token.(type) {
	case ast.LocalBind:
		name, err = bindOutput(t, cache, l.Scope, jPaths)
	default:
		logrus.Infof("handleDefault: %T", t)
		name = astext.TokenName(l.Token)
	}

	if err != nil {
		return nil, err
	}

	logrus.Debugf("handling default %s (%s): %T, %T", name, l.Loc.String(), l.Token, l.Parent.Token)

	resolved := &Resolved{
		Location:    l.Loc,
		Token:       l.Token,
		Description: name,
	}

	return resolved, nil
}

func bindOutput(bind ast.LocalBind, cache *langserver.NodeCache, scope Scope, jPaths []string) (string, error) {
	var name string

	switch t := bind.Body.(type) {
	case *ast.LiteralString:
		name = "string"
	case *ast.DesugaredObject:
		name = "object"
	case *ast.Function:
		name = "function"
	case *ast.Object:
		return astext.ObjectDescription(t)
	case *ast.Index:
		return resolvedIndex(t, cache, scope)
	case *ast.Var:
		return resolvedVar(t, jPaths, cache, scope)
	default:
		return fmt.Sprintf("(unknown) %s: %T", string(bind.Variable), t), nil
	}

	return fmt.Sprintf("(%s) %s", name, string(bind.Variable)), nil
}

func (l *Locatable) handleFunction(f *ast.Function) (*Resolved, error) {
	var sig bytes.Buffer
	setRequired := false
	for i, p := range f.Parameters.Required {
		setRequired = true
		if _, err := sig.WriteString(string(p)); err != nil {
			return nil, err
		}
		if i <= len(f.Parameters.Required)-2 {
			if _, err := sig.WriteString(", "); err != nil {
				return nil, err
			}
		}
	}

	for i, p := range f.Parameters.Optional {
		if setRequired {
			if _, err := sig.WriteString(", "); err != nil {
				return nil, err
			}
		}

		val := astext.TokenValue(p.DefaultArg)
		s := fmt.Sprintf("%s=%s", string(p.Name), val)
		if _, err := sig.WriteString(s); err != nil {
			return nil, err
		}

		if i <= len(f.Parameters.Optional)-2 {
			if _, err := sig.WriteString(", "); err != nil {
				return nil, err
			}
		}
	}

	switch t := l.Parent.Parent.Token.(type) {
	case ast.DesugaredObjectField:
		name := astext.TokenName(l.Parent.Parent.Token)
		resolved := &Resolved{
			Location:    l.Loc,
			Token:       l.Token,
			Description: fmt.Sprintf("(function) %s(%s)", name, sig.String()),
		}

		return resolved, nil
	default:
		return nil, errors.Errorf("can't handle function in a %T", t)
	}

}

func (l *Locatable) handleVar(t *ast.Var, jPaths []string, cache *langserver.NodeCache) (*Resolved, error) {
	if ref, ok := l.Scope[string(t.Id)]; ok {
		logrus.Debugf("%s points to a %T", t.Id, ref.Token)
		s, err := resolvedIdentifier(ref.Token, jPaths, cache, l.Scope)
		if err != nil {
			return nil, err
		}
		resolved := &Resolved{
			Location:    ref.Loc,
			Token:       ref.Token,
			Description: s,
		}

		return resolved, nil
	}

	return nil, ErrUnresolvable
}

func resolvedVar(t *ast.Var, jPaths []string, cache *langserver.NodeCache, scope Scope) (string, error) {
	if ref, ok := scope[string(t.Id)]; ok {
		logrus.Debugf("%s points to a %T", t.Id, ref.Token)
		return resolvedIdentifier(ref.Token, jPaths, cache, scope)
	}

	return "", ErrUnresolvable
}

func resolvedIdentifier(item interface{}, jPaths []string, cache *langserver.NodeCache, scope Scope) (string, error) {
	switch t := item.(type) {
	case *ast.Import:
		return importDescription(t, jPaths, cache, scope)
	case *ast.Index:
		return resolvedIndex(t, cache, scope)
	case *ast.Object:
		return astext.ObjectDescription(t)
	default:
		logrus.Infof("resolvedIdentifer: unable to resolve %T", t)
		return fmt.Sprintf("resolvedIdentifer %T: %s", t, astext.TokenName(item)), nil
	}
}

func resolvedIndex(i *ast.Index, cache *langserver.NodeCache, scope Scope) (string, error) {
	var indices []string
	var cur ast.Node = i
	for {
		switch t := cur.(type) {
		case *ast.Index:
			cur = t.Target

			if t.Id == nil {
				return "", errors.New("index didn't have an id")
			}
			indices = append([]string{string(*t.Id)}, indices...)
		case *ast.Var:
			varID := string(t.Id)
			if x, ok := scope[varID]; ok {
				logrus.Debugf("it points to a %T", x.Token)

				return describe(x.Token, indices, cache, scope)
			}

			return "", errors.Errorf("could not find %s in scope", varID)
		default:
			return "", errors.Errorf("unable to handle index target of type %T", t)
		}
	}
}

func importDescription(i *ast.Import, jPaths []string, cache *langserver.NodeCache, scope Scope) (string, error) {
	ne, err := cache.Get(i.File.Value)
	if err != nil {
		switch err.(type) {
		case *langserver.NodeCacheMissErr:
			return "node cache miss", nil
		default:
			return "", err
		}
	}

	return resolvedIdentifier(ne.Node, jPaths, cache, scope)
}

func describe(item interface{}, indicies []string, cache *langserver.NodeCache, scope Scope) (string, error) {
	switch t := item.(type) {
	case *ast.Object:
		return describeInObject(t, indicies, cache, scope)
	case *ast.Import:
		ne, err := cache.Get(t.File.Value)
		if err != nil {
			switch err.(type) {
			case *langserver.NodeCacheMissErr:
				return "node cache miss", nil
			default:
				return "", err
			}
		}

		return describe(ne.Node, indicies, cache, scope)
	case *ast.Index:
		spew.Fdump(os.Stderr, t)
		return resolvedIndex(t, cache, scope)
	default:
		logrus.Infof("describe %T", t)
		return astext.TokenName(t), nil
	}
}

func describeInObject(o *ast.Object, indicies []string, cache *langserver.NodeCache, scope Scope) (string, error) {
	if len(indicies) == 0 {
		return astext.ObjectDescription(o)
	}

	for i := range o.Fields {
		f := o.Fields[i]
		if astext.ObjectFieldName(f) != indicies[0] {
			continue
		}

		return describe(f.Expr2, indicies[1:], cache, scope)
	}

	return "", errors.Errorf("unable to find field %q n object", indicies[0])
}
