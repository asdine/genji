package expr

import (
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/scanner"
	"github.com/genjidb/genji/stringutil"
)

// A cmpOp is a comparison operator.
type cmpOp struct {
	*simpleOperator
}

// newCmpOp creates a comparison operator.
func newCmpOp(a, b Expr, t scanner.Token) *cmpOp {
	return &cmpOp{&simpleOperator{a, b, t}}
}

type EqOperator struct {
	*cmpOp
}

// Eq creates an expression that returns true if a equals b.
func Eq(a, b Expr) Expr {
	return &EqOperator{newCmpOp(a, b, scanner.EQ)}
}

func (op *EqOperator) String() string {
	return stringutil.Sprintf("%v = %v", op.a, op.b)
}

type NeqOperator struct {
	*cmpOp
}

// Neq creates an expression that returns true if a equals b.
func Neq(a, b Expr) Expr {
	return &NeqOperator{newCmpOp(a, b, scanner.NEQ)}
}

func (op *NeqOperator) String() string {
	return stringutil.Sprintf("%v != %v", op.a, op.b)
}

type GtOperator struct {
	*cmpOp
}

// Gt creates an expression that returns true if a is greater than b.
func Gt(a, b Expr) Expr {
	return &GtOperator{newCmpOp(a, b, scanner.GT)}
}

func (op *GtOperator) String() string {
	return stringutil.Sprintf("%v > %v", op.a, op.b)
}

type GteOperator struct {
	*cmpOp
}

// Gte creates an expression that returns true if a is greater than or equal to b.
func Gte(a, b Expr) Expr {
	return &GteOperator{newCmpOp(a, b, scanner.GTE)}
}

func (op *GteOperator) String() string {
	return stringutil.Sprintf("%v >= %v", op.a, op.b)
}

type LtOperator struct {
	*cmpOp
}

// Lt creates an expression that returns true if a is lesser than b.
func Lt(a, b Expr) Expr {
	return &LtOperator{newCmpOp(a, b, scanner.LT)}
}

func (op *LtOperator) String() string {
	return stringutil.Sprintf("%v < %v", op.a, op.b)
}

type LteOperator struct {
	*cmpOp
}

// Lte creates an expression that returns true if a is lesser than or equal to b.
func Lte(a, b Expr) Expr {
	return &LteOperator{newCmpOp(a, b, scanner.LTE)}
}

func (op *LteOperator) String() string {
	return stringutil.Sprintf("%v <= %v", op.a, op.b)
}

// Eval compares a and b together using the operator specified when constructing the CmpOp
// and returns the result of the comparison.
// Comparing with NULL always evaluates to NULL.
func (op cmpOp) Eval(env *Environment) (document.Value, error) {
	v1, v2, err := op.simpleOperator.eval(env)
	if err != nil {
		return falseLitteral, err
	}

	if v1.Type == document.NullValue || v2.Type == document.NullValue {
		return nullLitteral, nil
	}

	ok, err := op.compare(v1, v2)
	if ok {
		return trueLitteral, err
	}

	return falseLitteral, err
}

func (op cmpOp) compare(l, r document.Value) (bool, error) {
	switch op.Tok {
	case scanner.EQ:
		return l.IsEqual(r)
	case scanner.NEQ:
		return l.IsNotEqual(r)
	case scanner.GT:
		return l.IsGreaterThan(r)
	case scanner.GTE:
		return l.IsGreaterThanOrEqual(r)
	case scanner.LT:
		return l.IsLesserThan(r)
	case scanner.LTE:
		return l.IsLesserThanOrEqual(r)
	default:
		panic(stringutil.Sprintf("unknown token %v", op.Tok))
	}
}

// IsComparisonOperator returns true if e is one of
// =, !=, >, >=, <, <=, IS, IS NOT, IN, or NOT IN operators.
func IsComparisonOperator(op Operator) bool {
	switch op.(type) {
	case *EqOperator, *NeqOperator, *GtOperator, *GteOperator, *LtOperator, *LteOperator,
		*IsOperator, *IsNotOperator, *InOperator, *NotInOperator, *LikeOperator, *NotLikeOperator:
		return true
	}

	return false
}

// IsEqualOperator returns true if e is the = operator
func IsEqualOperator(op Operator) bool {
	_, ok := op.(*EqOperator)
	return ok
}

// IsAndOperator reports if e is the AND operator.
func IsAndOperator(op Operator) bool {
	_, ok := op.(*AndOp)
	return ok
}

// IsOrOperator reports if e is the OR operator.
func IsOrOperator(e Expr) bool {
	_, ok := e.(*OrOp)
	return ok
}

// IsInOperator reports if e is the IN operator.
func IsInOperator(e Expr) bool {
	_, ok := e.(*InOperator)
	return ok
}

// IsNotInOperator reports if e is the NOT IN operator.
func IsNotInOperator(e Expr) bool {
	_, ok := e.(*NotInOperator)
	return ok
}

type InOperator struct {
	*simpleOperator
}

// In creates an expression that evaluates to the result of a IN b.
func In(a, b Expr) Expr {
	return &InOperator{&simpleOperator{a, b, scanner.IN}}
}

func (op *InOperator) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	if a.Type == document.NullValue || b.Type == document.NullValue {
		return nullLitteral, nil
	}

	if b.Type != document.ArrayValue {
		return falseLitteral, nil
	}

	ok, err := document.ArrayContains(b.V.(document.Array), a)
	if err != nil {
		return nullLitteral, err
	}

	if ok {
		return trueLitteral, nil
	}
	return falseLitteral, nil
}

func (op InOperator) String() string {
	return stringutil.Sprintf("%v IN %v", op.a, op.b)
}

type NotInOperator struct {
	InOperator
}

// NotIn creates an expression that evaluates to the result of a NOT IN b.
func NotIn(a, b Expr) Expr {
	return &NotInOperator{InOperator{&simpleOperator{a, b, scanner.IN}}}
}

func (op *NotInOperator) Eval(env *Environment) (document.Value, error) {
	return invertBoolResult(op.InOperator.Eval)(env)
}

func (op *NotInOperator) String() string {
	return stringutil.Sprintf("%v NOT IN %v", op.a, op.b)
}

type IsOperator struct {
	*simpleOperator
}

// Is creates an expression that evaluates to the result of a IS b.
func Is(a, b Expr) Expr {
	return &IsOperator{&simpleOperator{a, b, scanner.IN}}
}

func (op *IsOperator) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	ok, err := a.IsEqual(b)
	if err != nil {
		return nullLitteral, err
	}
	if ok {
		return trueLitteral, nil
	}

	return falseLitteral, nil
}

func (op *IsOperator) String() string {
	return stringutil.Sprintf("%v IS %v", op.a, op.b)
}

type IsNotOperator struct {
	*simpleOperator
}

// IsNot creates an expression that evaluates to the result of a IS NOT b.
func IsNot(a, b Expr) Expr {
	return &IsNotOperator{&simpleOperator{a, b, scanner.IN}}
}

func (op *IsNotOperator) Eval(env *Environment) (document.Value, error) {
	a, b, err := op.simpleOperator.eval(env)
	if err != nil {
		return nullLitteral, err
	}

	ok, err := a.IsNotEqual(b)
	if err != nil {
		return nullLitteral, err
	}
	if ok {
		return trueLitteral, nil
	}

	return falseLitteral, nil
}

func (op *IsNotOperator) String() string {
	return stringutil.Sprintf("%v IS NOT %v", op.a, op.b)
}
