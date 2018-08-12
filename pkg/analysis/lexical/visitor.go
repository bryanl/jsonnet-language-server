package lexical

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// VisitFn visits a token.
type VisitFn func(token interface{}, parent *Locatable, env Env) error

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

// NewNodeVisitor creates an instance of Visitor.
func NewNodeVisitor(filename string, r io.Reader, opts ...VisitOpt) (*NodeVisitor, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading source")
	}

	node, err := jsonnet.SnippetToAST(filename, string(data))
	if err != nil {
		return nil, errors.Wrap(err, "parsing source")
	}

	env := Env{}

	v := &NodeVisitor{
		Node:   node,
		Parent: nil,
		Env:    env,
		Source: data,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v, nil
}

// Visit visits a node.
func (v *NodeVisitor) Visit() error {
	var parent *Locatable
	if v.Parent != nil {
		parent = &Locatable{
			Token: v.Parent,
			Loc:   *v.Parent.Loc(),
		}
	}

	return v.visit(v.Node, parent, v.Env)
}
func (v *NodeVisitor) visit(token interface{}, parent *Locatable, env Env) error {
	if token == nil {
		return nil
	}

	if v.PreVisit != nil {
		if err := v.PreVisit(token, parent, env); err != nil {
			return errors.Wrapf(err, "pre visiting %T", token)
		}
	}

	if err := v.visitToken(token, parent, env); err != nil {
		return err
	}

	if v.PostVisit != nil {
		if err := v.PostVisit(token, parent, env); err != nil {
			return errors.Wrapf(err, "post visiting %T", token)
		}
	}

	return nil
}

// nolint: gocyclo
func (v *NodeVisitor) visitToken(token interface{}, parent *Locatable, env Env) error {

	if node, ok := token.(ast.Node); ok {
		return v.handleNode(node, parent, env)
	}

	switch t := token.(type) {
	case astext.RequiredParameter:
		return v.handleIdentifier(t.ID, parent, env)
	case ast.DesugaredObjectField:
		return v.handleDesugaredObjectField(t, parent, env)
	case *ast.Identifier:
		if t == nil {
			return nil
		}
		return v.handleIdentifier(*t, parent, env)
	case ast.Identifier:
		return v.handleIdentifier(t, parent, env)
	case ast.LocalBind:
		return v.handleLocalBind(t, parent, env)
	default:
		return errors.Errorf("unable to handle token of type %T", t)
	}
}

// nolint: gocyclo
func (v *NodeVisitor) handleNode(node ast.Node, parent *Locatable, env Env) error {
	switch t := node.(type) {
	case *ast.Apply:
		return v.handleApply(t, parent, env)
	case *ast.ApplyBrace:
		return v.handleApplyBrace(t, parent, env)
	case *ast.Array:
		return v.handleArray(t, parent, env)
	case *ast.ArrayComp:
		return v.handleArrayComp(t, parent, env)
	case *ast.Binary:
		return v.handleBinary(t, parent, env)
	case *ast.Assert:
		return v.handleAssert(t, parent, env)
	case *ast.Conditional:
		return v.handleConditional(t, parent, env)
	case *ast.DesugaredObject:
		return v.handleDesugaredObject(t, parent, env)
	case *ast.Dollar:
		return v.handleDollar(t, parent, env)
	case *ast.Error:
		return v.handleError(t, parent, env)
	case *ast.Function:
		return v.handleFunction(t, parent, env)
	case *ast.Import:
		return v.handleImport(t, parent, env)
	case *ast.Index:
		return v.handleIndex(t, parent, env)
	case *ast.ImportStr:
		return v.handleImportStr(t, parent, env)
	case *ast.LiteralBoolean:
		return v.handleLiteralBoolean(t, parent)
	case *ast.LiteralNull:
		return v.handleLiteralNull(t, parent)
	case *ast.LiteralNumber:
		return v.handleLiteralNumber(t, parent)
	case *ast.LiteralString:
		return v.handleLiteralString(t, parent)
	case *ast.Local:
		return v.handleLocal(t, parent, env)
	case *ast.Parens:
		return v.handleParens(t, parent, env)
	case *ast.Object:
		return v.handleObject(t, parent, env)
	case *ast.ObjectComp:
		return v.handleObjectComp(t, parent, env)
	case *ast.Self:
		return v.handleSelf(t, parent, env)
	case *ast.Slice:
		return v.handleSlice(t, parent, env)
	case *ast.SuperIndex:
		return v.handleSuperIndex(t, parent, env)
	case *ast.Var:
		return v.handleVar(t, parent, env)
	default:
		return errors.Errorf("unable to handle node type %T", t)
	}
}

