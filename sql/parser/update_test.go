package parser

import (
	"testing"

	"github.com/genjidb/genji/sql/query"
	"github.com/genjidb/genji/sql/query/expr"
	"github.com/stretchr/testify/require"
)

func TestParserUdpate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected query.Statement
		errored  bool
	}{
		{"SET/No cond", "UPDATE test SET a = 1",
			query.UpdateStmt{
				TableName: "test",
				SetPairs: map[string]expr.Expr{
					"a": expr.IntValue(1),
				},
			},
			false},
		{"SET/No con field with double quotes", "UPDATE test SET \"favorite game\" = \"splinter cell\"",
			query.UpdateStmt{
				TableName: "test",
				SetPairs: map[string]expr.Expr{
					"favorite game": expr.TextValue("splinter cell"),
				},
			},
			false},
		{"SET/With cond", "UPDATE test SET a = 1, b = 2 WHERE age = 10",
			query.UpdateStmt{
				TableName: "test",
				SetPairs: map[string]expr.Expr{
					"a": expr.IntValue(1),
					"b": expr.IntValue(2),
				},
				WhereExpr: expr.Eq(expr.FieldSelector([]string{"age"}), expr.IntValue(10)),
			},
			false},
		{"UNSET/No cond", "UPDATE test UNSET a",
			query.UpdateStmt{
				TableName:   "test",
				UnsetFields: []string{"a"},
			},
			false},
		{"UNSET/With cond", "UPDATE test UNSET a, b WHERE age = 10",
			query.UpdateStmt{
				TableName:   "test",
				UnsetFields: []string{"a", "b"},
				WhereExpr:   expr.Eq(expr.FieldSelector([]string{"age"}), expr.IntValue(10)),
			},
			false},
		{"Trailing comma", "UPDATE test SET a = 1, WHERE age = 10", nil, true},
		{"No SET", "UPDATE test WHERE age = 10", nil, true},
		{"No pair", "UPDATE test SET WHERE age = 10", nil, true},
		{"query.Field only", "UPDATE test SET a WHERE age = 10", nil, true},
		{"No value", "UPDATE test SET a = WHERE age = 10", nil, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := ParseQuery(test.s)
			if test.errored {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, q.Statements, 1)
			require.EqualValues(t, test.expected, q.Statements[0])
		})
	}
}
