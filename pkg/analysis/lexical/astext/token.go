package astext

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// tokenName returns a name for a token.
// nolint: gocyclo
func TokenName(token interface{}) (string, error) {
	switch t := token.(type) {
	case *ast.Apply:
		return "apply", nil
	case *ast.Array:
		return "array", nil
	case *ast.Binary:
		return "binary", nil
	case *ast.Conditional:
		return "conditional", nil
	case *ast.DesugaredObject:
		return "object", nil
	case ast.DesugaredObjectField:
		name, err := TokenValue(t.Name)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(field) %s", name), nil
	case *ast.Function:
		return fmt.Sprintf("function"), nil
	case *ast.LiteralBoolean:
		return "bool", nil
	case *ast.LiteralNull:
		return "null", nil
	case *ast.LiteralNumber:
		return "number", nil
	case *ast.LiteralString:
		return "string", nil
	case ast.Identifier:
		return fmt.Sprintf("identifier %q", string(t)), nil
	case *ast.Import:
		return fmt.Sprintf("import %q", t.File.Value), nil
	case *ast.Index:
		return fmt.Sprintf("index"), nil
	case *ast.Local:
		return "local", nil
	case ast.LocalBind:
		return fmt.Sprintf("local bind %q", string(t.Variable)), nil
	case ast.NamedParameter:
		val, err := TokenValue(t.DefaultArg)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("optional parameter %s=%s", string(t.Name), val), nil
	case *ast.Self:
		return "self", nil
	case *ast.SuperIndex:
		return "super index", nil
	case *ast.Var:
		return fmt.Sprintf("var %q", string(t.Id)), nil
	case RequiredParameter:
		return fmt.Sprintf("required parameter %q", string(t.ID)), nil
	default:
		return "", errors.Errorf("don't know how to name %T", t)
	}
}

func TokenValue(token interface{}) (string, error) {
	switch t := token.(type) {
	case *ast.LiteralNumber:
		return t.OriginalString, nil
	case *ast.LiteralString:
		return t.Value, nil
	default:
		return "", errors.Errorf("unable to get value from %T", t)
	}
}
