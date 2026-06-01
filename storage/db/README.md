# storage/db

Opens a `*sql.DB` connection pool backed by PostgreSQL and runs SQL migrations.

## Usage

```go
import (
    "context-os/migrations"
    "context-os/storage/db"
)

sqlDB, err := db.Open(migrations.Files)
```

`Open` reads `DATABASE_URL` from the environment, falling back to the local-dev
default (`postgres://contextos:contextos@localhost:5432/contextos?sslmode=disable`),
pings the server, and runs any pending migrations from the provided `fs.FS`.

## Auto-migration

Migrations are tracked in a `schema_migrations` table. Each `.sql` file is
applied exactly once, in lexicographic order.  The caller passes the `fs.FS`
(typically the `migrations.Files` embed from `context-os/migrations`) so that
the embed path is resolved next to where the SQL files live in the source tree.

## Graceful degradation

The API binary logs a warning and skips workspace routes if Postgres is
unreachable at startup.  All other routes continue to work normally.
