package statement

import (
	"errors"

	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/internal/database"
	"github.com/genjidb/genji/internal/expr"
	"github.com/genjidb/genji/internal/planner"
	"github.com/genjidb/genji/internal/stream"
)

// ExplainStmt is a Statement that
// displays information about how a statement
// is going to be executed, without executing it.
type ExplainStmt struct {
	Statement Statement
}

// Run analyses the inner statement and displays its execution plan.
// If the statement is a stream, Optimize will be called prior to
// displaying all the operations.
// Explain currently only works on SELECT, UPDATE, INSERT and DELETE statements.
func (stmt *ExplainStmt) Run(tx *database.Transaction, params []expr.Param) (Result, error) {
	var ss *StreamStmt
	var err error
	var res Result

	switch t := stmt.Statement.(type) {
	case *SelectStmt:
		ss, err = t.ToStream()
	case *UpdateStmt:
		ss = t.ToStream()
	case *InsertStmt:
		ss, err = t.ToStream()
	case *DeleteStmt:
		ss, err = t.ToStream()
	default:
		return Result{}, errors.New("EXPLAIN only works on INSERT, SELECT, UPDATE AND DELETE statements")
	}
	if err != nil {
		return res, err
	}

	s, err := planner.Optimize(ss.Stream, tx)
	if err != nil {
		return Result{}, err
	}

	var plan string
	if s != nil {
		plan = s.String()
	} else {
		plan = "<no exec>"
	}

	newStatement := StreamStmt{
		Stream: &stream.Stream{
			Op: stream.Project(
				&expr.NamedExpr{
					ExprName: "plan",
					Expr:     expr.LiteralValue(document.NewTextValue(plan)),
				}),
		},
		ReadOnly: true,
	}
	return newStatement.Run(tx, params)
}

// IsReadOnly indicates that this statement doesn't write anything into
// the database.
func (s *ExplainStmt) IsReadOnly() bool {
	return true
}
