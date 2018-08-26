package astext

import (
	"bytes"
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

// Item is something that can identified.
type Item struct {
	token interface{}
}

var _ fmt.Stringer = (*Item)(nil)

// NewItem creates an instance of Item.
func NewItem(token interface{}) *Item {
	return &Item{
		token: token,
	}
}

func (i *Item) String() string {
	return TokenName(i.token)
}

// TokenName returns a name for a token.
// nolint: gocyclo
func TokenName(token interface{}) string {
	switch t := token.(type) {
	case *ast.Apply:
		return "(apply)"
	case *ast.Array:
		return "(array)"
	case *ast.ArrayComp:
		return "(arraycomp)"
	case *ast.Binary:
		return "(binary)"
	case *ast.Conditional:
		return "(conditional)"
	case *ast.DesugaredObject:
		return desugaredObject(t)
	case ast.DesugaredObjectField:
		name := TokenValue(t.Name)
		return fmt.Sprintf("(field) %s", name)
	case ast.ForSpec:
		return "forspec"
	case *ast.Function:
		return fmt.Sprintf("(function)")
	case *ast.LiteralBoolean:
		return "(bool)"
	case *ast.LiteralNull:
		return "(null)"
	case *ast.LiteralNumber:
		return fmt.Sprintf("(number) %s", t.OriginalString)
	case *ast.LiteralString:
		return fmt.Sprintf("(string) %s", TokenValue(t))
	case ast.Identifier:
		return fmt.Sprintf("(identifier) %s", string(t))
	case *ast.Identifier:
		return fmt.Sprintf("(identifier) %s", string(*t))
	case *ast.Import:
		return fmt.Sprintf("(import) %s", t.File.Value)
	case *ast.ImportStr:
		return fmt.Sprintf("(importstr) %s", t.File.Value)
	case *ast.Index:
		if t.Id == nil {
			return fmt.Sprintf("(array index) [%s]", TokenValue(t.Index))
		}
		return fmt.Sprintf("(index) %s", string(*t.Id))
	case *ast.Local:
		return "(local)"
	case ast.LocalBind:
		return fmt.Sprintf("(local bind) %s", string(t.Variable))
	case ast.NamedParameter:
		val := TokenValue(t.DefaultArg)
		return fmt.Sprintf("(optional parameter) %s=%s", string(t.Name), val)
	case *ast.Object:
		return ObjectDescription(t)
	case ast.ObjectField:
		return fmt.Sprintf("(field) %s", ObjectFieldName(t))
	case *ast.Self:
		return "(self)"
	case *ast.SuperIndex:
		return fmt.Sprintf("(super index) %s", string(*t.Id))
	case *ast.Var:
		return fmt.Sprintf("(var) %s", string(t.Id))
	case RequiredParameter:
		return fmt.Sprintf("(required parameter) %s", string(t.ID))
	default:
		return fmt.Sprintf("(unknown) %T", t)
	}
}

func TokenValue(token interface{}) string {
	switch t := token.(type) {
	case *ast.LiteralNumber:
		return t.OriginalString
	case *ast.LiteralString:
		return stringValue(t, true)
	default:
		return fmt.Sprintf("unknown value from %T", t)
	}
}

func stringValue(t *ast.LiteralString, quote bool) string {
	if !quote {
		return t.Value
	}

	switch t.Kind {
	case ast.StringDouble:
		return fmt.Sprintf(`"%s"`, t.Value)
	case ast.StringSingle:
		return fmt.Sprintf(`'%s'`, t.Value)
	case ast.StringBlock:
		return "<block string>"
	default:
		return t.Value
	}
}

func ObjectFieldName(f ast.ObjectField) string {
	if f.Id != nil {
		return string(*f.Id)
	}

	if f.Expr1 != nil {
		return TokenValue(f.Expr1)
	}

	panic("object field does not have a name")
}

func ObjectFieldVisibility(f ast.ObjectFieldHide) string {
	switch f {
	case ast.ObjectFieldHidden:
		return "::"
	case ast.ObjectFieldInherit:
		return ":"
	case ast.ObjectFieldVisible:
		return ":::"
	default:
		return ":"
	}
}

const (
	genericObject = "(object)"
)

func ObjectDescription(o *ast.Object) string {
	if o == nil {
		return genericObject
	}

	var buf bytes.Buffer
	if _, err := buf.WriteString("(object) {"); err != nil {
		return genericObject
	}

	// find object fields
	for i, field := range o.Fields {
		if i == 0 {
			if _, err := buf.WriteString("\n"); err != nil {
				return genericObject
			}
		}
		fieldName := ObjectFieldName(field)
		visibility := ObjectFieldVisibility(field.Hide)
		label := "field"
		if field.Params != nil {
			label = "function"
		}
		if _, err := buf.WriteString(fmt.Sprintf("  (%s) %s%s,\n", label, fieldName, visibility)); err != nil {
			return genericObject
		}
	}
	if _, err := buf.WriteString("}"); err != nil {
		return genericObject
	}

	return buf.String()
}

func desugaredObject(o *ast.DesugaredObject) string {
	if o == nil {
		return genericObject
	}

	var buf bytes.Buffer
	if _, err := buf.WriteString("(object) {"); err != nil {
		return genericObject
	}

	// find object fields
	for i, field := range o.Fields {
		if i == 0 {
			if _, err := buf.WriteString("\n"); err != nil {
				return genericObject
			}
		}

		name, ok := field.Name.(*ast.LiteralString)
		if !ok {
			continue
		}

		fieldName := stringValue(name, false)
		visibility := ObjectFieldVisibility(field.Hide)

		// local, ok := field.Body.(*ast.Local)
		// if !ok {
		// 	return genericObject
		// }

		// local

		label := "field"
		// if field.Params != nil {
		// 	label = "function"
		// }
		if _, err := buf.WriteString(fmt.Sprintf("  (%s) %s%s,\n", label, fieldName, visibility)); err != nil {
			return genericObject
		}
	}
	if _, err := buf.WriteString("}"); err != nil {
		return genericObject
	}

	return buf.String()
}
