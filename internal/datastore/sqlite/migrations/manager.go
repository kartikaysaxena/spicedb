package migrations

import (
   "database/sql"

   "github.com/authzed/spicedb/pkg/migrate"
)

// Manager is the singleton migration manager for SQLite.
var Manager = migrate.NewManager[*SQLiteDriver, *sql.DB, *sql.Tx]()