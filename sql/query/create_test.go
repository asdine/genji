package query_test

import (
	"testing"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/sql/parser"
	"github.com/stretchr/testify/require"
)

func parsePath(t testing.TB, str string) document.Path {
	vp, err := parser.ParsePath(str)
	require.NoError(t, err)
	return vp
}

func TestCreateTable(t *testing.T) {
	tests := []struct {
		name  string
		query string
		fails bool
	}{
		{"Basic", `CREATE TABLE test`, false},
		{"Exists", "CREATE TABLE test;CREATE TABLE test", true},
		{"If not exists", "CREATE TABLE IF NOT EXISTS test", false},
		{"If not exists, twice", "CREATE TABLE IF NOT EXISTS test;CREATE TABLE IF NOT EXISTS test", false},
		{"With primary key", "CREATE TABLE test(foo TEXT PRIMARY KEY)", false},
		{"With field constraints", "CREATE TABLE test(foo.a[1][2] TEXT primary key, bar[4][0].bat INTEGER not null, baz not null)", false},
		{"With no constraints", "CREATE TABLE test(a, b)", false},
		{"With coherent constraint(common)", "CREATE TABLE test(a DOCUMENT, a.b ARRAY, a.b[0] TEXT);", false},
		{"With coherent constraint(any)", "CREATE TABLE test(a, a.b[0] TEXT);", false},
		{"With coherent constraint(document)", "CREATE TABLE test(a DOCUMENT, a.b TEXT);", false},
		{"With coherent constraint(array)", "CREATE TABLE test(a ARRAY, a[0] TEXT);", false},
		{"With incoherent constraint(common)", "CREATE TABLE test(a INTEGER, a.b[0] TEXT);", true},
		{"With incoherent constraint(common)", "CREATE TABLE test(a DOCUMENT, a.b[0] TEXT, a.b.c TEXT);", true},
		{"With incoherent constraint(common)", "CREATE TABLE test(a DOCUMENT, a.b.c TEXT, a.b[0] TEXT);", true},
		{"With incoherent constraint(document)", "CREATE TABLE test(a INTEGER, a.b TEXT);", true},
		{"With incoherent constraint(array)", "CREATE TABLE test(a INTEGER, a[0] TEXT);", true},
		{"With duplicate constraints", "CREATE TABLE test(a INTEGER, a TEXT);", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := genji.Open(":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = db.Exec(test.query)
			if test.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			err = db.View(func(tx *genji.Tx) error {
				_, err := tx.GetTable("test")
				return err
			})
			require.NoError(t, err)
		})
	}

	t.Run("constraints", func(t *testing.T) {
		db, err := genji.Open(":memory:")
		require.NoError(t, err)
		defer db.Close()

		t.Run("with fixed size data types", func(t *testing.T) {
			err = db.Exec(`CREATE TABLE test(d double, b bool)`)
			require.NoError(t, err)

			err = db.View(func(tx *genji.Tx) error {
				tb, err := tx.GetTable("test")
				if err != nil {
					return err
				}

				info, err := tb.Info()
				if err != nil {
					return err
				}

				require.Equal(t, database.FieldConstraints{
					{Path: parsePath(t, "d"), Type: document.DoubleValue},
					{Path: parsePath(t, "b"), Type: document.BoolValue},
				}, info.FieldConstraints)
				return nil
			})
			require.NoError(t, err)

		})

		t.Run("with variable size data types", func(t *testing.T) {
			err = db.Exec(`
				CREATE TABLE test1(
					foo.bar[1].hello bytes PRIMARY KEY, foo.a[1][2] TEXT NOT NULL, bar[4][0].bat integer, b blob, t text, a array, d document
				)
			`)
			require.NoError(t, err)

			err = db.View(func(tx *genji.Tx) error {
				tb, err := tx.GetTable("test1")
				if err != nil {
					return err
				}
				info, err := tb.Info()
				if err != nil {
					return err
				}

				require.Equal(t, database.FieldConstraints{
					{Path: parsePath(t, "foo.bar[1].hello"), Type: document.BlobValue, IsPrimaryKey: true},
					{Path: parsePath(t, "foo.a[1][2]"), Type: document.TextValue, IsNotNull: true},
					{Path: parsePath(t, "bar[4][0].bat"), Type: document.IntegerValue},
					{Path: parsePath(t, "b"), Type: document.BlobValue},
					{Path: parsePath(t, "t"), Type: document.TextValue},
					{Path: parsePath(t, "a"), Type: document.ArrayValue},
					{Path: parsePath(t, "d"), Type: document.DocumentValue},
				}, info.FieldConstraints)
				return nil
			})
			require.NoError(t, err)

		})

		t.Run("with variable aliases data types", func(t *testing.T) {
			err = db.Exec(`
				CREATE TABLE test2(
					foo.bar[1].hello bytes PRIMARY KEY, foo.a[1][2] VARCHAR(255) NOT NULL, bar[4][0].bat tinyint,
				 	dp double precision, r real, b bigint, m mediumint, eight int8, ii int2, c character(64)
				)
			`)
			require.NoError(t, err)

			err = db.View(func(tx *genji.Tx) error {
				tb, err := tx.GetTable("test2")
				if err != nil {
					return err
				}
				info, err := tb.Info()
				if err != nil {
					return err
				}

				require.Equal(t, database.FieldConstraints{
					{Path: parsePath(t, "foo.bar[1].hello"), Type: document.BlobValue, IsPrimaryKey: true},
					{Path: parsePath(t, "foo.a[1][2]"), Type: document.TextValue, IsNotNull: true},
					{Path: parsePath(t, "bar[4][0].bat"), Type: document.IntegerValue},
					{Path: parsePath(t, "dp"), Type: document.DoubleValue},
					{Path: parsePath(t, "r"), Type: document.DoubleValue},
					{Path: parsePath(t, "b"), Type: document.IntegerValue},
					{Path: parsePath(t, "m"), Type: document.IntegerValue},
					{Path: parsePath(t, "eight"), Type: document.IntegerValue},
					{Path: parsePath(t, "ii"), Type: document.IntegerValue},
					{Path: parsePath(t, "c"), Type: document.TextValue},
				}, info.FieldConstraints)
				return nil
			})
			require.NoError(t, err)
		})

		t.Run("default values", func(t *testing.T) {
			tests := []struct {
				name        string
				query       string
				constraints database.FieldConstraints
				fails       bool
			}{
				{"With default, no type and integer default", "CREATE TABLE test(foo DEFAULT 10)", database.FieldConstraints{{Path: parsePath(t, "foo"), DefaultValue: document.NewDoubleValue(10)}}, false},
				{"With default, double type and integer default", "CREATE TABLE test(foo DOUBLE DEFAULT 10)", database.FieldConstraints{{Path: parsePath(t, "foo"), Type: document.DoubleValue, DefaultValue: document.NewDoubleValue(10)}}, false},
				{"With default, some type and compatible default", "CREATE TABLE test(foo BOOL DEFAULT 10)", database.FieldConstraints{{Path: parsePath(t, "foo"), Type: document.BoolValue, DefaultValue: document.NewBoolValue(true)}}, false},
				{"With default, some type and incompatible default", "CREATE TABLE test(foo BOOL DEFAULT 10.5)", nil, true},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					db, err := genji.Open(":memory:")
					require.NoError(t, err)
					defer db.Close()

					err = db.Exec(test.query)
					if test.fails {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)

					err = db.View(func(tx *genji.Tx) error {
						tb, err := tx.GetTable("test")
						info, err := tb.Info()
						if err != nil {
							return err
						}

						require.Equal(t, test.constraints, info.FieldConstraints)
						return err
					})
					require.NoError(t, err)
				})
			}
		})
	})
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		name  string
		query string
		fails bool
	}{
		{"Basic", "CREATE INDEX idx ON test (foo)", false},
		{"If not exists", "CREATE INDEX IF NOT EXISTS idx ON test (foo.bar)", false},
		{"Unique", "CREATE UNIQUE INDEX IF NOT EXISTS idx ON test (foo[1])", false},
		{"No fields", "CREATE INDEX idx ON test", true},
		{"More than 1 field", "CREATE INDEX idx ON test (foo, bar)", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := genji.Open(":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = db.Exec("CREATE TABLE test")
			require.NoError(t, err)

			err = db.Exec(test.query)
			if test.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
