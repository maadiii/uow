package uow

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrTransactionNotFound = errors.New("no transaction found in context")
	savepoint              = "SAVEPOINT"
)

// RF -> Repository
func NewPgx[R any](pool PgxPool, constructor func(pgx.Tx) R) UoW[R] {
	return &uowPgx[R]{pool, constructor}
}

type uowPgx[R any] struct {
	pool        PgxPool
	constructor func(tx pgx.Tx) R
}

func (u *uowPgx[RF]) Do(ctx context.Context, fn func(context.Context, RF) error, opts ...*TxOptions) error {
	txOpts := ConvertPgxTxOptions(opts...)

	tx, err := u.pool.BeginTx(ctx, txOpts)
	if err != nil {
		return err
	}

	if err := fn(ctx, u.constructor(tx)); err != nil {
		rollBackError := tx.Rollback(ctx)
		if rollBackError != nil {
			return errors.Join(err, rollBackError)
		}

		return err
	}

	return tx.Commit(ctx)
}

func (u *uowPgx[RF]) Begin(ctx context.Context, opts ...*TxOptions) (c context.Context, rf RF, err error) { //nolint
	txOpts := ConvertPgxTxOptions(opts...)

	tx, err := u.pool.BeginTx(ctx, txOpts)
	if err != nil {
		return
	}

	ctx = context.WithValue(ctx, CtxTxKey{}, tx)

	return ctx, u.constructor(tx), nil
}

func (u *uowPgx[RepoFactory]) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(pgx.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Commit(ctx)
}

func (u *uowPgx[RepoFactory]) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(pgx.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Rollback(ctx)
}

func (u *uowPgx[T]) SavePoint(ctx context.Context, name string) error {
	tx, ok := ctx.Value(CtxTxKey{}).(pgx.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	_, err := tx.Exec(ctx, fmt.Sprintf("%s %s", savepoint, name))

	return err
}

func ConvertPgxTxOptions(opts ...*TxOptions) pgx.TxOptions {
	var pgxOpts pgx.TxOptions

	if len(opts) == 0 || opts[0] == nil {
		pgxOpts.AccessMode = pgx.ReadWrite

		return pgxOpts
	}

	switch opts[0].Isolation { //nolint: exhaustive
	case LevelRepeatableRead:
		pgxOpts.IsoLevel = pgx.RepeatableRead
	case LevelSerializable:
		pgxOpts.IsoLevel = pgx.Serializable
	default:
		pgxOpts.IsoLevel = pgx.ReadCommitted
	}

	pgxOpts.AccessMode = ChooseAccessMode(opts[0])

	return pgxOpts
}

func ChooseAccessMode(opts *TxOptions) pgx.TxAccessMode {
	if opts == nil {
		return pgx.ReadWrite
	}

	if opts.ReadOnly {
		return pgx.ReadOnly
	}

	return pgx.ReadWrite
}

type (
	CtxTxKey struct{}
	PgxPool  interface {
		BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	}
)
