package lexical

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// VisitFn visits a token.
type VisitFn func(token interface{}, parent *locate.Locatable, scope locate.Scope) error

// Visitor visits.
type Visitor interface {
	Visit() error
}

// NodeVisitor visits a node and its children.
type NodeVisitor struct {
	Node   ast.Node
	Parent ast.Node
	Scope  locate.Scope
	Source []byte

	PreVisit  VisitFn
	PostVisit VisitFn

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
	*ForSpecVisitor
	*FunctionVisitor
	*IdentifierVisitor
	*ImportVisitor
	*ImportStrVisitor
	*IndexVisitor
	*LiteralBooleanVisitor
	*LiteralNullVisitor
	*LiteralNumberVisitor
	*LiteralStringVisitor
	*LocalBindVisitor
	*LocalVisitor
	*NamedParameterVisitor
	*ParensVisitor
	*ObjectFieldVisitor
	*ObjectCompVisitor
	*ObjectVisitor
	*SelfVisitor
	*SliceVisitor
	*SuperIndexVisitor
	*VarVisitor
}

// VisitOpt is an option for NodeVisitor.
type VisitOpt func(*NodeVisitor)

// PreVisit is a previsit option.
func PreVisit(fn VisitFn) VisitOpt {
	return func(v *NodeVisitor) {
		v.PreVisit = fn
	}
}

// PostVisit is a postvisit option.
func PostVisit(fn VisitFn) VisitOpt {
	return func(v *NodeVisitor) {
		v.PostVisit = fn
	}
}

func convertToNode(filename, snippet string) (ast.Node, error) {
	node, err := token.Parse(filename, snippet)
	if err != nil {
		return nil, errors.Wrap(err, "parsing source")
	}

	if err := token.DesugarFile(&node); err != nil {
		return nil, err
	}

	return node, nil
}

// NewNodeVisitor creates an instance of Visitor.
func NewNodeVisitor(filename string, r io.Reader, partial bool, opts ...VisitOpt) (*NodeVisitor, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading source")
	}

	node, err := convertToNode(filename, string(data))
	if err != nil {
		return nil, err
	}

	scope := locate.Scope{}

	v := &NodeVisitor{
		Node:   node,
		Parent: nil,
		Scope:  scope,
		Source: data,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v, nil
}

// Visit visits a node.
func (v *NodeVisitor) Visit() error {
	var parent *locate.Locatable
	if v.Parent != nil {
		parent = &locate.Locatable{
			Token: v.Parent,
			Loc:   *v.Parent.Loc(),
		}
	}

	return v.visit(v.Node, parent, v.Scope)
}
func (v *NodeVisitor) visit(token interface{}, parent *locate.Locatable, scope locate.Scope) error {
	if token == nil {
		return nil
	}

	if v.PreVisit != nil {
		if err := v.PreVisit(token, parent, scope); err != nil {
			return errors.Wrapf(err, "pre visiting %T", token)
		}
	}

	if err := v.visitToken(token, parent, scope); err != nil {
		return err
	}

	if v.PostVisit != nil {
		if err := v.PostVisit(token, parent, scope); err != nil {
			return errors.Wrapf(err, "post visiting %T", token)
		}
	}

	return nil
}

// nolint: gocyclo
func (v *NodeVisitor) visitToken(token interface{}, parent *locate.Locatable, scope locate.Scope) error {

	if node, ok := token.(ast.Node); ok {
		return v.handleNode(node, parent, scope)
	}

	switch t := token.(type) {
	case astext.RequiredParameter:
		return v.handleIdentifier(t.ID, parent, scope)
	case ast.DesugaredObjectField:
		return v.handleDesugaredObjectField(t, parent, scope)
	case ast.ForSpec:
		return v.handleForSpec(t, parent, scope)
	case *ast.Identifier:
		return nil
	case ast.Identifier:
		return v.handleIdentifier(t, parent, scope)
	case ast.LocalBind:
		return v.handleLocalBind(t, parent, scope)
	case ast.NamedParameter:
		return v.handleNamedParameter(t, parent, scope)
	case ast.ObjectField:
		return v.handleObjectField(t, parent, scope)
	default:
		return errors.Errorf("unable to handle token of type %T", t)
	}
}

