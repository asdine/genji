package parser

import (
	"testing"

	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/query"
	"github.com/stretchr/testify/require"
)

func TestParserCreateTable(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected query.Statement
		errored  bool
	}{
		{"Basic", "CREATE TABLE test", query.CreateTableStmt{TableName: "test"}, false},
		{"If not exists", "CREATE TABLE IF NOT EXISTS test", query.CreateTableStmt{TableName: "test", IfNotExists: true}, false},
		{"Path only", "CREATE TABLE test(a)", query.CreateTableStmt{}, true},
		{"With primary key", "CREATE TABLE test(foo INTEGER PRIMARY KEY)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsPrimaryKey: true},
					},
				},
			}, false},
		{"With primary key twice", "CREATE TABLE test(foo PRIMARY KEY PRIMARY KEY)",
			query.CreateTableStmt{}, true},
		{"With type", "CREATE TABLE test(foo INTEGER)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue},
					},
				},
			}, false},
		{"With not null", "CREATE TABLE test(foo NOT NULL)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), IsNotNull: true},
					},
				},
			}, false},
		{"With default", "CREATE TABLE test(foo DEFAULT \"10\")",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), DefaultValue: document.NewTextValue("10")},
					},
				},
			}, false},
		{"With unique", "CREATE TABLE test(foo UNIQUE)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), IsUnique: true},
					},
				},
			}, false},
		{"With default twice", "CREATE TABLE test(foo DEFAULT 10 DEFAULT 10)",
			query.CreateTableStmt{}, true},
		{"With not null twice", "CREATE TABLE test(foo NOT NULL NOT NULL)",
			query.CreateTableStmt{}, true},
		{"With unique twice", "CREATE TABLE test(foo UNIQUE UNIQUE)",
			query.CreateTableStmt{}, true},
		{"With type and not null", "CREATE TABLE test(foo INTEGER NOT NULL)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsNotNull: true},
					},
				},
			}, false},
		{"With not null and primary key", "CREATE TABLE test(foo INTEGER NOT NULL PRIMARY KEY)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsPrimaryKey: true, IsNotNull: true},
					},
				},
			}, false},
		{"With primary key and not null", "CREATE TABLE test(foo INTEGER PRIMARY KEY NOT NULL)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsPrimaryKey: true, IsNotNull: true},
					},
				},
			}, false},
		{"With multiple constraints", "CREATE TABLE test(foo INTEGER PRIMARY KEY, bar INTEGER NOT NULL, baz[4][1].bat TEXT)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsPrimaryKey: true},
						{Path: document.Path(parsePath(t, "bar")), Type: document.IntegerValue, IsNotNull: true},
						{Path: document.Path(parsePath(t, "baz[4][1].bat")), Type: document.TextValue},
					},
				},
			}, false},
		{"With table constraints / PK on defined field", "CREATE TABLE test(foo INTEGER, bar NOT NULL, PRIMARY KEY (foo))",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue, IsPrimaryKey: true},
						{Path: document.Path(parsePath(t, "bar")), IsNotNull: true},
					},
				},
			}, false},
		{"With table constraints / PK on undefined field", "CREATE TABLE test(foo INTEGER, PRIMARY KEY (bar))",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "foo")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "bar")), IsPrimaryKey: true},
					},
				},
			}, false},
		{"With table constraints / field constraint after table constraint", "CREATE TABLE test(PRIMARY KEY (bar), foo INTEGER)", nil, true},
		{"With table constraints / duplicate pk", "CREATE TABLE test(foo INTEGER PRIMARY KEY, PRIMARY KEY (bar))", nil, true},
		{"With table constraints / duplicate pk on same path", "CREATE TABLE test(foo INTEGER PRIMARY KEY, PRIMARY KEY (foo))", nil, true},
		{"With multiple primary keys", "CREATE TABLE test(foo PRIMARY KEY, bar PRIMARY KEY)",
			query.CreateTableStmt{}, true},
		{"With all supported fixed size data types",
			"CREATE TABLE test(d double, b bool)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "d")), Type: document.DoubleValue},
						{Path: document.Path(parsePath(t, "b")), Type: document.BoolValue},
					},
				},
			}, false},
		{"With all supported variable size data types",
			"CREATE TABLE test(i integer, b blob, byt bytes, t text, a array, d document)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "i")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "b")), Type: document.BlobValue},
						{Path: document.Path(parsePath(t, "byt")), Type: document.BlobValue},
						{Path: document.Path(parsePath(t, "t")), Type: document.TextValue},
						{Path: document.Path(parsePath(t, "a")), Type: document.ArrayValue},
						{Path: document.Path(parsePath(t, "d")), Type: document.DocumentValue},
					},
				},
			}, false},
		{"With integer aliases types",
			"CREATE TABLE test(i int, ii int2, ei int8, m mediumint, s smallint, b bigint, t tinyint)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "i")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "ii")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "ei")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "m")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "s")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "b")), Type: document.IntegerValue},
						{Path: document.Path(parsePath(t, "t")), Type: document.IntegerValue},
					},
				},
			}, false},
		{"With double aliases types",
			"CREATE TABLE test(dp DOUBLE PRECISION, r real, d double)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "dp")), Type: document.DoubleValue},
						{Path: document.Path(parsePath(t, "r")), Type: document.DoubleValue},
						{Path: document.Path(parsePath(t, "d")), Type: document.DoubleValue},
					},
				},
			}, false},
		{"With text aliases types",
			"CREATE TABLE test(v VARCHAR(255), c CHARACTER(64), t TEXT)",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "v")), Type: document.TextValue},
						{Path: document.Path(parsePath(t, "c")), Type: document.TextValue},
						{Path: document.Path(parsePath(t, "t")), Type: document.TextValue},
					},
				},
			}, false},
		{"With errored text aliases types",
			"CREATE TABLE test(v VARCHAR(1 IN [1, 2, 3] AND foo > 4) )",
			query.CreateTableStmt{
				TableName: "test",
				Info: database.TableInfo{
					FieldConstraints: []*database.FieldConstraint{
						{Path: document.Path(parsePath(t, "v")), Type: document.TextValue},
					},
				},
			}, true},
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

func TestParserCreateIndex(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected query.Statement
		errored  bool
	}{
		{"Basic", "CREATE INDEX idx ON test (foo)", query.CreateIndexStmt{IndexName: "idx", TableName: "test", Path: document.Path(parsePath(t, "foo"))}, false},
		{"If not exists", "CREATE INDEX IF NOT EXISTS idx ON test (foo.bar[1])", query.CreateIndexStmt{IndexName: "idx", TableName: "test", Path: document.Path(parsePath(t, "foo.bar[1]")), IfNotExists: true}, false},
		{"Unique", "CREATE UNIQUE INDEX IF NOT EXISTS idx ON test (foo[3].baz)", query.CreateIndexStmt{IndexName: "idx", TableName: "test", Path: document.Path(parsePath(t, "foo[3].baz")), IfNotExists: true, Unique: true}, false},
		{"No name", "CREATE UNIQUE INDEX ON test (foo[3].baz)", query.CreateIndexStmt{TableName: "test", Path: document.Path(parsePath(t, "foo[3].baz")), Unique: true}, false},
		{"No name with IF NOT EXISTS", "CREATE UNIQUE INDEX IF NOT EXISTS ON test (foo[3].baz)", nil, true},
		{"No fields", "CREATE INDEX idx ON test", nil, true},
		{"More than 1 path", "CREATE INDEX idx ON test (foo, bar)", nil, true},
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
