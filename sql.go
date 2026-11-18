package uow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// RF -> Repository
func NewSql[R any](db *sql.DB, constructor func(tx *sql.Tx) R) UoW[R] {
	return &uowsql[R]{db, constructor}
}

// R -> Repository
type uowsql[R any] struct {
	db          *sql.DB
	constructor func(tx *sql.Tx) R
}

func (u *uowsql[R]) Do(ctx context.Context, fn func(context.Context, R) error, opts ...*TxOptions) error {
	txOpts := ConvertSqlTxOptions(opts...)

	tx, err := u.db.BeginTx(ctx, txOpts)
	if err != nil {
		return err
	}

	if err := fn(ctx, u.constructor(tx)); err != nil {
		rollbackError := tx.Rollback()
		if rollbackError != nil {
			return errors.Join(err, rollbackError)
		}

		return err
	}

	return tx.Commit()
}

func (u *uowsql[R]) Begin(ctx context.Context, opts ...*TxOptions) (c context.Context, r R, err error) { //nolint
	txOpts := ConvertSqlTxOptions(opts...)

	tx, err := u.db.BeginTx(ctx, txOpts)
	if err != nil {
		return
	}

	return context.WithValue(ctx, CtxTxKey{}, tx), u.constructor(tx), nil
}

func (u *uowsql[R]) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*sql.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Commit()
}

func (u *uowsql[T]) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*sql.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Rollback()
}

func (u *uowsql[T]) SavePoint(ctx context.Context, name string) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*sql.Tx)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	_, err := tx.Exec(fmt.Sprintf("%s %s", savepoint, name))

	return err
}

func ConvertSqlTxOptions(opts ...*TxOptions) *sql.TxOptions {
	if len(opts) == 0 || opts[0] == nil {
		return nil
	}

	return &sql.TxOptions{
		Isolation: sql.IsolationLevel(opts[0].Isolation),
		ReadOnly:  opts[0].ReadOnly,
	}
}
