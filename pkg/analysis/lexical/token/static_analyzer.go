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

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type analyzerMap struct {
	store map[ast.Node]analyzerScope
}

func newAnalyzerMap() analyzerMap {
	return analyzerMap{
		store: make(map[ast.Node]analyzerScope),
	}
}

func (am *analyzerMap) add(node ast.Node, id string, idNode ast.Node) {
	as, ok := am.store[node]
	if !ok {
		as = newAnalyzerScope()
		am.store[node] = as
	}

	as.add(id, idNode)
}

type analyzerScope struct {
	store map[string]ast.Node
}

func newAnalyzerScope() analyzerScope {
	return analyzerScope{
		store: make(map[string]ast.Node),
	}
}

func (as *analyzerScope) add(id string, idNode ast.Node) {
	as.store[id] = idNode
}

type staticAnalyzer struct {
	err           error
	analyzerMap   analyzerMap
	enclosingNode ast.Node
	loc           ast.Location
	scope         *scopeCatalog
}

type analysisState struct {
	err error
}

// nolint: gocyclo
func (sa *staticAnalyzer) visit(a, parent ast.Node, inObject bool, vars *scopeCatalog) {
	// func (sa *staticAnalyzer) visit(a ast.Node, inObject bool, vars ast.IdentifierSet) {
	if sa.err != nil {
		return
	}

	switch a := a.(type) {
	case *ast.Apply:
		sa.visit(a.Target, a, inObject, vars)
		for _, arg := range a.Arguments.Positional {
			sa.visit(arg, a, inObject, vars)
		}
		for _, arg := range a.Arguments.Named {
			sa.visit(arg.Arg, a, inObject, vars)
		}
	case *ast.Array:
		for _, elem := range a.Elements {
			sa.visit(elem, a, inObject, vars)
		}
	case *ast.Binary:
		sa.visit(a.Left, a, inObject, vars)
		sa.visit(a.Right, a, inObject, vars)
	case *ast.Conditional:
		sa.visit(a.Cond, a, inObject, vars)
		sa.visit(a.BranchTrue, a, inObject, vars)
		sa.visit(a.BranchFalse, a, inObject, vars)
	case *ast.Error:
		sa.visit(a.Expr, a, inObject, vars)
	case *ast.Function:
		newVars := vars.Clone(a)
		for _, param := range a.Parameters.Required {
			newVars.Add(param, a)
		}
		for _, param := range a.Parameters.Optional {
			newVars.Add(param.Name, a)
		}
		for _, param := range a.Parameters.Optional {
			sa.visit(param.DefaultArg, a, inObject, newVars)
		}
		sa.visit(a.Body, a, inObject, newVars)
		vars = newVars
	case *ast.Import:
		//nothing to do here
	case *ast.ImportStr:
		//nothing to do here
	case *ast.InSuper:
		if !inObject {
			sa.err = locError(errors.Errorf("can't use super outside of an object"), *a.Loc())
			return
		}
		sa.visit(a.Index, a, inObject, vars)
	case *ast.SuperIndex:
		if !inObject {
			sa.err = locError(errors.Errorf("Can't use super outside of an object"), *a.Loc())
			return
		}
		sa.visit(a.Index, a, inObject, vars)
	case *ast.Index:
		sa.visit(a.Target, a, inObject, vars)
		sa.visit(a.Index, a, inObject, vars)
	case *ast.Local:
		for _, bind := range a.Binds {
			vars.Add(bind.Variable, bind.Body)
			// TODO track body
		}
		// Binds in local can be mutually or even self recursive
		for _, bind := range a.Binds {
			bindVars := vars.Clone(a)
			if bind.Fun != nil {
				sa.visit(bind.Fun, a, inObject, bindVars)
			} else {
				sa.visit(bind.Body, a, inObject, bindVars)
			}
		}

		sa.visit(a.Body, a, inObject, vars)
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
			sa.visit(field.Name, a, inObject, vars)
			sa.visit(field.Body, a, true, vars)
		}
		for _, assert := range a.Asserts {
			sa.visit(assert, a, true, vars)
		}
	case *ast.Object:
		newVars := vars.Clone(a)

		for _, field := range a.Fields {
			switch field.Kind {
			case ast.ObjectFieldID:
				newVars.Add(*field.Id, field.Expr2)
			case ast.ObjectFieldExpr, ast.ObjectFieldStr:
				sa.visit(field.Expr1, field.Expr2, inObject, newVars)
			}

			if field.Method != nil {
				method := field.Method
				for _, param := range method.Parameters.Required {
					newVars.Add(param, field.Method)
				}
				for _, param := range method.Parameters.Optional {
					newVars.Add(param.Name, field.Method)
				}
				for _, param := range method.Parameters.Optional {
					sa.visit(param.DefaultArg, field.Method, inObject, newVars)
				}
			}

			if field.Expr2 != nil {
				sa.visit(field.Expr2, a, true, newVars)
			}

			if field.Expr3 != nil {
				sa.visit(field.Expr3, a, true, newVars)
			}
		}
	case *ast.Self:
		if !inObject {
			sa.err = locError(errors.New("can't use self outside of an object"), *a.Loc())
			return
		}
	case *ast.Unary:
		sa.visit(a.Expr, a, inObject, vars)
	case *ast.Var:
		if !vars.Contains(a.Id) {
			fmt.Printf("vars: %#v\n", vars.ids)
			sa.err = locError(errors.Errorf("unknown variable: %v", a.Id), *a.Loc())
			return
		}

		vars.Add(a.Id, parent)
	case *astext.Partial:
		//nothing to do here
	case nil:
		return
	default:
		panic(fmt.Sprintf("Unexpected node %T", a))
	}

	a.SetFreeVariables(vars.FreeVariables())

	if inRange(sa.loc, *a.Loc()) {
		if sa.enclosingNode == nil {
			fmt.Printf("setting initial node\n")
			sa.enclosingNode = a
			sa.scope = vars
		} else if isRangeSmaller(*sa.enclosingNode.Loc(), *a.Loc()) {
			fmt.Printf("setting %T as node because %s is smaller than %s\n",
				a, a.Loc().String(), sa.enclosingNode.Loc().String())
			sa.enclosingNode = a
			sa.scope = vars
		} else {
			fmt.Printf("did %s in %T match %s\n",
				a.Loc().String(), a, sa.enclosingNode.Loc().String())
		}
	}
}

func analyze(node ast.Node, loc ast.Location) (*scopeCatalog, error) {
	sc := newScopeCatalog("std")
	sa := &staticAnalyzer{
		analyzerMap: newAnalyzerMap(),
		scope:       sc,
		loc:         loc,
	}
	sa.visit(node, nil, false, sc)
	if sa.err != nil {
		return nil, sa.err
	}

	return sa.scope, nil
}
