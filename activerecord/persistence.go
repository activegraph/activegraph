package activerecord

import (
	"context"
)

type Persistence interface {
	Insert(ctx context.Context) (*ActiveRecord, error)
	Update(ctx context.Context) (*ActiveRecord, error)
	Delete(ctx context.Context) error
	IsPersisted() bool
}

type InsertOperation struct {
	TableName      string
	Values         map[string]interface{}
	OnDuplicate    string
	ConflictTarget string
}

type DeleteOperation struct {
	TableName  string
	PrimaryKey string
	Value      interface{}
}

type Statements interface {
	BuildInsertStmt(op *InsertOperation) string
	BuildDeleteStmt(op *DeleteOperation) string
}

type Conn interface {
	//BeginTransaction(ctx context.Context) error
	//CommitTransaction(ctx context.Context) error
	//RollbackTransaction(ctx context.Context) error

	Statements

	ExecInsert(ctx context.Context, sql string, args ...interface{}) error
	// ExecDelete(ctx context.Context, sql string, args ...interface{}) error
	// ExecQuery(ctx context.Context, sql string, args ...interface{}) error
}
