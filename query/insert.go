package query

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/asdine/genji/database"
	"github.com/asdine/genji/query/expr"
	"github.com/asdine/genji/record"
	"github.com/asdine/genji/value"
)

// InsertStmt is a DSL that allows creating a full Insert query.
// It is typically created using the Insert function.
type InsertStmt struct {
	tableName  string
	fieldNames []string
	values     expr.LitteralExprList
	records    []interface{}
}

// Insert creates a DSL equivalent to the SQL Insert command.
func Insert() InsertStmt {
	return InsertStmt{}
}

// IsReadOnly always returns false. It implements the Statement interface.
func (stmt InsertStmt) IsReadOnly() bool {
	return false
}

// Run runs the Insert statement in the given transaction.
// It implements the Statement interface.
func (stmt InsertStmt) Run(tx *database.Tx, args []driver.NamedValue) Result {
	return stmt.exec(tx, args)
}

// Exec the Insert query within tx.
func (stmt InsertStmt) Exec(tx *database.Tx, args ...interface{}) Result {
	return stmt.exec(tx, argsToNamedValues(args))
}

// Into indicates in which table to write the new records.
// Calling this method before Run is mandatory.
func (stmt InsertStmt) Into(tableName string) InsertStmt {
	stmt.tableName = tableName
	return stmt
}

// Fields to associate with values passed to the Values method.
func (stmt InsertStmt) Fields(fieldNames ...string) InsertStmt {
	stmt.fieldNames = append(stmt.fieldNames, fieldNames...)
	return stmt
}

// Values is called to add one record. The list of supplied values will be used as the fields
// of this record.
func (stmt InsertStmt) Values(values ...expr.Expr) InsertStmt {
	stmt.values = append(stmt.values, expr.LitteralExprList(values))
	return stmt
}

// Records is called to add one or more records.
func (stmt InsertStmt) Records(records ...interface{}) InsertStmt {
	for _, r := range records {
		stmt.records = append(stmt.records, r)
	}

	return stmt
}

type KVPair struct {
	K string
	V expr.Expr
}

func (stmt InsertStmt) Pairs(pairs ...KVPair) InsertStmt {
	stmt.records = append(stmt.records, pairs)

	return stmt
}

func (stmt InsertStmt) exec(tx *database.Tx, args []driver.NamedValue) Result {
	if stmt.tableName == "" {
		return Result{err: errors.New("missing table name")}
	}

	if stmt.values == nil && stmt.records == nil {
		return Result{err: errors.New("values and records are empty")}
	}

	t, err := tx.GetTable(stmt.tableName)
	if err != nil {
		return Result{err: err}
	}

	stack := expr.EvalStack{
		Tx:     tx,
		Params: args,
	}

	if len(stmt.records) > 0 {
		return stmt.insertRecords(t, stack)
	}

	return stmt.insertValues(t, stack)
}

type paramExtractor interface {
	Extract(params []driver.NamedValue) (interface{}, error)
}

func (stmt InsertStmt) insertRecords(t *database.Table, stack expr.EvalStack) Result {
	if len(stmt.fieldNames) > 0 {
		return Result{err: errors.New("can't provide a field list with RECORDS clause")}
	}

	var res Result
	var err error

	for _, rec := range stmt.records {
		var r record.Record

		switch tp := rec.(type) {
		case record.Record:
			r = tp
		case paramExtractor:
			v, err := tp.Extract(stack.Params)
			if err != nil {
				return Result{err: err}
			}

			var ok bool
			r, ok = v.(record.Record)
			if !ok {
				return Result{err: fmt.Errorf("unsupported parameter of type %t, expecting record.Record", v)}
			}
		case []KVPair:
			var fb record.FieldBuffer
			for _, pair := range tp {
				v, err := pair.V.Eval(stack)
				if err != nil {
					res.err = err
					return res
				}

				if v.IsList {
					res.err = errors.New("invalid values")
					return res
				}

				fb.Add(record.Field{Name: pair.K, Value: v.Value.Value})
			}
			r = &fb
		}

		res.lastInsertRecordID, err = t.Insert(r)
		if err != nil {
			return Result{err: err}
		}

		res.rowsAffected++
	}

	res.Stream = record.NewStream(record.NewIteratorFromRecords())
	return res
}

func (stmt InsertStmt) insertValues(t *database.Table, stack expr.EvalStack) Result {
	var res Result

	// iterate over all of the records (r1, r2, r3, ...)
	for _, e := range stmt.values {
		var fb record.FieldBuffer

		v, err := e.Eval(stack)
		if err != nil {
			return Result{err: err}
		}

		// each record must be a list of values
		// (e1, e2, e3, ...)
		if !v.IsList {
			return Result{err: errors.New("invalid values")}
		}

		if len(stmt.fieldNames) != len(v.List) {
			return Result{err: fmt.Errorf("%d values for %d fields", len(v.List), len(stmt.fieldNames))}
		}

		// iterate over each value
		for i, v := range v.List {
			// get the field name
			fieldName := stmt.fieldNames[i]

			var lv *expr.LitteralValue

			// each value must be either a LitteralValue or a LitteralValueList with exactly
			// one value
			if !v.IsList {
				lv = &v.Value
			} else {
				if len(v.List) == 1 {
					if val := v.List[0]; !val.IsList {
						lv = &val.Value
					}
				}
				return Result{err: fmt.Errorf("value expected, got list")}
			}

			// Assign the value to the field and add it to the record
			fb.Add(record.Field{
				Name: fieldName,
				Value: value.Value{
					Type: lv.Type,
					Data: lv.Data,
				},
			})
		}

		res.lastInsertRecordID, err = t.Insert(&fb)
		if err != nil {
			return Result{err: err}
		}

		res.rowsAffected++
	}

	res.Stream = record.NewStream(record.NewIteratorFromRecords())

	return res
}