func (v *NodeVisitor) visitList(list []interface{}, parent *Locatable, env Env) error {
	for _, node := range list {
		if err := v.visit(node, parent, env); err != nil {
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

func (v *NodeVisitor) handleApply(n *ast.Apply, parent *Locatable, env Env) error {
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

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ApplyBraceVisitor is a visitor for ApplyBrace.
type ApplyBraceVisitor struct {
	VisitApplyBrace func(a *ast.ApplyBrace) error
}

func (v *NodeVisitor) handleApplyBrace(n *ast.ApplyBrace, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("ApplyBrace", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Left, n.Right}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ArrayVisitor is a visitor for Array.
type ArrayVisitor struct {
	VisitArray func(a *ast.Array) error
}

func (v *NodeVisitor) handleArray(n *ast.Array, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Array", n); err != nil {
		return err
	}

	nodes := []interface{}{}
	for _, element := range n.Elements {
		nodes = append(nodes, element)
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ArrayCompVisitor is a visitory for ArrayComp.
type ArrayCompVisitor struct {
	VisitArrayComp func(ac *ast.ArrayComp) error
}

func (v *NodeVisitor) handleArrayComp(n *ast.ArrayComp, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("ArrayComp", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Body, n.Spec}
	forSpec := n.Spec
	if forSpec.Outer != nil {
		nodes = append(nodes, forSpec.Outer)
	}

	for _, ifSpec := range forSpec.Conditions {
		nodes = append(nodes, ifSpec.Expr)
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// AssertVisitor is a visitor for Assert.
type AssertVisitor struct {
	VisitAssert func(n *ast.Assert) error
}

func (v *NodeVisitor) handleAssert(n *ast.Assert, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Assert", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Cond, n.Message, n.Rest}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// BinaryVisitor is a visitor for Binary.
type BinaryVisitor struct {
	VisitBinary func(n *ast.Binary) error
}

func (v *NodeVisitor) handleBinary(n *ast.Binary, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Binary", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Left, n.Right}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ConditionalVisitor is a visitor for Conditional.
type ConditionalVisitor struct {
	VisitConditional func(n *ast.Conditional) error
}

func (v *NodeVisitor) handleConditional(n *ast.Conditional, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Conditional", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Cond, n.BranchTrue, n.BranchFalse}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// DesugaredObjectFieldVisitor is a visitor for DesugaredObjectField.
type DesugaredObjectFieldVisitor struct {
	VisitDesugaredObjectField func(n ast.DesugaredObjectField) error
}

func (v *NodeVisitor) handleDesugaredObjectField(n ast.DesugaredObjectField, parent *Locatable, env Env) error {
	logrus.Infof("visiting %T", n)
	if err := v.visitTypeIfExists("DesugaredObjectField", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Name, n.Body}

	r, err := locate.DesugaredObjectField(n, parent.Loc, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    r,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// DesugaredObjectVisitor is a visitor for DesugaredObject.
type DesugaredObjectVisitor struct {
	VisitDesugaredObject func(n *ast.DesugaredObject) error
}

func (v *NodeVisitor) handleDesugaredObject(n *ast.DesugaredObject, parent *Locatable, env Env) error {
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

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// DollarVisitor is a visitor for Dollar.
type DollarVisitor struct {
	VisitDollar func(n *ast.Dollar) error
}

func (v *NodeVisitor) handleDollar(n *ast.Dollar, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Dollar", n); err != nil {
		return err
	}

	nodes := []interface{}{}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ErrorVisitor is a visitor for Error.
type ErrorVisitor struct {
	VisitError func(n *ast.Error) error
}

func (v *NodeVisitor) handleError(n *ast.Error, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Error", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Expr}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// FunctionVisitor is a visitor for Function.
type FunctionVisitor struct {
	VisitFunction func(n *ast.Function) error
}

func (v *NodeVisitor) handleFunction(n *ast.Function, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Function", n); err != nil {
		return err
	}

	// TODO create new env from params and visit the Parameters
	envWithParams := env

	nodes := []interface{}{}

	for _, id := range n.Parameters.Required {
		nodes = append(nodes, astext.RequiredParameter{ID: id})
	}

	for _, opt := range n.Parameters.Optional {
		nodes = append(nodes, opt)
	}

	nodes = append(nodes, n.Body)

	loc := *n.Loc()
	if loc.Begin.Line == 0 {
		loc = parent.Loc
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    loc,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, envWithParams)
}

// IdentifierVisitor is a visitor for Identifier.
type IdentifierVisitor struct {
	VisitIdentifier func(n ast.Identifier) error
}

func (v *NodeVisitor) handleIdentifier(n ast.Identifier, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Identifier", n); err != nil {
		return errors.Wrap(err, "visit Identifier")
	}

	return nil
}

// ImportVisitor is a visitor for Import.
type ImportVisitor struct {
	VisitImport func(n *ast.Import) error
}

func (v *NodeVisitor) handleImport(n *ast.Import, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Import", n); err != nil {
		return err
	}

	nodes := []interface{}{n.File}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ImportStrVisitor is a visitor for ImportStr.
type ImportStrVisitor struct {
	VisitImportStr func(n *ast.ImportStr) error
}

func (v *NodeVisitor) handleImportStr(n *ast.ImportStr, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("ImportStr", n); err != nil {
		return err
	}

	nodes := []interface{}{n.File}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// IndexVisitor is a visitor for Index.
type IndexVisitor struct {
	VisitIndex func(n *ast.Index) error
}

func (v *NodeVisitor) handleIndex(n *ast.Index, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Index", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Target, n.Index}
	if n.Id != nil {
		nodes = append(nodes, n.Id)
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// LiteralBooleanVisitor is a visitor for LiteralBoolean.
type LiteralBooleanVisitor struct {
	VisitLiteralBoolean func(n *ast.LiteralBoolean) error
}

func (v *NodeVisitor) handleLiteralBoolean(n *ast.LiteralBoolean, parent *Locatable) error {
	if err := v.visitTypeIfExists("LiteralBoolean", n); err != nil {
		return err
	}

	return nil
}

// LiteralNullVisitor is a visitor for LiteralNull.
type LiteralNullVisitor struct {
	VisitLiteralNull func(n *ast.LiteralNull) error
}

func (v *NodeVisitor) handleLiteralNull(n *ast.LiteralNull, parent *Locatable) error {
	if err := v.visitTypeIfExists("LiteralNull", n); err != nil {
		return err
	}

	return nil
}

// LiteralNumberVisitor is a visitor for LiteralNumber.
type LiteralNumberVisitor struct {
	VisitLiteralNumber func(n *ast.LiteralNumber) error
}

func (v *NodeVisitor) handleLiteralNumber(n *ast.LiteralNumber, parent *Locatable) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LiteralStringVisitor is a visitor for LiteralString.
type LiteralStringVisitor struct {
	VisitLiteralString func(n *ast.LiteralString) error
}

func (v *NodeVisitor) handleLiteralString(n *ast.LiteralString, parent *Locatable) error {
	if err := v.visitTypeIfExists("LiteralString", n); err != nil {
		return err
	}

	return nil
}

// LocalVisitor is a visitor for Local.
type LocalVisitor struct {
	VisitLocal func(n *ast.Local) error
}

func (v *NodeVisitor) handleLocal(n *ast.Local, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Local", n); err != nil {
		return err
	}

	// TODO create new envWithBinds by merging tree.envFromLocalBinds(n)
	envWithBinds := env

	nodes := []interface{}{}

	for _, bind := range n.Binds {
		nodes = append(nodes, bind)
	}

	nodes = append(nodes, n.Body)

	loc := *n.Loc()
	if loc.Begin.Line == 0 {
		loc = parent.Loc
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    loc,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, envWithBinds)
}

// LocalBindVisitor is a visitor for LocalBind.
type LocalBindVisitor struct {
	VisitLocalBind func(n ast.LocalBind) error
}

func (v *NodeVisitor) handleLocalBind(lb ast.LocalBind, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("LocalBind", lb); err != nil {
		return err
	}

	// TODO merge env with local bind params
	envWithParams := env

	nodes := []interface{}{lb.Variable, lb.Body}
	if lb.Fun != nil {
		nodes = append(nodes, lb.Fun)
	}

	r, err := locate.LocalBind(lb, parent.Loc, string(v.Source))
	if err != nil {
		return err
	}

	locatable := &Locatable{
		Token:  lb,
		Loc:    r,
		Parent: parent,
	}

	return v.visitList(nodes, locatable, envWithParams)
}

// ParensVisitor is a visitor for Parens.
type ParensVisitor struct {
	VisitParens func(n *ast.Parens) error
}

func (v *NodeVisitor) handleParens(n *ast.Parens, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Parens", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Inner}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ObjectCompVisitor is a visitor for ObjectComp.
type ObjectCompVisitor struct {
	VisitObjectComp func(n *ast.ObjectComp) error
}

func (v *NodeVisitor) handleObjectComp(n *ast.ObjectComp, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("ObjectComp", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Spec}
	for _, field := range n.Fields {
		nodes = append(nodes, field)
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// ObjectFieldVisitor is a visitor for ObjectField.
type ObjectFieldVisitor struct {
	VisitObjectField func(n ast.ObjectField) error
}

func (v *NodeVisitor) handleObjectField(n ast.ObjectField, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("ObjectField", n); err != nil {
		return err
	}

	// TODO: need env from params here
	envWithParams := env

	tokens := []interface{}{}
	if n.Id != nil {
		tokens = append(tokens, n.Id)
	}

	if n.Expr1 != nil {
		tokens = append(tokens, n.Expr1)
	}

	tokens = append(tokens, n.Expr2, n.Expr3)

	locatable := &Locatable{
		Token:  n,
		Parent: parent,
	}

	return v.visitList(tokens, locatable, envWithParams)
}

// ObjectVisitor is a visitor for Object.
type ObjectVisitor struct {
	VisitObject func(n *ast.Object) error
}

func (v *NodeVisitor) handleObject(n *ast.Object, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Object", n); err != nil {
		return err
	}

	// TODO get env from local
	envWithLocals := env

	nodes := []interface{}{}
	for _, field := range n.Fields {
		nodes = append(nodes, field)
	}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, envWithLocals)
}

// SelfVisitor is a visitor for Self.
type SelfVisitor struct {
	VisitSelf func(n *ast.Self) error
}

func (v *NodeVisitor) handleSelf(n *ast.Self, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Self", n); err != nil {
		return err
	}

	return nil
}

// SliceVisitor is a visitor for Slice.
type SliceVisitor struct {
	VisitSlice func(n *ast.Slice) error
}

func (v *NodeVisitor) handleSlice(n *ast.Slice, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Slice", n); err != nil {
		return err
	}

	nodes := []interface{}{n.BeginIndex, n.EndIndex, n.Step}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// SuperIndexVisitor is a visitor for SuperIndex.
type SuperIndexVisitor struct {
	VisitSuperIndex func(n *ast.SuperIndex) error
}

func (v *NodeVisitor) handleSuperIndex(n *ast.SuperIndex, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("SuperIndex", n); err != nil {
		return err
	}

	nodes := []interface{}{n.Index}

	locatable := &Locatable{
		Token:  n,
		Loc:    *n.Loc(),
		Parent: parent,
	}

	return v.visitList(nodes, locatable, env)
}

// VarVisitor is a visitor for Var.
type VarVisitor struct {
	VisitVar func(n *ast.Var) error
}

func (v *NodeVisitor) handleVar(n *ast.Var, parent *Locatable, env Env) error {
	if err := v.visitTypeIfExists("Var", n); err != nil {
		return err
	}

	return nil
}
