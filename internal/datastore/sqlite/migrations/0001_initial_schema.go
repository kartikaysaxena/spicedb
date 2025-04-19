package migrations

import (
   "context"
   "fmt"
)

// 0001_initial_schema is the first migration for SQLite.
// TODO: copy and adapt the initial schema DDL statements.
func Up_0001_initial_schema(ctx context.Context, wrapper interface{}) error {
   // wrapper may be *sql.DB or custom type
   return fmt.Errorf("sqlite initial schema migration not implemented")
}

// TODO: Add Down_0001_initial_schema if down migrations are supported.