// nolint: gocyclo
func (v *NodeVisitor) handleNode(node ast.Node, parent *locate.Locatable, scope locate.Scope) error {
	switch t := node.(type) {
	case *ast.Apply:
		return v.handleApply(t, parent, scope)
	case *ast.ApplyBrace:
		return v.handleApplyBrace(t, parent, scope)
	case *ast.Array:
		return v.handleArray(t, parent, scope)
	case *ast.ArrayComp:
		return v.handleArrayComp(t, parent, scope)
	case *ast.Binary:
		return v.handleBinary(t, parent, scope)
	case *ast.Assert:
		return v.handleAssert(t, parent, scope)
	case *ast.Conditional:
		return v.handleConditional(t, parent, scope)
	case *ast.DesugaredObject:
		return v.handleDesugaredObject(t, parent, scope)
	case *ast.Dollar:
		return v.handleDollar(t, parent, scope)
	case *ast.Error:
		return v.handleError(t, parent, scope)
	case *ast.Function:
		return v.handleFunction(t, parent, scope)
	case *ast.Import:
		return v.handleImport(t, parent, scope)
	case *ast.Index:
		return v.handleIndex(t, parent, scope)
	case *ast.ImportStr:
		return v.handleImportStr(t, parent, scope)
	case *ast.LiteralBoolean:
		return v.handleLiteralBoolean(t, parent)
	case *ast.LiteralNull:
		return v.handleLiteralNull(t, parent)
	case *ast.LiteralNumber:
		return v.handleLiteralNumber(t, parent)
	case *ast.LiteralString:
		return v.handleLiteralString(t, parent)
	case *ast.Local:
		return v.handleLocal(t, parent, scope)
	case *ast.Parens:
		return v.handleParens(t, parent, scope)
	case *ast.Object:
		return v.handleObject(t, parent, scope)
	case *ast.ObjectComp:
		return v.handleObjectComp(t, parent, scope)
	case *ast.Self:
		return v.handleSelf(t, parent, scope)
	case *ast.Slice:
		return v.handleSlice(t, parent, scope)
	case *ast.SuperIndex:
		return v.handleSuperIndex(t, parent, scope)
	case *ast.Var:
		return v.handleVar(t, parent, scope)
	case *astext.Partial:
		return nil
	default:
		return errors.Errorf("unable to handle node type %T", t)
	}
}

