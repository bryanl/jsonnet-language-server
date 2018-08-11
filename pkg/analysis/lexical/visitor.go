package lexical

import (
	"fmt"
	"reflect"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// PreVisit visits a token.
type PreVisit func(token, parent interface{}, env Env) error

// Env is a map of options.
type Env map[string]interface{}

// Visitor visits.
type Visitor interface {
	Visit() error
}

// NodeVisitor visits a node and its children.
type NodeVisitor struct {
	Node   ast.Node
	Parent ast.Node
	Env    Env

	PreVisit PreVisit

	*ApplyVisitor
	*ApplyBraceVisitor
	*ArrayVisitor
	*ArrayCompVisitor
	*AssertVisitor
	*BinaryVisitor
	*ConditionalVisitor
	*DesugaredObjectFieldVisitor
	*DesugaredObjectVisitor
	*DollarVisitor
	*ErrorVisitor
	*FunctionVisitor
	*ImportVisitor
	*ImportStrVisitor
	*IndexVisitor
	*LiteralBooleanVisitor
	*LiteralNullVisitor
	*LiteralNumberVisitor
	*LiteralStringVisitor
	*LocalBindVisitor
	*LocalVisitor
	*ParensVisitor
	*ObjectFieldVisitor
	*ObjectCompVisitor
	*ObjectVisitor
	*SelfVisitor
	*SliceVisitor
	*SuperIndexVisitor
	*VarVisitor
}

// NewNodeVisitor creates an instance of Visitor.
func NewNodeVisitor(node, parent ast.Node, env Env, pv PreVisit) *NodeVisitor {
	return &NodeVisitor{
		Node:     node,
		Parent:   parent,
		Env:      env,
		PreVisit: pv,
	}
}

// Visit visits a node.
func (v *NodeVisitor) Visit() error {
	return v.visit(v.Node, v.Parent, v.Env)
}

func (v *NodeVisitor) visit(token, parent interface{}, env Env) error {
	if token == nil {
		return nil
	}

	if v.PreVisit != nil {
		if err := v.PreVisit(token, parent, env); err != nil {
			return errors.Wrap(err, "previsit")
		}
	}

	if node, ok := token.(ast.Node); ok {
		return v.handleNode(node, env)
	}

	switch t := token.(type) {
	case ast.LocalBind:
		return v.handleLocalBind(t, env)
	default:
		return errors.Errorf("unable to handle token of type %T", t)
	}
}

// nolint: gocyclo
func (v *NodeVisitor) handleNode(node ast.Node, env Env) error {
	switch t := node.(type) {
	case *ast.Apply:
		return v.handleApply(t, env)
	case *ast.ApplyBrace:
		return v.handleApplyBrace(t, env)
	case *ast.Array:
		return v.handleArray(t, env)
	case *ast.ArrayComp:
		return v.handleArrayComp(t, env)
	case *ast.Binary:
		return v.handleBinary(t, env)
	case *ast.Assert:
		return v.handleAssert(t, env)
	case *ast.Conditional:
		return v.handleConditional(t, env)
	case *ast.DesugaredObject:
		return v.handleDesugaredObject(t, env)
	case *ast.Dollar:
		return v.handleDollar(t, env)
	case *ast.Error:
		return v.handleError(t, env)
	case *ast.Function:
		return v.handleFunction(t, env)
	case *ast.Import:
		return v.handleImport(t, env)
	case *ast.Index:
		return v.handleIndex(t, env)
	case *ast.ImportStr:
		return v.handleImportStr(t, env)
	case *ast.LiteralBoolean:
		return v.handleLiteralBoolean(t)
	case *ast.LiteralNull:
		return v.handleLiteralNull(t)
	case *ast.LiteralNumber:
		return v.handleLiteralNumber(t)
	case *ast.LiteralString:
		return v.handleLiteralString(t)
	case *ast.Local:
		return v.handleLocal(t, env)
	case *ast.Parens:
		return v.handleParens(t, env)
	case *ast.Object:
		return v.handleObject(t, env)
	case *ast.ObjectComp:
		return v.handleObjectComp(t, env)
	case *ast.Self:
		return v.handleSelf(t, env)
	case *ast.Slice:
		return v.handleSlice(t, env)
	case *ast.SuperIndex:
		return v.handleSuperIndex(t, env)
	case *ast.Var:
		return v.handleVar(t, env)
	default:
		return errors.Errorf("unable to handle node type %T", t)
	}

	return nil
}

func (v *NodeVisitor) visitList(list []ast.Node, parent ast.Node, env Env) error {
	for _, node := range list {
		if err := v.visit(node, parent, env); err != nil {
			return errors.Wrapf(err, "visiting %T", node)
		}
	}

	return nil
}

// ApplyVisitor is a visitor for Apply.
type ApplyVisitor struct {
	VisitApply func(a *ast.Apply) error
}

func (v *NodeVisitor) visitTypeIfExists(name string, i interface{}) error {
	fieldName := fmt.Sprintf("Visit%s", name)

	in := reflect.ValueOf(i)
	method := reflect.ValueOf(v).MethodByName(fieldName)
	if reflect.DeepEqual(method, reflect.Value{}) {
		return nil
	}

	results := method.Call([]reflect.Value{in})
	if len(results) != 1 {
		return errors.Errorf("%s returned something unexpected", fieldName)
	}

	err, ok := reflect.ValueOf(results[0]).Interface().(error)
	if !ok {
		return errors.Errorf("%s did not return an error", fieldName)
	}

	return errors.Wrapf(err, "visit %s", name)
}

func (v *NodeVisitor) handleApply(n *ast.Apply, env Env) error {
	if err := v.visitTypeIfExists("Apply", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Target}
	nodes = append(nodes, n.Arguments.Positional...)
	for _, arg := range n.Arguments.Named {
		nodes = append(nodes, arg.Arg)
	}

	return v.visitList(nodes, n, env)
}

// ApplyBraceVisitor is a visitor for ApplyBrace.
type ApplyBraceVisitor struct {
	VisitApplyBrace func(a *ast.ApplyBrace) error
}

func (v *NodeVisitor) handleApplyBrace(n *ast.ApplyBrace, env Env) error {
	if err := v.visitTypeIfExists("ApplyBrace", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Left, n.Right}
	return v.visitList(nodes, n, env)
}

// ArrayVisitor is a visitor for Array.
type ArrayVisitor struct {
	VisitArray func(a *ast.Array) error
}

func (v *NodeVisitor) handleArray(n *ast.Array, env Env) error {
	if err := v.visitTypeIfExists("Array", n); err != nil {
		return err
	}

	return v.visitList(n.Elements, n, env)
}

// ArrayCompVisitor is a visitory for ArrayComp.
type ArrayCompVisitor struct {
	VisitArrayComp func(ac *ast.ArrayComp) error
}

func (v *NodeVisitor) handleArrayComp(n *ast.ArrayComp, env Env) error {
	if err := v.visitTypeIfExists("ArrayComp", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Body, n.Spec.Expr}
	forSpec := n.Spec
	if forSpec.Outer != nil {
		nodes = append(nodes, forSpec.Outer.Expr)
	}

	for _, ifSpec := range forSpec.Conditions {
		nodes = append(nodes, ifSpec.Expr)
	}

	return v.visitList(nodes, n, env)
}

// AssertVisitor is a visitor for Assert.
type AssertVisitor struct {
	VisitAssert func(n *ast.Assert) error
}

func (v *NodeVisitor) handleAssert(n *ast.Assert, env Env) error {
	if err := v.visitTypeIfExists("Assert", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Cond, n.Message, n.Rest}

	return v.visitList(nodes, n, env)
}

// BinaryVisitor is a visitor for Binary.
type BinaryVisitor struct {
	VisitBinary func(n *ast.Binary) error
}

func (v *NodeVisitor) handleBinary(n *ast.Binary, env Env) error {
	if err := v.visitTypeIfExists("Binary", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Left, n.Right}

	return v.visitList(nodes, n, env)
}

// ConditionalVisitor is a visitor for Conditional.
type ConditionalVisitor struct {
	VisitConditional func(n *ast.Conditional) error
}

func (v *NodeVisitor) handleConditional(n *ast.Conditional, env Env) error {
	if err := v.visitTypeIfExists("Conditional", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Cond, n.BranchTrue, n.BranchFalse}

	return v.visitList(nodes, n, env)
}

// DesugaredObjectFieldVisitor is a visitor for DesugaredObjectField.
type DesugaredObjectFieldVisitor struct {
	VisitDesugaredObjectField func(n *ast.DesugaredObjectField) error
}

func (v *NodeVisitor) handleDesugaredObjectField(n *ast.DesugaredObjectField, env Env) error {
	if err := v.visitTypeIfExists("DesugaredObjectField", n); err != nil {
		return err
	}
	nodes := []ast.Node{n.Name, n.Body}

	return v.visitList(nodes, nil, env)
}

// DesugaredObjectVisitor is a visitor for DesugaredObject.
type DesugaredObjectVisitor struct {
	VisitDesugaredObject func(n *ast.DesugaredObject) error
}

func (v *NodeVisitor) handleDesugaredObject(n *ast.DesugaredObject, env Env) error {
	if err := v.visitTypeIfExists("DesugaredObject", n); err != nil {
		return err
	}

	nodes := []ast.Node{}
	nodes = append(nodes, n.Asserts...)

	if err := v.visitList(nodes, n, env); err != nil {
		return err
	}

	for i := range n.Fields {
		field := &n.Fields[i]
		if err := v.visitTypeIfExists("DesugaredObjectField", field); err != nil {
			return errors.Wrap(err, "visit DesugaredObjectField")
		}
	}

	return nil
}

// DollarVisitor is a visitor for Dollar.
type DollarVisitor struct {
	VisitDollar func(n *ast.Dollar) error
}

func (v *NodeVisitor) handleDollar(n *ast.Dollar, env Env) error {
	if err := v.visitTypeIfExists("Dollar", n); err != nil {
		return err
	}

	nodes := []ast.Node{}

	return v.visitList(nodes, n, env)
}

// ErrorVisitor is a visitor for Error.
type ErrorVisitor struct {
	VisitError func(n *ast.Error) error
}

func (v *NodeVisitor) handleError(n *ast.Error, env Env) error {
	if err := v.visitTypeIfExists("Error", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Expr}

	return v.visitList(nodes, n, env)
}

// FunctionVisitor is a visitor for Function.
type FunctionVisitor struct {
	VisitFunction func(n *ast.Function) error
}

func (v *NodeVisitor) handleFunction(n *ast.Function, env Env) error {
	if err := v.visitTypeIfExists("Function", n); err != nil {
		return err
	}

	// TODO create new env from params and visit the Parameters
	// TODO visit n.Body with env params

	return nil
}

// ImportVisitor is a visitor for Import.
type ImportVisitor struct {
	VisitImport func(n *ast.Import) error
}

func (v *NodeVisitor) handleImport(n *ast.Import, env Env) error {
	if err := v.visitTypeIfExists("Import", n); err != nil {
		return err
	}

	return nil
}

// ImportStrVisitor is a visitor for ImportStr.
type ImportStrVisitor struct {
	VisitImportStr func(n *ast.ImportStr) error
}

func (v *NodeVisitor) handleImportStr(n *ast.ImportStr, env Env) error {
	if err := v.visitTypeIfExists("ImportStr", n); err != nil {
		return err
	}

	return nil
}

// IndexVisitor is a visitor for Index.
type IndexVisitor struct {
	VisitIndex func(n *ast.Index) error
}

func (v *NodeVisitor) handleIndex(n *ast.Index, env Env) error {
	if err := v.visitTypeIfExists("Index", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Target, n.Index}

	return v.visitList(nodes, n, env)
}

// LiteralBooleanVisitor is a visitor for LiteralBoolean.
type LiteralBooleanVisitor struct {
	VisitLiteralBoolean func(n *ast.LiteralBoolean) error
}

func (v *NodeVisitor) handleLiteralBoolean(n *ast.LiteralBoolean) error {
	if err := v.visitTypeIfExists("LiteralBoolean", n); err != nil {
		return err
	}

	return nil
}

// LiteralNullVisitor is a visitor for LiteralNull.
type LiteralNullVisitor struct {
	VisitLiteralNull func(n *ast.LiteralNull) error
}

func (v *NodeVisitor) handleLiteralNull(n *ast.LiteralNull) error {
	if err := v.visitTypeIfExists("LiteralNull", n); err != nil {
		return err
	}

	return nil
}

// LiteralNumberVisitor is a visitor for LiteralNumber.
type LiteralNumberVisitor struct {
	VisitLiteralNumber func(n *ast.LiteralNumber) error
}

func (v *NodeVisitor) handleLiteralNumber(n *ast.LiteralNumber) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LiteralStringVisitor is a visitor for LiteralString.
type LiteralStringVisitor struct {
	VisitLiteralString func(n *ast.LiteralString) error
}

func (v *NodeVisitor) handleLiteralString(n *ast.LiteralString) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LocalVisitor is a visitor for Local.
type LocalVisitor struct {
	VisitLocal func(n *ast.Local) error
}

func (v *NodeVisitor) handleLocal(n *ast.Local, env Env) error {
	if err := v.visitTypeIfExists("Local", n); err != nil {
		return err
	}

	// TODO create new envWithBinds by merging tree.envFromLocalBinds(n)
	envWithBinds := env

	for _, bind := range n.Binds {
		if err := v.visit(bind, n, envWithBinds); err != nil {
			return err
		}
	}

	if err := v.visit(n.Body, n, envWithBinds); err != nil {
		return err
	}

	return nil
}

// LocalBindVisitor is a visitor for LocalBind.
type LocalBindVisitor struct {
	VisitLocalBind func(n ast.LocalBind) error
}

func (v *NodeVisitor) handleLocalBind(lb ast.LocalBind, env Env) error {
	// TODO figure out location range for local bind

	if err := v.visitTypeIfExists("LocalBind", lb); err != nil {
		return err
	}

	// TODO merge env with local bind params
	envWithParams := env

	if fun := lb.Fun; fun != nil {
		for _, param := range fun.Parameters.Optional {
			if err := v.visit(param, lb, envWithParams); err != nil {
				return err
			}
		}
		for _, param := range fun.Parameters.Required {
			if err := v.visit(param, lb, envWithParams); err != nil {
				return err
			}
		}
	}

	if err := v.visit(lb.Body, lb, envWithParams); err != nil {
		return err
	}

	return nil
}

// ParensVisitor is a visitor for Parens.
type ParensVisitor struct {
	VisitParens func(n *ast.Parens) error
}

func (v *NodeVisitor) handleParens(n *ast.Parens, env Env) error {
	if err := v.visitTypeIfExists("Parens", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Inner}

	return v.visitList(nodes, n, env)
}

// ObjectCompVisitor is a visitor for ObjectComp.
type ObjectCompVisitor struct {
	VisitObjectComp func(n *ast.ObjectComp) error
}

func (v *NodeVisitor) handleObjectComp(n *ast.ObjectComp, env Env) error {
	if err := v.visitTypeIfExists("ObjectComp", n); err != nil {
		return err
	}

	// TODO how to visit a for spec? should also revisit ArrayComp

	nodes := []ast.Node{}

	return v.visitList(nodes, n, env)
}

// ObjectFieldVisitor is a visitor for ObjectField.
type ObjectFieldVisitor struct {
	VisitObjectField func(n *ast.ObjectField) error
}

func (v *NodeVisitor) handleObjectField(n *ast.ObjectField, env Env) error {
	if err := v.visitTypeIfExists("ObjectField", n); err != nil {
		return err
	}

	// TODO: need env from params here

	// nodes := []ast.Node{}

	// return v.visitList(nodes, n, env)

	return nil
}

// ObjectVisitor is a visitor for Object.
type ObjectVisitor struct {
	VisitObject func(n *ast.Object) error
}

func (v *NodeVisitor) handleObject(n *ast.Object, env Env) error {
	if err := v.visitTypeIfExists("Object", n); err != nil {
		return err
	}

	// TODO this needs help

	nodes := []ast.Node{}

	return v.visitList(nodes, n, env)
}

// SelfVisitor is a visitor for Self.
type SelfVisitor struct {
	VisitSelf func(n *ast.Self) error
}

func (v *NodeVisitor) handleSelf(n *ast.Self, env Env) error {
	if err := v.visitTypeIfExists("Self", n); err != nil {
		return err
	}

	return nil
}

// SliceVisitor is a visitor for Slice.
type SliceVisitor struct {
	VisitSlice func(n *ast.Slice) error
}

func (v *NodeVisitor) handleSlice(n *ast.Slice, env Env) error {
	if err := v.visitTypeIfExists("Slice", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.BeginIndex, n.EndIndex, n.Step}

	return v.visitList(nodes, n, env)
}

// SuperIndexVisitor is a visitor for SuperIndex.
type SuperIndexVisitor struct {
	VisitSuperIndex func(n *ast.SuperIndex) error
}

func (v *NodeVisitor) handleSuperIndex(n *ast.SuperIndex, env Env) error {
	if err := v.visitTypeIfExists("SuperIndex", n); err != nil {
		return err
	}

	nodes := []ast.Node{n.Index}

	return v.visitList(nodes, n, env)
}

// VarVisitor is a visitor for Var.
type VarVisitor struct {
	VisitVar func(n *ast.Var) error
}

func (v *NodeVisitor) handleVar(n *ast.Var, env Env) error {
	if err := v.visitTypeIfExists("Var", n); err != nil {
		return err
	}

	return nil
}
