<!--
Adding first‐class SQLite support is a large undertaking—essentially a fifth full SQL backend,
with its own migrations, SQL quirks, watch API, and matching integration tests.
This document captures the high-level roadmap and initial steps for implementing SQLite support.
-->
# SQLite Support Roadmap

Adding first‑class SQLite support is a large undertaking—essentially a fifth full SQL backend,
with its own migrations, SQL quirks, watch API, and matching integration tests.
Here’s the high‑level roadmap:

1. Register the new engine
   - Add “sqlite” to the engine constants in `pkg/cmd/datastore/datastore.go`
   - Wire up `BuilderForEngine` to point at `newSQLiteDatastore`
2. Build the SQLite driver layer
   - Create `internal/datastore/sqlite` package
   - Implement `NewSQLiteDatastore` and `NewReadOnlySQLiteDatastore` using `github.com/mattn/go-sqlite3`
   - Wire in our common query code (`internal/datastore/common`) wherever possible
3. Migrations
   - Copy and adapt the initial schema migration (`0001_initial_schema.go`) into `internal/datastore/sqlite/migrations`, tweaking DDL for SQLite (e.g. AUTOINCREMENT, foreign-key pragmas)
   - Implement `driver.go` in that package to register migrations with our manager (mirroring the patterns in CRDB and Postgres)
4. Full API support
   - Port transaction, GC, caveats, watch, and revision support from Postgres/MySQL to SQLite, using SQLite’s snapshot and WAL features where possible
   - Ensure our gRPC services (Check, Read, Watch, Write, etc.) operate correctly against the SQLite backend
5. Testing
   - Unit tests in `internal/datastore/sqlite` to exercise all operations (create, update, delete, watch, snapshot reads)
   - Extend `cmd/spicedb` integration tests (`migrate`, `serve`, `restgateway`, `schemawatch`, etc.) to run with `--datastore-engine=sqlite` against a temporary file or `:memory:`
   - Hook SQLite into the `internal/testserver` harness so e2e and steelthread tests can target SQLite

Because this is quite a bit of work, we will start by scaffolding steps 1 & 2 in code:
1. Engine constant & builder in `pkg/cmd/datastore`
2. Empty `internal/datastore/sqlite` package for implementation

---
_Documented on: DATE_