package locate

import (
	"bytes"
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrUnresolvable means the loctable can't be resolved.
	ErrUnresolvable = errors.New("unresolvable")
)

// Env is a map of options.
type Env map[string]Locatable

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
	if l == nil {
		return nil, errors.Errorf("locatable is nil")
	}

	switch t := l.Token.(type) {
	case *ast.Var:
		return l.handleVar(t)
	case *ast.Index:
		return l.handleIndex(t)
	case ast.Identifier:
		return l.handleDefault()
	case *ast.Identifier:
		return l.handleDefault()
	case *ast.Function:
		return l.handleFunction(t)
	case ast.NamedParameter:
		return l.handleNamedParameter(t)
	case astext.RequiredParameter:
		return l.handleRequiredParameter(t)
	default:
		logrus.Errorf("unable to resolve %T", l.Token)
		return nil, ErrUnresolvable
	}
}

func (l *Locatable) handleIndex(i *ast.Index) (*Resolved, error) {
	description := fmt.Sprintf("(index) %s", string(*i.Id))

	logrus.Debugf("index points to a %T at %s", i.Target, i.Target.Loc().String())

	result := &Resolved{
		Location:    *i.Target.Loc(),
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

func (l *Locatable) handleDefault() (*Resolved, error) {
	var name string
	var err error

	switch t := l.Parent.Token.(type) {
	case ast.LocalBind:
		logrus.Debug("bind output")
		name, err = bindOutput(t)
	default:
		logrus.Debug("default output")
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

func bindOutput(bind ast.LocalBind) (string, error) {
	var name string

	switch t := bind.Body.(type) {
	case *ast.LiteralString:
		name = "string"
	case *ast.DesugaredObject, *ast.Object:
		name = "object"
	case *ast.Function:
		name = "function"
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

func (l *Locatable) handleVar(t *ast.Var) (*Resolved, error) {
	if ref, ok := l.Env[string(t.Id)]; ok {
		s := l.resolvedIdentifier(&ref)
		resolved := &Resolved{
			Location:    ref.Loc,
			Token:       ref.Token,
			Description: s,
		}

		return resolved, nil
	}

	return nil, ErrUnresolvable
}

func (l *Locatable) resolvedIdentifier(ref *Locatable) string {
	id, ok := ref.Token.(ast.Identifier)
	if !ok {
		return astext.TokenName(ref.Token)
	}

	switch t := ref.Parent.Token.(type) {
	case ast.LocalBind:
		name := astext.TokenName(t.Body)
		return fmt.Sprintf("(%s) %s", name, string(id))
	case *ast.Index:
		name := astext.TokenName(t.Target)
		return fmt.Sprintf("(%s) %s", name, string(id))
	default:
		return astext.TokenName(ref.Token)
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