func (v *NodeVisitor) visitList(list []interface{}, parent *locate.Locatable, scope locate.Scope) error {
	for _, node := range list {
		if err := v.visit(node, parent, scope); err != nil {
			return err
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

func (v *NodeVisitor) handleApply(n *ast.Apply, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Apply", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Target}
	for _, arg := range n.Arguments.Positional {
		nodes = append(nodes, arg)
	}
	for _, arg := range n.Arguments.Named {
		nodes = append(nodes, arg.Arg)
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ApplyBraceVisitor is a visitor for ApplyBrace.
type ApplyBraceVisitor struct {
	VisitApplyBrace func(a *ast.ApplyBrace) error
}

func (v *NodeVisitor) handleApplyBrace(n *ast.ApplyBrace, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ApplyBrace", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Left, n.Right}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ArrayVisitor is a visitor for Array.
type ArrayVisitor struct {
	VisitArray func(a *ast.Array) error
}

func (v *NodeVisitor) handleArray(n *ast.Array, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Array", n); err != nil {
		return err
	}

	nodes := []interface{}{}
	for _, element := range n.Elements {
		nodes = append(nodes, element)
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ArrayCompVisitor is a visitory for ArrayComp.
type ArrayCompVisitor struct {
	VisitArrayComp func(ac *ast.ArrayComp) error
}

func (v *NodeVisitor) handleArrayComp(n *ast.ArrayComp, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ArrayComp", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Body, n.Spec}

	// TODO handle this as their own type
	// forSpec := n.Spec
	// if forSpec.Outer != nil {
	// 	nodes = append(nodes, forSpec.Outer)
	// }

	// for _, ifSpec := range forSpec.Conditions {
	// 	nodes = append(nodes, ifSpec)
	// }

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// AssertVisitor is a visitor for Assert.
type AssertVisitor struct {
	VisitAssert func(n *ast.Assert) error
}

func (v *NodeVisitor) handleAssert(n *ast.Assert, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Assert", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Cond, n.Message, n.Rest}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// BinaryVisitor is a visitor for Binary.
type BinaryVisitor struct {
	VisitBinary func(n *ast.Binary) error
}

func (v *NodeVisitor) handleBinary(n *ast.Binary, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Binary", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Left, n.Right}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ConditionalVisitor is a visitor for Conditional.
type ConditionalVisitor struct {
	VisitConditional func(n *ast.Conditional) error
}

func (v *NodeVisitor) handleConditional(n *ast.Conditional, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Conditional", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Cond, n.BranchTrue, n.BranchFalse}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// DesugaredObjectFieldVisitor is a visitor for DesugaredObjectField.
type DesugaredObjectFieldVisitor struct {
	VisitDesugaredObjectField func(n ast.DesugaredObjectField) error
}

func (v *NodeVisitor) handleDesugaredObjectField(n ast.DesugaredObjectField, parent *locate.Locatable, scope locate.Scope) error {
	logrus.Debugf("visiting %T", n)
	if err := v.visitTypeIfExists("DesugaredObjectField", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Name, n.Body}

	r, err := locate.DesugaredObjectField(n, parent.Loc, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    r,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// DesugaredObjectVisitor is a visitor for DesugaredObject.
type DesugaredObjectVisitor struct {
	VisitDesugaredObject func(n *ast.DesugaredObject) error
}

func (v *NodeVisitor) handleDesugaredObject(n *ast.DesugaredObject, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("DesugaredObject", n); err != nil {
		return err
	}

	nodes := []interface{}{}
	for _, assert := range n.Asserts {
		nodes = append(nodes, assert)
	}

	for _, field := range n.Fields {
		nodes = append(nodes, field)
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// DollarVisitor is a visitor for Dollar.
type DollarVisitor struct {
	VisitDollar func(n *ast.Dollar) error
}

func (v *NodeVisitor) handleDollar(n *ast.Dollar, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Dollar", n); err != nil {
		return err
	}

	nodes := []interface{}{}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ErrorVisitor is a visitor for Error.
type ErrorVisitor struct {
	VisitError func(n *ast.Error) error
}

func (v *NodeVisitor) handleError(n *ast.Error, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Error", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Expr}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ForSpecVisitor is a visitor for ForSpec.
type ForSpecVisitor struct {
	VisitForSpec func(n *ast.ForSpec) error
}

func (v *NodeVisitor) handleForSpec(n ast.ForSpec, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ForSpec", n); err != nil {
		return errors.Wrap(err, "visit ForSpec")
	}

	r, err := locate.ForSpec(n, parent, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    r,
		Parent: parent,
	}

	nodes := []interface{}{n.Expr, n.VarName}

	if n.Outer != nil {
		nodes = append(nodes, n.Outer)
	}

	for _, ifspec := range n.Conditions {
		nodes = append(nodes, ifspec)
	}

	return v.visitList(nodes, locatable, scope)
}

// FunctionVisitor is a visitor for Function.
type FunctionVisitor struct {
	VisitFunction func(n *ast.Function) error
}

func (v *NodeVisitor) handleFunction(n *ast.Function, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Function", n); err != nil {
		return err
	}

	// TODO create new scope from params and visit the Parameters
	scopeWithParams := scope

	nodes := []interface{}{}

	loc := *n.Loc()
	if loc.Begin.Line == 0 {
		loc = parent.Loc
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    loc,
		Parent: parent,
	}

	for _, id := range n.Parameters.Required {
		p := astext.RequiredParameter{ID: id}

		r, err := locate.RequiredParameter(p, loc, string(v.Source))
		if err != nil {
			return err
		}

		l := locate.Locatable{
			Token:  p,
			Parent: locatable,
			Loc:    r,
		}

		scopeWithParams[string(id)] = l

		nodes = append(nodes, p)
	}

	for _, opt := range n.Parameters.Optional {
		nodes = append(nodes, opt)
	}

	nodes = append(nodes, n.Body)

	return v.visitList(nodes, locatable, scopeWithParams)
}

// IdentifierVisitor is a visitor for Identifier.
type IdentifierVisitor struct {
	VisitIdentifier func(n ast.Identifier) error
}

func (v *NodeVisitor) handleIdentifier(n ast.Identifier, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Identifier", n); err != nil {
		return errors.Wrap(err, "visit Identifier")
	}

	return nil
}

// ImportVisitor is a visitor for Import.
type ImportVisitor struct {
	VisitImport func(n *ast.Import) error
}

func (v *NodeVisitor) handleImport(n *ast.Import, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Import", n); err != nil {
		return err
	}

	if n.File == nil {
		return errors.New("import file value is nil")
	}

	return nil
}

// ImportStrVisitor is a visitor for ImportStr.
type ImportStrVisitor struct {
	VisitImportStr func(n *ast.ImportStr) error
}

func (v *NodeVisitor) handleImportStr(n *ast.ImportStr, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ImportStr", n); err != nil {
		return err
	}

	nodes := []interface{}{n.File}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// IndexVisitor is a visitor for Index.
type IndexVisitor struct {
	VisitIndex func(n *ast.Index) error
}

func (v *NodeVisitor) handleIndex(n *ast.Index, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Index", n); err != nil {
		return err
	}

	if n.Id != nil {
		r, err := locate.Index(n, parent, string(v.Source))
		if err != nil {
			return err
		}

		locatable := &locate.Locatable{
			Token:  n,
			Loc:    r,
			Parent: parent,
		}

		return v.visitList([]interface{}{n.Id, n.Target}, locatable, scope)
	} else if n.Index != nil {
		locatable := &locate.Locatable{
			Token:  n,
			Loc:    *n.Loc(),
			Parent: parent,
		}

		return v.visitList([]interface{}{n.Target}, locatable, scope)
	} else {
		return errors.New("index id and index were nil")
	}
}

// LiteralBooleanVisitor is a visitor for LiteralBoolean.
type LiteralBooleanVisitor struct {
	VisitLiteralBoolean func(n *ast.LiteralBoolean) error
}

func (v *NodeVisitor) handleLiteralBoolean(n *ast.LiteralBoolean, parent *locate.Locatable) error {
	if err := v.visitTypeIfExists("LiteralBoolean", n); err != nil {
		return err
	}

	return nil
}

// LiteralNullVisitor is a visitor for LiteralNull.
type LiteralNullVisitor struct {
	VisitLiteralNull func(n *ast.LiteralNull) error
}

func (v *NodeVisitor) handleLiteralNull(n *ast.LiteralNull, parent *locate.Locatable) error {
	if err := v.visitTypeIfExists("LiteralNull", n); err != nil {
		return err
	}

	return nil
}

// LiteralNumberVisitor is a visitor for LiteralNumber.
type LiteralNumberVisitor struct {
	VisitLiteralNumber func(n *ast.LiteralNumber) error
}

func (v *NodeVisitor) handleLiteralNumber(n *ast.LiteralNumber, parent *locate.Locatable) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LiteralStringVisitor is a visitor for LiteralString.
type LiteralStringVisitor struct {
	VisitLiteralString func(n *ast.LiteralString) error
}

func (v *NodeVisitor) handleLiteralString(n *ast.LiteralString, parent *locate.Locatable) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LocalVisitor is a visitor for Local.
type LocalVisitor struct {
	VisitLocal func(n *ast.Local) error
}

func (v *NodeVisitor) handleLocal(n *ast.Local, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Local", n); err != nil {
		return err
	}

	scopeWithBinds := scope

	nodes := []interface{}{}

	loc := *n.Loc()
	if loc.Begin.Line == 0 {
		loc = parent.Loc
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    loc,
		Parent: parent,
	}

	for _, bind := range n.Binds {
		r, err := locate.LocalBind(bind, loc, string(v.Source))
		if err != nil {
			return err
		}

		bindLocatable := &locate.Locatable{
			Parent: locatable,
			Token:  bind,
			Loc:    r,
		}

		idLocation, err := locate.Identifier(bind.Variable, bindLocatable, string(v.Source))
		if err != nil {
			return err
		}

		l := locate.Locatable{
			Token:  bind.Body,
			Parent: bindLocatable,
			Loc:    idLocation,
		}

		scopeWithBinds[string(bind.Variable)] = l

		nodes = append(nodes, bind)
	}

	nodes = append(nodes, n.Body)

	return v.visitList(nodes, locatable, scopeWithBinds)
}

// LocalBindVisitor is a visitor for LocalBind.
type LocalBindVisitor struct {
	VisitLocalBind func(n ast.LocalBind) error
}

func (v *NodeVisitor) handleLocalBind(lb ast.LocalBind, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("LocalBind", lb); err != nil {
		return err
	}

	// TODO merge scope with local bind params
	scopeWithParams := scope

	nodes := []interface{}{lb.Variable, lb.Body}
	if lb.Fun != nil {
		nodes = append(nodes, lb.Fun)
	}

	r, err := locate.LocalBind(lb, parent.Loc, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &locate.Locatable{
		Token:  lb,
		Loc:    r,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scopeWithParams)
}

// NamedParameterVisitor is a visitor for NamedParameter.
type NamedParameterVisitor struct {
	VisitNamedParameter func(n ast.NamedParameter) error
}

func (v *NodeVisitor) handleNamedParameter(n ast.NamedParameter, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("NamedParameter", n); err != nil {
		return errors.Wrap(err, "visit NamedParameter")
	}

	return nil
}

// ParensVisitor is a visitor for Parens.
type ParensVisitor struct {
	VisitParens func(n *ast.Parens) error
}

func (v *NodeVisitor) handleParens(n *ast.Parens, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Parens", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Inner}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ObjectCompVisitor is a visitor for ObjectComp.
type ObjectCompVisitor struct {
	VisitObjectComp func(n *ast.ObjectComp) error
}

func (v *NodeVisitor) handleObjectComp(n *ast.ObjectComp, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ObjectComp", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Spec}
	for _, field := range n.Fields {
		nodes = append(nodes, field)
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// ObjectFieldVisitor is a visitor for ObjectField.
type ObjectFieldVisitor struct {
	VisitObjectField func(n ast.ObjectField) error
}

func (v *NodeVisitor) handleObjectField(n ast.ObjectField, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("ObjectField", n); err != nil {
		return err
	}

	// TODO: need scope from params here
	scopeWithParams := scope

	tokens := []interface{}{}
	if n.Id != nil {
		tokens = append(tokens, n.Id)
	}

	if n.Expr1 != nil {
		tokens = append(tokens, n.Expr1)
	}

	tokens = append(tokens, n.Expr2, n.Expr3)

	r, err := locate.ObjectField(n, parent, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &locate.Locatable{
		Token:  n,
		Parent: parent,
		Loc:    r,
	}

	return v.visitList(tokens, locatable, scopeWithParams)
}

// ObjectVisitor is a visitor for Object.
type ObjectVisitor struct {
	VisitObject func(n *ast.Object) error
}

func (v *NodeVisitor) handleObject(n *ast.Object, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Object", n); err != nil {
		return err
	}

	// TODO get scope from local
	scopeWithLocals := scope

	nodes := []interface{}{}
	for _, field := range n.Fields {
		nodes = append(nodes, field)
	}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scopeWithLocals)
}

// SelfVisitor is a visitor for Self.
type SelfVisitor struct {
	VisitSelf func(n *ast.Self) error
}

func (v *NodeVisitor) handleSelf(n *ast.Self, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Self", n); err != nil {
		return err
	}

	return nil
}

// SliceVisitor is a visitor for Slice.
type SliceVisitor struct {
	VisitSlice func(n *ast.Slice) error
}

func (v *NodeVisitor) handleSlice(n *ast.Slice, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Slice", n); err != nil {
		return err
	}

	nodes := []interface{}{n.BeginIndex, n.EndIndex, n.Step}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// SuperIndexVisitor is a visitor for SuperIndex.
type SuperIndexVisitor struct {
	VisitSuperIndex func(n *ast.SuperIndex) error
}

func (v *NodeVisitor) handleSuperIndex(n *ast.SuperIndex, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("SuperIndex", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Index}

	locatable := &locate.Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, scope)
}

// VarVisitor is a visitor for Var.
type VarVisitor struct {
	VisitVar func(n *ast.Var) error
}

func (v *NodeVisitor) handleVar(n *ast.Var, parent *locate.Locatable, scope locate.Scope) error {
	if err := v.visitTypeIfExists("Var", n); err != nil {
		return err
	}

	return nil
}
