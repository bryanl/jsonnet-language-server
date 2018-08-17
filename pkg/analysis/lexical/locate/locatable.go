package locate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
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

	logrus.Debugf("resolving %T", l.Token)

	switch t := l.Token.(type) {
	case *ast.Var:
		return l.handleVar(t)
	case *ast.Index:
		return l.handleIndex(t)
	case ast.Identifier:
		return l.handleDefault()
	case *ast.Identifier:
		return l.handleDefault()
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

func (l *Locatable) handleIndex(i *ast.Index) (*Resolved, error) {
	var indices []string
	var cur ast.Node = i
	for {
		switch t := cur.(type) {
		case *ast.Index:
			cur = t.Target

			if t.Id == nil {
				return nil, errors.New("index didn't have an id")
			}
			indices = append([]string{string(*t.Id)}, indices...)
		case *ast.Var:
			varID := string(t.Id)
			if x, ok := l.Env[varID]; ok {
				logrus.Debugf("it points to a %T", x.Token)

				description, err := describe(x.Token, indices)
				if err != nil {
					return nil, err
				}

				result := &Resolved{
					Location:    x.Loc,
					Token:       l.Token,
					Description: description,
				}

				return result, nil
			}

			return nil, errors.Errorf("could not find %s in env", varID)
		default:
			return nil, errors.Errorf("unable to handle index target of type %T", t)
		}
	}
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
	case *ast.DesugaredObject:
		name = "object"
	case *ast.Function:
		name = "function"
	case *ast.Object:
		return astext.ObjectDescription(t)
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
		logrus.Debugf("%s points to a %T", t.Id, ref.Token)
		s, err := resolvedIdentifier(ref.Token)
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

func resolvedIdentifier(item interface{}) (string, error) {
	switch t := item.(type) {
	case *ast.Import:
		return importDescription(t)
	case *ast.Object:
		return astext.ObjectDescription(t)
	default:
		logrus.Debugf("resolvedIdentifer: did not match %T", t)
		return astext.TokenName(item), nil
	}
}

func importDescription(i *ast.Import) (string, error) {
	// switch t := token.(type) {
	// case *ast.Import:
	// TODO this needs to come from somewhere else
	jPaths := []string{
		"/Users/bryan/go/src/github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/testdata/lexical",
	}

	// 	logrus.Infof("replacing import with its node from %q", t.File.Value)
	// 	node, err := importSource(jPaths, t.File.Value)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	item = node

	// }

	node, err := importSource(jPaths, i.File.Value)
	if err != nil {
		return "", err
	}

	return resolvedIdentifier(node)

}

func importSource(paths []string, name string) (ast.Node, error) {
	for _, jPath := range paths {
		sourcePath := filepath.Join(jPath, name)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		/* #nosec */
		source, err := ioutil.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}

		return parse(sourcePath, string(source))
	}

	return nil, errors.Errorf("unable to find import %q", name)
}

func (l *Locatable) IsFunctionParam() bool {
	if _, isVar := l.Token.(*ast.Var); isVar {
		if _, isParentLocal := l.Parent.Token.(*ast.Local); isParentLocal {
			return true
		}
	}

	return false
}

func describe(item interface{}, indicies []string) (string, error) {
	switch t := item.(type) {
	case *ast.Object:
		return describeInObject(t, indicies)
	default:
		return astext.TokenName(t), nil
	}
}

func describeInObject(o *ast.Object, indicies []string) (string, error) {
	if len(indicies) == 0 {
		return astext.ObjectDescription(o)
	}

	for i := range o.Fields {
		f := o.Fields[i]
		if astext.ObjectFieldName(f) != indicies[0] {
			continue
		}

		return describe(f.Expr2, indicies[1:])
	}

	spew.Dump(indicies, o)
	return "", errors.Errorf("unable to find field %q n object", indicies[0])
}

func parse(filename, snippet string) (ast.Node, error) {
	tokens, err := parser.Lex(filename, snippet)
	if err != nil {
		return nil, err
	}
	node, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}

	return node, nil
}
