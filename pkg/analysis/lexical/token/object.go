package token

import (
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type objectPath struct {
	path       []string
	loc        jpos.Range
	body       ast.Node
	requiredID *ast.Identifier
}

func pathToLocation(o *ast.DesugaredObject, pos jpos.Position) (objectPath, error) {
	// check if position is over field name.
	name, r, err := fieldNameAt(o, pos)
	if err == nil {
		return objectPath{
			path: []string{name},
			loc:  r,
		}, nil
	}

	// check if position is over field body.
	for _, field := range o.Fields {
		fn, ok := findFieldFunction(field)
		if ok {
			name, err := fieldName(field)
			if err != nil {
				return objectPath{}, err
			}

			start, ok := o.FieldLocs[name]
			if !ok {
				return objectPath{}, errors.Errorf("unable to find location for field %s", name)
			}

			end := fn.Body.Loc()
			if end == nil {
				return objectPath{}, errors.Errorf("body has no location for field %s", name)
			}

			jfr := ast.LocationRange{
				FileName: end.FileName,
				Begin:    start.Begin,
				End:      end.End,
			}
			if !pos.IsInJsonnetRange(jfr) {
				continue
			}

			// check required parameters
			for _, id := range fn.Parameters.Required {
				paramLoc, ok := fn.Parameters.RequiredLocs[id]
				if !ok {
					return objectPath{}, errors.Errorf("unable to find location for %s", id)
				}

				if !pos.IsInJsonnetRange(paramLoc) {
					continue
				}

				op := objectPath{
					path:       []string{name},
					loc:        jpos.FromJsonnetRange(paramLoc),
					body:       field.Body,
					requiredID: &id,
				}

				return op, nil
			}
		}

		bodyLoc := field.Body.Loc()
		if bodyLoc == nil || !pos.IsInJsonnetRange(*bodyLoc) {
			continue
		}

		var name string
		switch n := field.Name.(type) {
		case *ast.LiteralString:
			name = n.Value
		default:
			continue
		}

		// field body should be a local to contain scope
		local, ok := field.Body.(*ast.Local)
		if !ok {
			return objectPath{}, errors.New("expected scope to be defined in field body")
		}

		switch n := local.Body.(type) {
		case *ast.DesugaredObject:
			// if body is an object look in there
			op, err := pathToLocation(n, pos)
			if err != nil {
				return objectPath{}, err
			}

			op.path = append([]string{name}, op.path...)
			return op, nil
		default:
			r, err := fieldNameLoc(o, name)
			if err != nil {
				return objectPath{}, err
			}

			// return the path
			op := objectPath{
				path: []string{name},
				loc:  r,
				body: field.Body,
			}
			return op, nil
		}
	}

	return objectPath{}, errors.Errorf("position %s could not be identified", pos.String())
}

func fieldName(field ast.DesugaredObjectField) (string, error) {
	switch name := field.Name.(type) {
	case *ast.LiteralString:
		return name.Value, nil
	default:
		return "", errors.New("field name is not a string")
	}
}

func fieldNameAt(o *ast.DesugaredObject, pos jpos.Position) (string, jpos.Range, error) {
	for k, loc := range o.FieldLocs {
		if pos.IsInJsonnetRange(loc) {
			switch k := k.(type) {
			case string:
				return k, jpos.FromJsonnetRange(loc), nil
			case *ast.Var:
				return "", jpos.Range{}, errors.New("variable keys are unsupported")
			default:
				return "", jpos.Range{}, errors.Errorf("invalid field name type %T", k)
			}
		}
	}

	return "", jpos.Range{}, errors.Errorf("position %s isn't in an object key", pos.String())
}

func fieldNameLoc(o *ast.DesugaredObject, name string) (jpos.Range, error) {
	for k, loc := range o.FieldLocs {
		switch k := k.(type) {
		case string:
			if name == k {
				return jpos.FromJsonnetRange(loc), nil
			}
		default:
			continue
		}
	}

	return jpos.Range{}, errors.Errorf("field %q was not found", name)
}

func findFieldFunction(field ast.DesugaredObjectField) (*ast.Function, bool) {
	local, ok := field.Body.(*ast.Local)
	if ok {
		fn, ok := local.Body.(*ast.Function)
		return fn, ok
	}

	return nil, false
}
