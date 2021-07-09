package expr

import (
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/internal/environment"
	"github.com/genjidb/genji/internal/stringutil"
)

var (
	TrueLiteral  = document.NewBoolValue(true)
	FalseLiteral = document.NewBoolValue(false)
	NullLiteral  = document.NewNullValue()
)

// An Expr evaluates to a value.
type Expr interface {
	Eval(*environment.Environment) (document.Value, error)
	String() string
}

type isEqualer interface {
	IsEqual(Expr) bool
}

// Equal reports whether a and b are equal by first calling IsEqual
// if they have an IsEqual method with this signature:
//   IsEqual(Expr) bool
// If not, it returns whether a and b values are equal.
func Equal(a, b Expr) bool {
	if aa, ok := a.(isEqualer); ok {
		return aa.IsEqual(b)
	}

	if bb, ok := b.(isEqualer); ok {
		return bb.IsEqual(a)
	}

	return a == b
}

// Parentheses is a special expression which turns
// any sub-expression as unary.
// It hides the underlying operator, if any, from the parser
// so that it doesn't get reordered by precedence.
type Parentheses struct {
	E Expr
}

// Eval calls the underlying expression Eval method.
func (p Parentheses) Eval(env *environment.Environment) (document.Value, error) {
	return p.E.Eval(env)
}

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (p Parentheses) IsEqual(other Expr) bool {
	if other == nil {
		return false
	}

	o, ok := other.(Parentheses)
	if !ok {
		return false
	}

	return Equal(p.E, o.E)
}

func (p Parentheses) String() string {
	return stringutil.Sprintf("(%v)", p.E)
}

func invertBoolResult(f func(env *environment.Environment) (document.Value, error)) func(env *environment.Environment) (document.Value, error) {
	return func(env *environment.Environment) (document.Value, error) {
		v, err := f(env)

		if err != nil {
			return v, err
		}
		if v == TrueLiteral {
			return FalseLiteral, nil
		}
		if v == FalseLiteral {
			return TrueLiteral, nil
		}
		return v, nil
	}
}

// NamedExpr is an expression with a name.
type NamedExpr struct {
	Expr

	ExprName string
}

// Name returns ExprName.
func (e *NamedExpr) Name() string {
	return e.ExprName
}

func (e *NamedExpr) String() string {
	return stringutil.Sprintf("%s", e.Expr)
}

func Walk(e Expr, fn func(Expr) bool) bool {
	if e == nil {
		return true
	}
	if !fn(e) {
		return false
	}

	switch t := e.(type) {
	case Operator:
		if !Walk(t.LeftHand(), fn) {
			return false
		}
		if !Walk(t.RightHand(), fn) {
			return false
		}
	case *NamedExpr:
		return Walk(t.Expr, fn)
	case Function:
		for _, p := range t.Params() {
			if !Walk(p, fn) {
				return false
			}
		}
	}

	return true
}

type NextValueFor struct {
	SeqName string
}

// Eval calls the underlying expression Eval method.
func (n NextValueFor) Eval(env *environment.Environment) (document.Value, error) {
	catalog := env.GetCatalog()
	tx := env.GetTx()

	if catalog == nil || tx == nil {
		return NullLiteral, stringutil.Errorf(`NEXT VALUE FOR cannot be evaluated`)
	}

	seq, err := catalog.GetSequence(n.SeqName)
	if err != nil {
		return NullLiteral, err
	}

	i, err := seq.Next(tx, catalog)
	if err != nil {
		return NullLiteral, err
	}

	return document.NewIntegerValue(i), nil
}

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (n NextValueFor) IsEqual(other Expr) bool {
	if other == nil {
		return false
	}

	o, ok := other.(NextValueFor)
	if !ok {
		return false
	}

	return o.SeqName == n.SeqName
}

func (n NextValueFor) String() string {
	return stringutil.Sprintf("NEXT VALUE FOR %s", n.SeqName)
}
