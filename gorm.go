package uow

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

func NewGorm[R any](db *gorm.DB, constructor func(tx *gorm.DB) R) UoW[R] {
	return &uowGorm[R]{db, constructor}
}

type uowGorm[R any] struct {
	db          *gorm.DB
	constructor func(tx *gorm.DB) R
}

func (u *uowGorm[R]) Do(ctx context.Context, fn func(context.Context, R) error, opts ...*TxOptions) error {
	txOpts := ConvertGormTxOptions(opts...)

	tx := u.db.WithContext(ctx).Begin(txOpts)
	if tx.Error != nil {
		return tx.Error
	}

	defer rollbackOnPanic(func() error { return tx.Rollback().Error })

	if err := fn(ctx, u.constructor(tx)); err != nil {
		rollbackError := tx.Rollback().Error
		if rollbackError != nil {
			return errors.Join(err, rollbackError)
		}

		return err
	}

	return tx.Commit().Error
}

func (u *uowGorm[R]) Begin(ctx context.Context, opts ...*TxOptions) (c context.Context, r R, err error) { //nolint
	txOpts := ConvertGormTxOptions(opts...)

	tx := u.db.WithContext(ctx).Begin(txOpts)
	if tx.Error != nil {
		return c, r, tx.Error
	}

	return context.WithValue(ctx, CtxTxKey{}, tx), u.constructor(tx), nil
}

func (u *uowGorm[R]) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Commit().Error
}

func (u *uowGorm[R]) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Rollback().Error
}

func (u *uowGorm[R]) SavePoint(ctx context.Context, name string) error {
	tx, ok := ctx.Value(CtxTxKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return ErrTransactionNotFound
	}

	return tx.Exec(savepointStatement(name)).Error
}

func ConvertGormTxOptions(opts ...*TxOptions) *sql.TxOptions {
	return ConvertSqlTxOptions(opts...)
}
