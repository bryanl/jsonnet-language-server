package jsonnet

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"

	"github.com/google/go-jsonnet/ast"
)

// manifestNode
func (i *interpreter) manifestNode(trace *TraceElement, v value) (interface{}, error) {
	e := &evaluator{i: i, trace: trace}
	switch v := v.(type) {

	case *valueBoolean:
		return &ast.LiteralBoolean{Value: v.value}, nil

	case *valueFunction:
		cl, ok := v.ec.(*closure)
		if !ok {
			return nil, errors.New("expected a closure")
		}

		return &cl.params, nil

	case *valueNumber:
		return &ast.LiteralNumber{Value: v.value, OriginalString: unparseNumber(v.value)}, nil

	case *valueString:
		// TODO: get string type
		return &ast.LiteralString{Value: v.getString()}, nil

	case *valueNull:
		return &ast.LiteralNull{}, nil

	case *valueArray:
		array := &ast.Array{}
		for _, th := range v.elements {
			elVal, err := e.evaluate(th)
			if err != nil {
				return nil, err
			}
			val, err := i.manifestNode(trace, elVal)
			if err != nil {
				return nil, err
			}

			node, ok := val.(ast.Node)
			if !ok {
				return nil, errors.New("array element was not a node")
			}
			array.Elements = append(array.Elements, node)
		}
		return array, nil

	case valueObject:
		fhm := objectFieldsVisibility(v)
		fieldNames := make([]string, len(fhm))
		c := 0
		for fieldName := range fhm {
			fieldNames[c] = fieldName
			c++
		}
		sort.Strings(fieldNames)

		err := checkAssertions(e, v)
		if err != nil {
			return nil, err
		}

		result := &ast.Object{}
		for j := range fieldNames {
			fieldName := fieldNames[j]
			fieldValue, err := v.index(e, fieldName)
			if err != nil {
				return nil, err
			}

			// ! this causes building to loop
			if fieldName == "openAPIV3SchemaType" {
				continue
			}

			val, err := i.manifestNode(trace, fieldValue)
			if err != nil {
				return nil, err
			}

			var node ast.Node
			var params *ast.Parameters
			switch val := val.(type) {
			case ast.Node:
				node = val
			case *Parameters:
				params = &ast.Parameters{
					Required: val.required,
					Optional: []ast.NamedParameter{},
				}

				// TODO reenable optional args
				// params.Optional = make([]ast.NamedParameter, len(val.optional))
				// for j := range val.optional {
				// 	opt := val.optional[j]
				// 	p := ast.NamedParameter{
				// 		DefaultArg: opt.defaultArg,
				// 		Name:       opt.name,
				// 	}
				// 	params.Optional[j] = p
				// }
			}

			fieldID := ast.Identifier(fieldName)
			field := ast.ObjectField{
				Hide:   fhm[fieldName],
				Kind:   ast.ObjectFieldID,
				Id:     &fieldID,
				Expr2:  node,
				Params: params,
			}
			result.Fields = append(result.Fields, field)
		}
		return result, nil

	default:
		return nil, makeRuntimeError(
			fmt.Sprintf("manifesting this value not implemented yet: %s", reflect.TypeOf(v)),
			i.getCurrentStackTrace(trace),
		)

	}
}

func buildOutputNode(i *interpreter, trace *TraceElement, result value) (ast.Node, error) {
	item, err := i.manifestNode(trace, result)
	if err != nil {
		return nil, err
	}
	if node, ok := item.(ast.Node); ok {
		return node, nil
	}

	return nil, errors.New("result was not a node")
}

func evaluateToNode(node ast.Node, ext vmExtMap, tla vmExtMap, nativeFuncs map[string]*NativeFunction,
	maxStack int, importer Importer) (ast.Node, error) {

	i, err := buildInterpreter(ext, nativeFuncs, maxStack, importer)
	if err != nil {
		return nil, err
	}

	result, manifestationTrace, err := evaluateAux(i, node, tla)
	if err != nil {
		return nil, err
	}

	newNode, err := buildOutputNode(i, manifestationTrace, result)
	if err != nil {
		return nil, err
	}

	return newNode, nil
}

func (vm *VM) evaluateToNode(filename string, snippet string) (ast.Node, error) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())
		}
	}()
	node, err := snippetToAST(filename, snippet)
	if err != nil {
		return nil, err
	}

	output, err := evaluateToNode(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importer)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (vm *VM) EvaluateToNode(filename string, snippet string) (ast.Node, error) {
	output, err := vm.evaluateToNode(filename, snippet)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}

	return output, nil
}
