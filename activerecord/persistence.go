package activerecord

import (
	"context"
	"fmt"

	"github.com/activegraph/activegraph/activesupport"
)

type ErrTableNotFound struct {
	TableName string
}

func (e ErrTableNotFound) Error() string {
	return fmt.Sprintf("ErrTableNotFound: '%s'", e.TableName)
}

type Persistence interface {
	Insert() (*ActiveRecord, error)
	Update() (*ActiveRecord, error)
	Delete() error
	IsPersisted() bool
}

type InsertOperation struct {
	TableName      string
	ColumnValues   []ColumnValue
	OnDuplicate    string
	ConflictTarget string
}

type DeleteOperation struct {
	TableName  string
	PrimaryKey string
	Value      interface{}
}

type Dependency struct {
	TableName  string
	ForeignKey string
	PrimaryKey string
}

type QueryOperation struct {
	Text    string
	Args    []interface{}
	Columns []string
}

type ColumnValue struct {
	Name  string
	Type  Type
	Value interface{}
}

type ColumnDefinition struct {
	Name         string
	Type         Type
	NotNull      bool
	IsPrimaryKey bool
}

type Conn interface {
	BeginTransaction(ctx context.Context) (Conn, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error

	Exec(ctx context.Context, query string, args ...interface{}) error
	ExecInsert(ctx context.Context, op *InsertOperation) (id interface{}, err error)
	ExecDelete(ctx context.Context, op *DeleteOperation) (err error)
	ExecQuery(ctx context.Context, op *QueryOperation, cb func(activesupport.Hash) bool) (err error)

	ColumnDefinitions(ctx context.Context, tableName string) ([]ColumnDefinition, error)
	LookupType(typeName string) (Type, error)

	Close() error
}

type errConn struct {
	err error
}

func (c *errConn) BeginTransaction(ctx context.Context) (Conn, error) {
	return nil, c.err
}

func (c *errConn) CommitTransaction(ctx context.Context) error {
	return c.err
}

func (c *errConn) RollbackTransaction(ctx context.Context) error {
	return c.err
}

func (c *errConn) Exec(context.Context, string, ...interface{}) error {
	return c.err
}

func (c *errConn) ExecInsert(context.Context, *InsertOperation) (interface{}, error) {
	return nil, c.err
}

func (c *errConn) ExecDelete(context.Context, *DeleteOperation) error {
	return c.err
}

func (c *errConn) ExecQuery(context.Context, *QueryOperation, func(activesupport.Hash) bool) error {
	return c.err
}

func (c *errConn) LookupType(typeName string) (Type, error) {
	return nil, c.err
}

func (c *errConn) ColumnDefinitions(ctx context.Context, tableName string) ([]ColumnDefinition, error) {
	return nil, c.err
}

func (c *errConn) Close() error {
	return c.err
}
