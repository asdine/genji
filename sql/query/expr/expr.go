package expr

import (
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/scanner"
)

var (
	trueLitteral  = document.NewBoolValue(true)
	falseLitteral = document.NewBoolValue(false)
	nullLitteral  = document.NewNullValue()
)

// An Expr evaluates to a value.
type Expr interface {
	Eval(EvalStack) (document.Value, error)
	Equal(Expr) bool
}

// EvalStack contains information about the context in which
// the expression is evaluated.
// Any of the members can be nil except the transaction.
type EvalStack struct {
	Tx       *database.Transaction
	Document document.Document
	Params   []Param
	Cfg      *database.TableConfig
}

type simpleOperator struct {
	a, b Expr
	Tok  scanner.Token
}

func (op simpleOperator) Precedence() int {
	return op.Tok.Precedence()
}

func (op simpleOperator) LeftHand() Expr {
	return op.a
}

func (op simpleOperator) RightHand() Expr {
	return op.b
}

func (op *simpleOperator) SetLeftHandExpr(a Expr) {
	op.a = a
}

func (op *simpleOperator) SetRightHandExpr(b Expr) {
	op.b = b
}

func (op *simpleOperator) Token() scanner.Token {
	return op.Tok
}

func (op *simpleOperator) eval(ctx EvalStack) (document.Value, document.Value, error) {
	va, err := op.a.Eval(ctx)
	if err != nil {
		return nullLitteral, nullLitteral, err
	}

	vb, err := op.b.Eval(ctx)
	if err != nil {
		return nullLitteral, nullLitteral, err
	}

	return va, vb, nil
}

// Equal compares this expression with the other expression and returns
// true if they are equal.
func (op *simpleOperator) Equal(other Expr) bool {
	if other == nil {
		return false
	}

	oop, ok := other.(Operator)
	if !ok {
		return false
	}

	return op.Tok == oop.Token() &&
		op.a.Equal(oop.LeftHand()) &&
		op.b.Equal(oop.RightHand())
}

// An Operator is a binary expression that
// takes two operands and executes an operation on them.
type Operator interface {
	Expr

	Precedence() int
	LeftHand() Expr
	RightHand() Expr
	SetLeftHandExpr(Expr)
	SetRightHandExpr(Expr)
	Token() scanner.Token
}
