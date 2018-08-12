package lexical

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

type Resolved struct {
	Location    ast.LocationRange
	Token       interface{}
	Description string
}

type Locatable struct {
	Token  interface{}
	Loc    ast.LocationRange
	Parent *Locatable
	Env    Env
}

func (l *Locatable) Resolve() (*Resolved, error) {
	var resolved *Resolved
	var err error

	switch t := l.Token.(type) {
	case *ast.Var:
		resolved, err = l.handleVar(t)
	default:
		logrus.Errorf("unable to resolve %T", l.Token)
	}

	if err != nil {
		return nil, err
	}

	name, err := tokenName(l.Token)
	if err != nil {
		return nil, err
	}

	if resolved == nil {
		resolved = &Resolved{
			Location:    l.Loc,
			Token:       l.Token,
			Description: name,
		}
	}

	return resolved, nil
}

func (l *Locatable) handleVar(t *ast.Var) (*Resolved, error) {
	if ref, ok := l.Env[string(t.Id)]; ok {
		s, err := l.resolvedIdentifier(&ref)
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

	return nil, nil
}

func (l *Locatable) resolvedIdentifier(ref *Locatable) (string, error) {
	id, ok := ref.Token.(ast.Identifier)
	if !ok {
		return tokenName(ref.Token)
	}

	switch ref.Parent.Token.(type) {
	case ast.LocalBind:
		return fmt.Sprintf("(function) %s()", string(id)), nil
	default:
		return tokenName(ref.Token)
	}

}

func (l *Locatable) IsFunctionParam() bool {
	if _, isVar := l.Token.(*ast.Var); isVar {
		if _, isParentLocal := l.Parent.Token.(*ast.Local); isParentLocal {
			return true
		}
	}

	return false
}
