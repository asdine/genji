// +build wasm

package genji

import (
	"context"

	"github.com/genjidb/genji/database"
	"github.com/genjidb/genji/document/encoding/custom"
	"github.com/genjidb/genji/engine"
)

// New initializes the DB using the given engine.
func New(ctx context.Context, ng engine.Engine) (*DB, error) {
	return newDatabase(ctx, ng, database.Options{Codec: custom.NewCodec()})
}
