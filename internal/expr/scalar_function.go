package expr

import (
	"strings"

	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/internal/environment"
	"github.com/genjidb/genji/internal/stringutil"
)

// A ScalarFunctionDef is the definition type for functions which operates on scalar values in contrast to other SQL functions
// such as the SUM aggregator wich operates on expressions instead.
//
// This difference allows to simply define them with a CallFn function that takes multiple document.Value and
// return another document.Value, rather than having to manually evaluate expressions (see FunctionDef).
type ScalarFunctionDef struct {
	name   string
	arity  int
	callFn func(...document.Value) (document.Value, error)
}

// Name returns the defined function named (as an indent, so no parentheses).
func (fd *ScalarFunctionDef) Name() string {
	return fd.name
}

// String returns the defined function name and its arguments.
func (fd *ScalarFunctionDef) String() string {
	args := make([]string, 0, fd.arity)
	for i := 0; i < fd.arity; i++ {
		args = append(args, stringutil.Sprintf("arg%d", i+1))
	}
	return stringutil.Sprintf("%s(%s)", fd.name, strings.Join(args, ", "))
}

// Function returns a Function expr node.
func (fd *ScalarFunctionDef) Function(args ...Expr) (Function, error) {
	if len(args) != fd.arity {
		return nil, stringutil.Errorf("%s takes %d argument, not %d", fd.String(), fd.arity, len(args))
	}
	return &ScalarFunction{
		params: args,
		def:    fd,
	}, nil
}

// Arity return the arity of the defined function.
func (fd *ScalarFunctionDef) Arity() int {
	return fd.arity
}

// A ScalarFunction is a function which operates on scalar values in contrast to other SQL functions
// such as the SUM aggregator wich operates on expressions instead.
type ScalarFunction struct {
	def    *ScalarFunctionDef
	params []Expr
}

// Eval returns a document.Value based on the given environment and the underlying function
// definition.
func (sf *ScalarFunction) Eval(env *environment.Environment) (document.Value, error) {
	args, err := sf.evalParams(env)
	if err != nil {
		return document.Value{}, err
	}
	return sf.def.callFn(args...)
}

// evalParams evaluate all arguments given to the function in the context of the given environmment.
func (sf *ScalarFunction) evalParams(env *environment.Environment) ([]document.Value, error) {
	values := make([]document.Value, 0, len(sf.params))
	for _, param := range sf.params {
		v, err := param.Eval(env)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// String returns a string represention of the function expression and its arguments.
func (sf *ScalarFunction) String() string {
	return stringutil.Sprintf("%s(%v)", sf.def.name, sf.params)
}

// Params return the function arguments.
func (sf *ScalarFunction) Params() []Expr {
	return sf.params
}
