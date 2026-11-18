package uow_test

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PgxExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PgxRepository struct {
	pool PgxExecer
}

func (p PgxRepository) Users() PgxUserRepo {
	return PgxUserRepo{p.pool}
}

type PgxUserRepo struct {
	pool PgxExecer
}

func (p PgxUserRepo) Create(name string) error {
	_, err := p.pool.Exec(context.Background(), "INSERT INTO users (name) VALUES ($1)", name)

	return err
}

func (p PgxUserRepo) Get(id int) (string, error) {
	row := p.pool.QueryRow(context.Background(), "SELECT name FROM users WHERE id = $1", id)
	var name string
	if err := row.Scan(&name); err != nil {
		return name, err
	}

	return name, nil
}
