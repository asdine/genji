package stream_test

import (
	"testing"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/parser"
	"github.com/genjidb/genji/sql/query/expr"
	"github.com/genjidb/genji/stream"
	"github.com/stretchr/testify/require"
)

func TestTableIterator(t *testing.T) {
	tests := []struct {
		name      string
		documents []string
		fails     bool
	}{
		{"empty", nil, false},
		{"ok", []string{`{"a": 1}`, `{"a": 2}`}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := genji.Open(":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = db.Exec("CREATE TABLE test")
			require.NoError(t, err)

			for _, doc := range test.documents {
				err = db.Exec("INSERT INTO test VALUES ?", document.NewFromJSON([]byte(doc)))
				require.NoError(t, err)
			}

			tx, err := db.Begin(false)
			require.NoError(t, err)
			defer tx.Rollback()

			table, err := tx.GetTable("test")
			require.NoError(t, err)

			s := stream.New(stream.NewTableIterator(table, nil))

			var i int
			err = s.Iterate(func(env *expr.Environment) error {
				d, ok := env.GetDocument()
				require.True(t, ok)
				require.JSONEq(t, test.documents[i], document.NewDocumentValue(d).String())
				i++
				return nil
			})
			if test.fails {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("String", func(t *testing.T) {
		require.Equal(t, `sort(a)`, stream.Sort(parser.MustParseExpr("a")).String())
	})
}