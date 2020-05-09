package query

import (
	"errors"

	"github.com/asdine/genji/database"
	"github.com/asdine/genji/document"
	"github.com/asdine/genji/document/encoding"
	"github.com/asdine/genji/engine"
	"github.com/asdine/genji/sql/query/expr"
)

// updateBufferSize is the size of the buffer used to update documents.
const updateBufferSize = 100

// UpdateStmt is a DSL that allows creating a full Update query.
type UpdateStmt struct {
	TableName string
	Pairs     map[string]expr.Expr
	WhereExpr expr.Expr
}

// IsReadOnly always returns false. It implements the Statement interface.
func (stmt UpdateStmt) IsReadOnly() bool {
	return false
}

// Run runs the Update table statement in the given transaction.
// It implements the Statement interface.
func (stmt UpdateStmt) Run(tx *database.Transaction, args []expr.Param) (Result, error) {
	var res Result

	if stmt.TableName == "" {
		return res, errors.New("missing table name")
	}

	if len(stmt.Pairs) == 0 {
		return res, errors.New("Set method not called")
	}

	stack := expr.EvalStack{
		Tx:     tx,
		Params: args,
	}

	t, err := tx.GetTable(stmt.TableName)
	if err != nil {
		return res, err
	}

	// replace store implementation by a resumable store, temporarily.
	rit := resumableIterator{store: t.Store}

	st := document.NewStream(&rit)
	st = st.Filter(whereClause(stmt.WhereExpr, stack)).Limit(updateBufferSize)

	keys := make([][]byte, updateBufferSize)
	docs := make([]document.FieldBuffer, updateBufferSize)

	for {
		var i int

		err = st.Iterate(func(d document.Document) error {
			rk, ok := d.(document.Keyer)
			if !ok {
				return errors.New("attempt to update document without key")
			}

			docs[i].Reset()
			err := docs[i].ScanDocument(d)
			if err != nil {
				return err
			}

			for fname, e := range stmt.Pairs {
				ev, err := e.Eval(expr.EvalStack{
					Tx:       tx,
					Document: d,
					Params:   args,
				})
				if err != nil && err != document.ErrFieldNotFound {
					return err
				}

				_, err = docs[i].GetByField(fname)
				switch err {
				case nil:
					// If no error, it means that the field already exists
					// and it should be replaced.
					_ = docs[i].Replace(fname, ev)
				case document.ErrFieldNotFound:
					// If the field doesn't exist,
					// it should be added to the document.
					docs[i].Set(fname, ev)
				}
			}

			// copy the key and reuse the buffer
			keys[i] = append(keys[i][0:0], rk.Key()...)
			i++

			return nil
		})

		for j := 0; j < i; j++ {
			err = t.Replace(keys[j], docs[j])
			if err != nil {
				return res, err
			}
		}

		if i < deleteBufferSize {
			break
		}

		rit.curKey = keys[i-1]
	}

	return res, err
}

// storeFromKey implements an engine.Store which iterates from a certain key.
// it is used to resume iteration.
type resumableIterator struct {
	store engine.Store

	curKey []byte
}

// AscendGreaterOrEqual uses key as pivot if pivot is nil
func (u *resumableIterator) Iterate(fn func(d document.Document) error) error {
	var d encodedDocumentWithKey
	var err error

	it := u.store.NewIterator(engine.IteratorConfig{})
	defer it.Close()

	for it.Seek(u.curKey); it.Valid(); it.Next() {
		item := it.Item()

		d.key = item.Key()
		d.EncodedDocument, err = item.ValueCopy(d.EncodedDocument)
		if err != nil {
			return err
		}

		err = fn(&d)
		if err != nil {
			return err
		}
	}

	return nil
}

type encodedDocumentWithKey struct {
	encoding.EncodedDocument

	key []byte
}

func (e encodedDocumentWithKey) Key() []byte {
	return e.key
}
