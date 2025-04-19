package migrations

import (
   "context"
   "database/sql"
   "fmt"
   _ "github.com/mattn/go-sqlite3"

   "github.com/authzed/spicedb/pkg/migrate"
)

// SQLiteDriver is an implementation of migrate.Driver for SQLite.
// TODO: implement migration logic using SQLite PRAGMAs and version table.
type SQLiteDriver struct {
   db *sql.DB
}

// NewSQLiteDriver initializes a new migration driver for SQLite using the provided DSN.
func NewSQLiteDriver(dsn string) (*SQLiteDriver, error) {
   // TODO: open database connection and configure transactions
   return nil, fmt.Errorf("sqlite migration driver not implemented")
}

// Conn returns the underlying DB connection for migration execution.
func (d *SQLiteDriver) Conn() *sql.DB {
   return d.db
}

// RunTx executes a migration within a transaction.
func (d *SQLiteDriver) RunTx(ctx context.Context, f migrate.TxMigrationFunc[*sql.Tx]) error {
   // TODO: begin tx, call f, commit/rollback
   return fmt.Errorf("sqlite migration RunTx not implemented")
}

// Version returns the current migration version applied to the database.
func (d *SQLiteDriver) Version(ctx context.Context) (string, error) {
   // TODO: query version table
   return "", fmt.Errorf("sqlite migration Version not implemented")
}

// WriteVersion records a migration version change in the database.
func (d *SQLiteDriver) WriteVersion(ctx context.Context, tx *sql.Tx, version, replaced string) error {
   // TODO: update version table
   return fmt.Errorf("sqlite migration WriteVersion not implemented")
}

// Close closes the underlying database connection.
func (d *SQLiteDriver) Close(ctx context.Context) error {
   if d.db != nil {
       return d.db.Close()
   }
   return nil
}

var _ migrate.Driver[*sql.DB, *sql.Tx] = &SQLiteDriver{}