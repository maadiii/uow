# uow

Small unit-of-work helpers for Go that run a callback inside a database
transaction and hand it a repository built on that transaction. Available for
`database/sql`, [GORM](https://gorm.io) and [pgx](https://github.com/jackc/pgx).

## Install

```sh
go get github.com/maadiii/uow
```

## Usage

Each backend takes its native handle plus a constructor that builds a
repository from a transaction. Repositories are yours; the library only decides
when the transaction begins, commits or rolls back.

```go
type Repos struct {
	Users *UserRepo
}

func NewRepos(tx *sql.Tx) Repos {
	return Repos{Users: &UserRepo{tx: tx}}
}

uow := uow.NewSql(db, NewRepos)

err := uow.Do(ctx, func(ctx context.Context, repos Repos) error {
	return repos.Users.Create(ctx, "Alice")
})
```

The same shape works for the other backends:

```go
uow := uow.NewGorm(gormDB, func(tx *gorm.DB) Repos { return NewRepos(tx) })
uow := uow.NewPgx(pool, func(tx pgx.Tx) Repos { return NewRepos(tx) })
```

`NewPgx` only needs something that can begin a transaction, so any `*pgxpool.Pool`
satisfies it.

Runnable versions of the example live in [`examples/main.go`](examples/main.go) (SQL)
and [`examples/pgx/main.go`](examples/pgx/main.go) (pgx pool); point
`DATABASE_URL` at a Postgres database and run `go run ./examples` or
`go run ./examples/pgx`.

### Transaction semantics

`Do(ctx, fn, opts...)`:

- commits when `fn` returns `nil`;
- rolls back when `fn` returns an error, joining a rollback failure with the
  original error instead of hiding either;
- rolls back and **re-panics** when `fn` panics, so the transaction is never
  left open while the caller still sees the original failure;
- returns any begin or commit error unchanged.

### Manual control

`Begin` returns a context carrying the transaction and the repository built on
it. `Commit`, `Rollback` and `SavePoint` take that context back:

```go
ctx, repos, err := uow.Begin(ctx)
if err != nil {
	return err
}

if err := repos.Users.Create(ctx, "Alice"); err != nil {
	_ = uow.Rollback(ctx)
	return err
}

if err := uow.SavePoint(ctx, "after-alice"); err != nil {
	_ = uow.Rollback(ctx)
	return err
}

return uow.Commit(ctx)
```

Calling `Commit`, `Rollback` or `SavePoint` with a context that carries no
transaction returns `uow.ErrTransactionNotFound`.

Savepoint names are quoted as SQL identifiers (`SAVEPOINT "after-alice"`), so a
name containing quotes, semicolons or comments cannot change the statement.

### Transaction options

Both `Do` and `Begin` accept an optional `*uow.TxOptions`:

```go
err := uow.Do(ctx, fn, &uow.TxOptions{
	Isolation: uow.LevelSerializable,
	ReadOnly:  true,
})
```

`ReadOnly` becomes `TxOptions.ReadOnly` for `database/sql` and GORM and
`AccessMode = pgx.ReadOnly` for pgx; a write attempted inside a read-only
transaction is rejected by the server.

Isolation levels are named after `database/sql` and passed through unchanged
there (a driver may reject a level it does not support). Postgres has fewer
levels than the list, so pgx maps them as follows:

| `uow` level          | pgx level          |
| -------------------- | ------------------ |
| `LevelDefault`       | `ReadCommitted`    |
| `LevelReadUncommitted` | `ReadCommitted`  |
| `LevelReadCommitted` | `ReadCommitted`    |
| `LevelWriteCommitted` | `ReadCommitted`   |
| `LevelRepeatableRead` | `RepeatableRead`  |
| `LevelSnapshot`      | `ReadCommitted`    |
| `LevelSerializable`  | `Serializable`     |
| `LevelLinearizable`  | `ReadCommitted`    |

Nil or absent options leave the behaviour to the driver.

## Tests

```sh
go test ./...
```

Besides the mock-based unit tests, the suite runs real-database integration
tests against a `postgres:16-alpine` container started through testcontainers.
Docker must be running; without it those tests call
`testcontainers.SkipIfProviderIsNotHealthy` and skip.

Every backend has a mock suite (`sql_test.go`, `gorm_test.go`, `pgx_test.go`) and
an integration suite (`sql_integration_test.go`, `gorm_integration_test.go`,
`pgx_integration_test.go`). The integration suites run the same flows against
the container, so isolation levels, read-only mode and savepoint quoting are
asserted by Postgres itself rather than only by the mock's expectations. There
are no build tags: `go test ./...` runs everything.

For a coverage report:

```sh
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

The library package is at 100% statement coverage; the `examples` commands are
run by hand and are not covered.
