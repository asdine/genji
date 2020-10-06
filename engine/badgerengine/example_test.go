package badgerengine_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/engine/badgerengine"
)

func Example() {
	ctx := context.Background()

	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	ng, err := badgerengine.NewEngine(badger.DefaultOptions(filepath.Join(dir, "badger")))
	if err != nil {
		log.Fatal(err)
	}

	db, err := genji.New(ctx, ng)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
