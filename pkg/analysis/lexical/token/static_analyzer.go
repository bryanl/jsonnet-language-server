/*
Copyright 2016 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package token

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type staticAnalyzer struct {
	err error
}

type analysisState struct {
	err      error
	freeVars ast.IdentifierSet
}

func (sa *staticAnalyzer) visit(a ast.Node, inObject bool, vars ast.IdentifierSet) {
	if sa.err != nil {
		return
	}

	switch a := a.(type) {
	case *ast.Apply:
		sa.visit(a.Target, inObject, vars)
		for _, arg := range a.Arguments.Positional {
			sa.visit(arg, inObject, vars)
		}
		for _, arg := range a.Arguments.Named {
			sa.visit(arg.Arg, inObject, vars)
		}
	case *ast.Array:
		for _, elem := range a.Elements {
			sa.visit(elem, inObject, vars)
		}
	case *ast.Binary:
		sa.visit(a.Left, inObject, vars)
		sa.visit(a.Right, inObject, vars)
	case *ast.Conditional:
		sa.visit(a.Cond, inObject, vars)
		sa.visit(a.BranchTrue, inObject, vars)
		sa.visit(a.BranchFalse, inObject, vars)
	case *ast.Error:
		sa.visit(a.Expr, inObject, vars)
	case *ast.Function:
		newVars := vars.Clone()
		for _, param := range a.Parameters.Required {
			newVars.Add(param)
		}
		for _, param := range a.Parameters.Optional {
			newVars.Add(param.Name)
		}
		for _, param := range a.Parameters.Optional {
			sa.visit(param.DefaultArg, inObject, newVars)
		}
		sa.visit(a.Body, inObject, newVars)
	case *ast.Import:
		//nothing to do here
	case *ast.ImportStr:
		//nothing to do here
	case *ast.InSuper:
		if !inObject {
			sa.err = locError(errors.Errorf("can't use super outside of an object"), *a.Loc())
			return
		}
		sa.visit(a.Index, inObject, vars)
	case *ast.SuperIndex:
		if !inObject {
			sa.err = locError(errors.Errorf("Can't use super outside of an object"), *a.Loc())
			return
		}
		sa.visit(a.Index, inObject, vars)
	case *ast.Index:
		sa.visit(a.Target, inObject, vars)
		sa.visit(a.Index, inObject, vars)
	case *ast.Local:
		newVars := vars.Clone()
		for _, bind := range a.Binds {
			newVars.Add(bind.Variable)
		}
		// Binds in local can be mutually or even self recursive
		for _, bind := range a.Binds {
			bindVars := newVars.Clone()
			if bind.Fun != nil {
				fun := bind.Fun
				for _, param := range fun.Parameters.Required {
					bindVars.Add(param)
				}
				for _, param := range fun.Parameters.Optional {
					bindVars.Add(param.Name)
				}
				for _, param := range fun.Parameters.Optional {
					sa.visit(param.DefaultArg, inObject, bindVars)
				}
				sa.visit(a.Body, inObject, bindVars)
			}
			sa.visit(bind.Body, inObject, bindVars)
		}

		sa.visit(a.Body, inObject, newVars)
	case *ast.LiteralBoolean:
		//nothing to do here
	case *ast.LiteralNull:
		//nothing to do here
	case *ast.LiteralNumber:
		//nothing to do here
	case *ast.LiteralString:
		//nothing to do here
	case *ast.DesugaredObject:
		for _, field := range a.Fields {
			// Field names are calculated *outside* of the object
			sa.visit(field.Name, inObject, vars)
			sa.visit(field.Body, true, vars)
		}
		for _, assert := range a.Asserts {
			sa.visit(assert, true, vars)
		}
	case *ast.Object:
		newVars := vars.Clone()

		for _, field := range a.Fields {
			fieldVars := newVars.Clone()
			switch field.Kind {
			case ast.ObjectFieldID:
				newVars.Add(*field.Id)
			case ast.ObjectFieldExpr, ast.ObjectFieldStr:
				sa.visit(field.Expr1, inObject, newVars)
			}

			if field.Method != nil {
				method := field.Method
				for _, param := range method.Parameters.Required {
					fieldVars.Add(param)
				}
				for _, param := range method.Parameters.Optional {
					fieldVars.Add(param.Name)
				}
				for _, param := range method.Parameters.Optional {
					sa.visit(param.DefaultArg, inObject, fieldVars)
				}
			}

			if field.Expr2 != nil {
				sa.visit(field.Expr2, true, fieldVars)
			}

			if field.Expr3 != nil {
				sa.visit(field.Expr3, true, fieldVars)
			}
		}
	case *ast.Self:
		if !inObject {
			sa.err = locError(errors.New("can't use self outside of an object"), *a.Loc())
			return
		}
	case *ast.Unary:
		sa.visit(a.Expr, inObject, vars)
	case *ast.Var:
		if !vars.Contains(a.Id) {
			sa.err = locError(errors.Errorf("unknown variable: %v", a.Id), *a.Loc())
			return
		}

		vars.Add(a.Id)
	case *partial:
		//nothing to do here
	case nil:
		return
	default:
		panic(fmt.Sprintf("Unexpected node %T", a))
	}

	a.SetFreeVariables(vars.ToOrderedSlice())
}

func analyze(node ast.Node) error {
	sa := &staticAnalyzer{}
	sa.visit(node, false, ast.NewIdentifierSet("std"))
	return sa.err
}
