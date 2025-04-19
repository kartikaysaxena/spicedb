package sqlite

import (
   "context"
   "fmt"
   "database/sql"
   _ "github.com/mattn/go-sqlite3"

   datastoreinternal "github.com/authzed/spicedb/internal/datastore"
   "github.com/authzed/spicedb/internal/datastore/common"
   "github.com/authzed/spicedb/internal/datastore/revisions"
   log "github.com/authzed/spicedb/internal/logging"
   "github.com/authzed/spicedb/pkg/datastore"
)

// Engine is the identifier for the SQLite datastore engine.
const Engine = "sqlite"

func init() {
   // Register SQLite engine.
   datastore.Engines = append(datastore.Engines, Engine)
}

// Option configures SQLite datastore behavior.
// TODO: add options for pragmas, caching, WAL, etc.
type Option func(*sqliteOptions)

type sqliteOptions struct {
   // placeholder for future configuration
}

// generateConfig applies options to default settings.
func generateConfig(opts []Option) (sqliteOptions, error) {
   cfg := sqliteOptions{}
   for _, opt := range opts {
       opt(&cfg)
   }
   return cfg, nil
}

// NewSQLiteDatastore initializes a SQLite-based SpiceDB datastore.
// TODO: implement full datastore support using database/sql and common query code.
func NewSQLiteDatastore(ctx context.Context, uri string, options ...Option) (datastore.Datastore, error) {
   // TODO: open connection, configure pooling, apply migrations
   return nil, fmt.Errorf("sqlite datastore not implemented")
}

// NewReadOnlySQLiteDatastore initializes a read-only SQLite datastore.
// TODO: implement read-only support.
func NewReadOnlySQLiteDatastore(ctx context.Context, uri string, index uint32, options ...Option) (datastore.ReadOnlyDatastore, error) {
   // TODO: open connection in read-only mode
   return nil, fmt.Errorf("sqlite readonly datastore not implemented")
}