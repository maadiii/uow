// Command example runs a unit of work over database/sql with the pgx driver.
//
// Usage:
//
//	DATABASE_URL=postgres://uow:uow@localhost:5432/uow?sslmode=disable go run ./examples
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	"github.com/maadiii/uow"
)

const createTableSQL = `CREATE TABLE IF NOT EXISTS users (
	id   serial PRIMARY KEY,
	name text UNIQUE NOT NULL
)`

type usersRepo struct {
	tx *sql.Tx
}

func (r usersRepo) Create(ctx context.Context, name string) error {
	_, err := r.tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)

	return err
}

type repos struct {
	Users usersRepo
}

func newRepos(tx *sql.Tx) repos {
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

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := run(context.Background(), db); err != nil {
		return err
	}

	log.Print("committed Alice and Bob")

	return nil
}

func run(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return err
	}

	// Everything inside the callback shares one transaction: if the second
	// insert fails, the first one is rolled back too.
	return uow.NewSql(db, newRepos).Do(ctx, func(ctx context.Context, repos repos) error {
		if err := repos.Users.Create(ctx, "Alice"); err != nil {
			return err
		}

		return repos.Users.Create(ctx, "Bob")
	})
}
