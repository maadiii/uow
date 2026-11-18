// Command pgx-example runs a unit of work over a pgx connection pool.
//
// Usage:
//
//	DATABASE_URL=postgres://uow:uow@localhost:5432/uow?sslmode=disable go run ./examples/pgx
package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maadiii/uow"
)

const createTableSQL = `CREATE TABLE IF NOT EXISTS users (
	id   serial PRIMARY KEY,
	name text UNIQUE NOT NULL
)`

type usersRepo struct {
	tx pgx.Tx
}

func (r usersRepo) Create(ctx context.Context, name string) error {
	_, err := r.tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", name)

	return err
}

type repos struct {
	Users usersRepo
}

func newRepos(tx pgx.Tx) repos {
	return repos{Users: usersRepo{tx: tx}}
}

func main() {
	if err := runMain(); err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := run(ctx, pool); err != nil {
		return err
	}

	log.Print("committed Alice and Bob")

	return nil
}

func run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, createTableSQL); err != nil {
		return err
	}

	// Everything inside the callback shares one transaction: if the second
	// insert fails, the first one is rolled back too.
	return uow.NewPgx(pool, newRepos).Do(ctx, func(ctx context.Context, repos repos) error {
		if err := repos.Users.Create(ctx, "Alice"); err != nil {
			return err
		}

		return repos.Users.Create(ctx, "Bob")
	})
}
