package activerecord

import (
	"context"
)

type Persistence interface {
	Insert() (*ActiveRecord, error)
	Update() (*ActiveRecord, error)
	Delete() error
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

type QueryOperation struct {
	TableName  string
	Columns    []string
	Values     map[string]interface{}
	Predicates []string
	Args       []interface{}
}

type Conn interface {
	//BeginTransaction(ctx context.Context) error
	//CommitTransaction(ctx context.Context) error
	//RollbackTransaction(ctx context.Context) error

	ExecInsert(ctx context.Context, op *InsertOperation) (id interface{}, err error)
	ExecDelete(ctx context.Context, op *DeleteOperation) (err error)
	ExecQuery(ctx context.Context, op *QueryOperation) (rows []map[string]interface{}, err error)
}
