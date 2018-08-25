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

package static

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

type nodeScope struct {
	store map[ast.Identifier]ast.Node
}

func newNodeScope() *nodeScope {
	return &nodeScope{
		store: make(map[ast.Identifier]ast.Node),
	}
}

func (ns *nodeScope) Add(i ast.Identifier, node ast.Node) {
	ns.store[i] = node
}

func (ns *nodeScope) Remove(i ast.Identifier) {
	delete(ns.store, i)
}

func (ns *nodeScope) ToScope() ast.Scope {
	return ns.store
}

type analysisState struct {
	err       error
	freeVars  ast.IdentifierSet
	nodeScope *nodeScope
}

func newAnalysisState() *analysisState {
	return &analysisState{
		freeVars:  ast.NewIdentifierSet(),
		nodeScope: newNodeScope(),
	}
}

func (s *analysisState) Add(node ast.Node) {
	s.freeVars.AddIdentifiers(node.FreeVariables())
	for k, v := range node.Scope() {
		s.nodeScope.Add(k, v)
	}
}

type analysisVars struct {
	ids   ast.IdentifierSet
	scope map[ast.Identifier]ast.Node
}

func newAnalysisVars() *analysisVars {
	return &analysisVars{
		ids: ast.NewIdentifierSet("std"),
		scope: map[ast.Identifier]ast.Node{
			ast.Identifier("std"): nil,
		},
	}
}

func (v *analysisVars) Clone() *analysisVars {
	newVars := newAnalysisVars()
	newVars.ids = v.ids.Clone()

	for k, v := range v.scope {
		newVars.scope[k] = v
	}

	return newVars
}

func (v *analysisVars) Add(id ast.Identifier, node ast.Node) {
	v.ids.Add(id)
	v.scope[id] = node
}

func (v *analysisVars) Contains(id ast.Identifier) bool {
	return v.ids.Contains(id)
}

func visitNext(a ast.Node, inObject bool, vars *analysisVars, state *analysisState) {
	if state.err != nil {
		return
	}
	state.err = analyzeVisit(a, inObject, vars)
	state.Add(a)
}

func analyzeVisit(a ast.Node, inObject bool, vars *analysisVars) error {
	s := newAnalysisState()

	// TODO(sbarzowski) Test somehow that we're visiting all the nodes
	switch a := a.(type) {
	case *ast.Apply:
		visitNext(a.Target, inObject, vars, s)
		for _, arg := range a.Arguments.Positional {
			visitNext(arg, inObject, vars, s)
		}
		for _, arg := range a.Arguments.Named {
			visitNext(arg.Arg, inObject, vars, s)
		}
	case *ast.Array:
		for _, elem := range a.Elements {
			visitNext(elem, inObject, vars, s)
		}
	case *ast.Binary:
		visitNext(a.Left, inObject, vars, s)
		visitNext(a.Right, inObject, vars, s)
	case *ast.Conditional:
		visitNext(a.Cond, inObject, vars, s)
		visitNext(a.BranchTrue, inObject, vars, s)
		visitNext(a.BranchFalse, inObject, vars, s)
	case *ast.Error:
		visitNext(a.Expr, inObject, vars, s)
	case *ast.Function:
		newVars := vars.Clone()
		for _, param := range a.Parameters.Required {
			newVars.Add(param, nil)
		}
		for _, param := range a.Parameters.Optional {
			newVars.Add(param.Name, nil)
		}
		for _, param := range a.Parameters.Optional {
			visitNext(param.DefaultArg, inObject, newVars, s)
		}
		visitNext(a.Body, inObject, newVars, s)
		// Parameters are free inside the body, but not visible here or outside
		for _, param := range a.Parameters.Required {
			s.freeVars.Remove(param)
			s.nodeScope.Remove(param)
		}
		for _, param := range a.Parameters.Optional {
			s.freeVars.Remove(param.Name)
			s.nodeScope.Remove(param.Name)
		}
	case *ast.Import:
		//nothing to do here
	case *ast.ImportStr:
		//nothing to do here
	case *ast.InSuper:
		if !inObject {
			return parser.MakeStaticError("Can't use super outside of an object.", *a.Loc())
		}
		visitNext(a.Index, inObject, vars, s)
	case *ast.SuperIndex:
		if !inObject {
			return parser.MakeStaticError("Can't use super outside of an object.", *a.Loc())
		}
		visitNext(a.Index, inObject, vars, s)
	case *ast.Index:
		visitNext(a.Target, inObject, vars, s)
		visitNext(a.Index, inObject, vars, s)
	case *ast.Local:
		newVars := vars.Clone()
		for _, bind := range a.Binds {
			newVars.Add(bind.Variable, a)
		}
		// Binds in local can be mutually or even self recursive
		for _, bind := range a.Binds {
			visitNext(bind.Body, inObject, newVars, s)
		}
		visitNext(a.Body, inObject, newVars, s)

		// Any usage of newly created variables inside are considered free
		// but they are not here or outside
		for _, bind := range a.Binds {
			s.freeVars.Remove(bind.Variable)
			s.nodeScope.Remove(bind.Variable)
		}
	case *ast.LiteralBoolean:
		//nothing to do here
	case *ast.LiteralNull:
		//nothing to do here
	case *ast.LiteralNumber:
		//nothing to do here
	case *ast.LiteralString:
		//nothing to do here
	case *ast.Object:
		for _, field := range a.Fields {
			switch field.Kind {
			case ast.ObjectFieldID:
				vars.Add(*field.Id, field.Expr2)
			case ast.ObjectFieldExpr, ast.ObjectFieldStr:
				visitNext(field.Expr1, inObject, vars, s)
			}

			if field.Method != nil {
				visitNext(field.Method, true, vars, s)
			}

			visitNext(field.Expr2, true, vars, s)
		}
	case *ast.DesugaredObject:
		for _, field := range a.Fields {
			// Field names are calculated *outside* of the object
			visitNext(field.Name, inObject, vars, s)
			visitNext(field.Body, true, vars, s)
		}
		for _, assert := range a.Asserts {
			visitNext(assert, true, vars, s)
		}
	case *astext.Partial:
		// Nothing to do here
	case *ast.Self:
		if !inObject {
			return parser.MakeStaticError("Can't use self outside of an object.", *a.Loc())
		}
	case *ast.Unary:
		visitNext(a.Expr, inObject, vars, s)
	case *ast.Var:
		if !vars.Contains(a.Id) {
			return parser.MakeStaticError(fmt.Sprintf("Unknown variable: %v", a.Id), *a.Loc())
		}
		s.freeVars.Add(a.Id)
		s.nodeScope.Add(a.Id, a)
	default:
		panic(fmt.Sprintf("Unexpected node %T", a))
	}
	a.SetFreeVariables(s.freeVars.ToOrderedSlice())
	a.SetScope(s.nodeScope.ToScope())
	return s.err
}

func Analyze(node ast.Node) error {
	return analyzeVisit(node, false, newAnalysisVars())
}
