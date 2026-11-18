package uow

import (
	"context"
	"fmt"
	"strconv"
)

// RF -> Repository
type UoW[R any] interface {
	Do(ctx context.Context, fn func(ctx context.Context, repo R) error, opts ...*TxOptions) error
	Begin(ctx context.Context, opts ...*TxOptions) (context.Context, R, error)
	SavePoint(ctx context.Context, name string) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// IsolationLevel is the transaction isolation level used in [TxOptions].
type IsolationLevel int

// Various isolation levels that drivers may support in [DB.BeginTx].
// If a driver does not support a given isolation level an error may be returned.
//
// See https://en.wikipedia.org/wiki/Isolation_(database_systems)#Isolation_levels.
const (
	LevelDefault IsolationLevel = iota
	LevelReadUncommitted
	LevelReadCommitted
	LevelWriteCommitted
	LevelRepeatableRead
	LevelSnapshot
	LevelSerializable
	LevelLinearizable
)

// String returns the name of the transaction isolation level.
func (i IsolationLevel) String() string {
	switch i {
	case LevelDefault:
		return "Default"
	case LevelReadUncommitted:
		return "Read Uncommitted"
	case LevelReadCommitted:
		return "Read Committed"
	case LevelWriteCommitted:
		return "Write Committed"
	case LevelRepeatableRead:
		return "Repeatable Read"
	case LevelSnapshot:
		return "Snapshot"
	case LevelSerializable:
		return "Serializable"
	case LevelLinearizable:
		return "Linearizable"
	default:
		return "IsolationLevel(" + strconv.Itoa(int(i)) + ")"
	}
}

var _ fmt.Stringer = LevelDefault

// TxOptions holds the transaction options to be used in [DB.BeginTx].
type TxOptions struct {
	// Isolation is the transaction isolation level.
	// If zero, the driver or database's default level is used.
	Isolation IsolationLevel
	ReadOnly  bool
}
