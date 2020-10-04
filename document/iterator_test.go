package document_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/stretchr/testify/require"
)

func ExampleStream_First() {
	db, err := genji.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	err = db.Exec(ctx, "CREATE TABLE user")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(ctx, "INSERT INTO user (id, name, age) VALUES (?, ?, ?)", 10, "foo", 15)
	if err != nil {
		log.Fatal(err)
	}

	result, err := db.Query(ctx, "SELECT id, name, age FROM user WHERE name = ?", "foo")
	if err != nil {
		panic(err)
	}
	defer result.Close()

	d, err := result.First()
	if err != nil {
		panic(err)
	}

	var id uint64
	var name string
	var age uint8

	err = document.Scan(d, &id, &name, &age)
	if err != nil {
		panic(err)
	}

	fmt.Println(id, name, age)

	// Output:
	// 10 foo 15
}

func ExampleStream_Iterate() {
	type User struct {
		ID      int64
		Name    string
		Age     uint32
		Address struct {
			City    string
			ZipCode string
		}
	}

	db, err := genji.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	err = db.Exec(ctx, "CREATE TABLE IF NOT EXISTS user")
	if err != nil {
		log.Fatal(err)
	}

	for i := 1; i <= 10; i++ {
		err = db.Exec(ctx, "INSERT INTO user VALUES ?", &User{
			ID:   int64(i),
			Name: fmt.Sprintf("foo%d", i),
			Age:  uint32(i * 10),
			Address: struct {
				City    string
				ZipCode string
			}{
				City:    "Lyon",
				ZipCode: fmt.Sprintf("69%03d", i),
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	result, err := db.Query(ctx, `SELECT id, name, age, address FROM user WHERE age >= 18`)
	if err != nil {
		panic(err)
	}
	defer result.Close()

	err = result.Iterate(func(d document.Document) error {
		// Scan into a struct
		var u User
		err = document.StructScan(d, &u)
		if err != nil {
			return err
		}

		fmt.Println(u)

		// Or scan individual variables
		// Types of variables don't have to exactly match with the types stored
		var id uint64
		var name []byte
		var age uint8
		var address map[string]string

		err = document.Scan(d, &id, &name, &age, &address)
		if err != nil {
			return err
		}

		fmt.Println(id, string(name), age, address)
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Output: {2 foo2 20 {Lyon 69002}}
	// 2 foo2 20 map[city:Lyon zipcode:69002]
	// {3 foo3 30 {Lyon 69003}}
	// 3 foo3 30 map[city:Lyon zipcode:69003]
	// {4 foo4 40 {Lyon 69004}}
	// 4 foo4 40 map[city:Lyon zipcode:69004]
	// {5 foo5 50 {Lyon 69005}}
	// 5 foo5 50 map[city:Lyon zipcode:69005]
	// {6 foo6 60 {Lyon 69006}}
	// 6 foo6 60 map[city:Lyon zipcode:69006]
	// {7 foo7 70 {Lyon 69007}}
	// 7 foo7 70 map[city:Lyon zipcode:69007]
	// {8 foo8 80 {Lyon 69008}}
	// 8 foo8 80 map[city:Lyon zipcode:69008]
	// {9 foo9 90 {Lyon 69009}}
	// 9 foo9 90 map[city:Lyon zipcode:69009]
	// {10 foo10 100 {Lyon 69010}}
	// 10 foo10 100 map[city:Lyon zipcode:69010]
}

func TestIteratorToJSONArray(t *testing.T) {
	var docs []document.Document
	for i := 0; i < 3; i++ {
		fb := document.NewFieldBuffer()
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"a": %d}`, i)), fb)
		require.NoError(t, err)
		docs = append(docs, fb)
	}

	it := document.NewIterator(docs...)
	var buf bytes.Buffer
	err := document.IteratorToJSONArray(&buf, it)
	require.NoError(t, err)
	require.Equal(t, `[{"a": 0}, {"a": 1}, {"a": 2}]`, buf.String())
}
